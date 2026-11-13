#!/bin/sh
# rehearse.sh — the rehearsal before a release, in the VMs, from this checkout.
#
#   test/rehearsal/rehearse.sh [--vm ubuntu|windows|both] [--from <snapshot>] [--update | --update-from <version>] [--sessions]
#
# It builds the release from this checkout (scripts/release-build.sh; the plugin travels
# inside the program), serves it over HTTP on 127.0.0.1 (a VM reaches this computer's
# 127.0.0.1 at 10.0.2.2), and for each VM: restores the snapshot (`clean` by default: a
# machine that never saw augmented brain; name another one, say with the last release
# installed, to rehearse an update), starts it, installs or updates with the published
# installer pointed at that release, runs checks.py there, and switches the VM off. The
# logs go to test/rehearsal/logs/ (not committed). --update first installs the release
# published last, from GitHub, and runs its doctor, so the release of this checkout goes in
# as an update: the same as --from a snapshot with the last release, without keeping one.
# --update-from <version> does the same from a release you name (say 0.23.0-beta): once
# this checkout is published, the release published last is this same version.
# On Ubuntu it then runs the smoke test with a real systemd (smoke.py): remote install,
# a reboot of the VM, the unit back with its PATH and permission mode, and session start.
# --sessions also asks a real Claude a couple of things: the VM's Claude must be logged in (in a snapshot you keep for that).
#
# The VMs are the ones of test/vm/ubuntu-desktop and test/vm/windows-11, made once with
# vagrant up (see their Vagrantfile). Their Vagrant state lives in the main copy of the
# repository, which is where this looks for it. Both take ports 2222 and 53389 on this
# computer, and their `clean` snapshot is a saved state, where Vagrant cannot move them:
# rehearse one only while the other is off.
set -eu
top=$(git rev-parse --show-toplevel)
cd "$top"
main=$(cd "$(git rev-parse --git-common-dir)/.." && pwd)
vms=both
from=clean
sessions=""
update=""
oldversion=""
while [ $# -gt 0 ]; do
  case "$1" in
    --vm) vms=$2; shift 2 ;;
    --from) from=$2; shift 2 ;;
    --sessions) sessions=--sessions; shift ;;
    --update) update=1; shift ;;
    --update-from) update=1; oldversion=$2; shift 2 ;;
    *) echo "rehearse: unknown option $1"; exit 2 ;;
  esac
done
case "$vms" in both) vms="ubuntu windows" ;; ubuntu|windows) ;; *) echo "rehearse: --vm ubuntu, windows or both"; exit 2 ;; esac

version="$(cat VERSION)-rehearsal"
work=$(mktemp -d)
server=""
trap 'kill "$server" 2>/dev/null || true; rm -rf "$work"' EXIT
logs="$top/test/rehearsal/logs"
mkdir -p "$logs"

echo "rehearse: building $version…"
scripts/release-build.sh "$version" "$work/release" >/dev/null
cp install.sh install.ps1 "$work/release/"
cp test/rehearsal/checks.py test/rehearsal/smoke.py "$work/release/"
port=$(python3 -I -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
(cd "$work/release" && exec python3 -I -m http.server --bind 127.0.0.1 "$port" >/dev/null 2>&1) &
server=$!
url="http://10.0.2.2:$port"

status=0
for vm in $vms; do
  dir="$main/test/vm/ubuntu-desktop"; name=augmented-brain-ubuntu
  [ "$vm" = windows ] && { dir="$main/test/vm/windows-11"; name=augmented-brain-windows; }
  log="$logs/$vm-$(date +%Y%m%d-%H%M).log"
  echo "rehearse: $vm, from snapshot $from${update:+, updating ${oldversion:-the release published last}} (log: $log)"
  VBoxManage snapshot "$name" restore "$from" >/dev/null
  VAGRANT_CWD="$dir" vagrant up --no-provision >/dev/null
  code=0
  if [ "$vm" = ubuntu ]; then
    VAGRANT_CWD="$dir" vagrant ssh -c "set -e
      export PATH=\$HOME/.local/bin:\$PATH AB_RELEASE_URL=$url
      rm -rf ~/ab-release && mkdir -p ~/ab-release
      curl -fsS $url/checks.py -o ~/ab-release/checks.py
      curl -fsS $url/smoke.py -o ~/ab-release/smoke.py
      if [ -n '$update' ]; then
        echo '##### the release published last'
        curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | env -u AB_RELEASE_URL AB_VERSION='${oldversion:-latest}' AB_YES=1 AB_NO_DESKTOP=1 sh
        augmented-brain --version; augmented-brain doctor || true
        echo '##### updating'
      fi
      curl -fsSL $url/install.sh | AB_YES=1 AB_NO_DESKTOP=1 sh
      python3 ~/ab-release/checks.py $version $sessions" >"$log" 2>&1 || code=$?
    # The smoke test with a real systemd: remote install with a pretend claude, a
    # reboot, and the unit must come back with its PATH and permission mode; then
    # session start must run a session for a task (smoke.py).
    smoke="export PATH=\$HOME/.local/bin:\$PATH; python3 ~/ab-release/smoke.py"
    VAGRANT_CWD="$dir" vagrant ssh -c "$smoke before" >>"$log" 2>&1 || code=$?
    VAGRANT_CWD="$dir" vagrant reload --no-provision >/dev/null 2>&1 || code=$?
    VAGRANT_CWD="$dir" vagrant ssh -c "$smoke after" >>"$log" 2>&1 || code=$?
  else
    VAGRANT_CWD="$dir" vagrant winrm -s powershell -c "
      \$ProgressPreference = 'SilentlyContinue'
      \$env:AB_RELEASE_URL = '$url'; \$env:AB_YES = '1'; \$env:AB_NO_DESKTOP = '1'
      \$r = \"\$HOME\\ab-release\"; if (Test-Path \$r) { Remove-Item -Recurse -Force \$r }
      New-Item -ItemType Directory \$r | Out-Null
      Invoke-WebRequest -UseBasicParsing '$url/checks.py' -OutFile \"\$r\\checks.py\"
      if ('$update') {
        Write-Output '##### the release published last'
        \$keep = \$env:AB_RELEASE_URL; \$env:AB_RELEASE_URL = \$null; \$env:AB_VERSION = '${oldversion:-latest}'
        irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
        \$env:AB_RELEASE_URL = \$keep; \$env:AB_VERSION = \$null
        \$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')
        augmented-brain --version; augmented-brain doctor
        Write-Output '##### updating'
      }
      irm '$url/install.ps1' | iex
      \$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')
      \$py = \$false; try { & python --version 2>\$null | Out-Null; \$py = (\$LASTEXITCODE -eq 0) } catch {}
      if (-not \$py) {
        # A clean Windows has no Python, which the checks are written in: only a
        # python.exe that sends you to the Microsoft Store, hence trying it, not finding it.
        Invoke-WebRequest -UseBasicParsing https://www.python.org/ftp/python/3.12.10/python-3.12.10-amd64.exe -OutFile \"\$r\\py.exe\"
        Start-Process \"\$r\\py.exe\" -ArgumentList '/quiet','InstallAllUsers=1','PrependPath=1' -Wait
        \$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')
      }
      \$env:PYTHONIOENCODING = 'utf-8'
      python \"\$r\\checks.py\" $version $sessions
      exit \$LASTEXITCODE" >"$log" 2>&1 || code=$?
  fi
  grep -E '^(BAD|#####)|ok, [0-9]+ BAD' "$log" || true
  [ "$code" = 0 ] || { echo "rehearse: $vm FAILED (exit $code): see $log"; status=1; }
  VAGRANT_CWD="$dir" vagrant halt >/dev/null || true
done
[ "$status" = 0 ] && echo "rehearse: all good."
exit "$status"
