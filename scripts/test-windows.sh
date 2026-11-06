#!/bin/sh
# test-windows.sh — the tests on Windows, before a push, in the Windows VM.
#
#   scripts/test-windows.sh            # every package with tests
#   scripts/test-windows.sh ./internal/paths ./internal/guard
#
# What passes on Linux can fail on Windows (paths with \, no file modes, CRLF): CI was
# where we found out. This builds each package's tests for Windows here, with the
# repository's files next to them, serves them over HTTP on 127.0.0.1 (the VM reaches
# this computer's 127.0.0.1 at 10.0.2.2; uploading through WinRM is slow), starts the
# VM of test/vm/windows-11, runs each test binary in its package's folder there, says
# which failed, and switches the VM off again. No Go is needed in the VM. Needs
# vagrant, VirtualBox and the VM made once (see its Vagrantfile), and python3.
set -eu
top=$(git rev-parse --show-toplevel)
cd "$top"
# The VM's Vagrant state lives in the main copy of the repository (a worktree has none).
main=$(cd "$(git rev-parse --git-common-dir)/.." && pwd)
vm="$main/test/vm/windows-11"
[ -d "$vm/.vagrant" ] || { echo "test-windows: no Windows VM in $vm (vagrant up there first)"; exit 2; }

work=$(mktemp -d)
trap 'kill "$server" 2>/dev/null || true; rm -rf "$work"' EXIT
server=""

echo "test-windows: building the tests for Windows…"
pkgs=${*:-./...}
# shellcheck disable=SC2086
go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}} {{.Dir}}{{end}}' $pkgs | sed '/^$/d' >"$work/pkgs"
module=$(go list -m)
mkdir -p "$work/tree"
git ls-files -z | (cd "$top" && xargs -0 cp --parents -t "$work/tree")
while read -r ip dir; do
  rel=${dir#"$top"}; rel=${rel#/}
  name=$(echo "${ip#"$module"}" | tr / _ | sed 's/^_//')
  GOOS=windows GOARCH=amd64 go test -c -o "$work/tree/$rel/${name:-root}.test.exe" "$ip"
  printf '%s\t%s\n' "$rel" "${name:-root}.test.exe" >>"$work/tree/tests.tsv"
done <"$work/pkgs"
(cd "$work/tree" && python3 -I -c 'import os,sys,zipfile
z=zipfile.ZipFile("../ab.zip","w",zipfile.ZIP_DEFLATED)
for r,_,fs in os.walk("."):
    for f in fs: z.write(os.path.join(r,f))')
echo "test-windows: $(wc -l <"$work/pkgs") packages, $(du -h "$work/ab.zip" | cut -f1) to send."

port=$(python3 -I -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
(cd "$work" && exec python3 -I -m http.server --bind 127.0.0.1 "$port" >/dev/null 2>&1) &
server=$!

echo "test-windows: starting the VM…"
VAGRANT_CWD="$vm" vagrant up --no-provision >/dev/null

ps=$(cat <<EOF
\$ProgressPreference = 'SilentlyContinue'
\$d = 'C:\ab-test'
if (Test-Path \$d) { Remove-Item -Recurse -Force \$d }
New-Item -ItemType Directory \$d | Out-Null
Invoke-WebRequest -UseBasicParsing http://10.0.2.2:$port/ab.zip -OutFile "\$d.zip"
Expand-Archive "\$d.zip" \$d
\$failed = @()
foreach (\$line in Get-Content "\$d\tests.tsv") {
  \$rel, \$exe = \$line -split "\`t"
  Push-Location (Join-Path \$d \$rel)
  \$out = & ".\\\$exe" '-test.count=1' 2>&1 | Out-String
  if (\$LASTEXITCODE -ne 0) { \$failed += \$rel; Write-Output "MAL \$rel"; Write-Output \$out } else { Write-Output "ok  \$rel" }
  Pop-Location
}
Remove-Item -Recurse -Force \$d, "\$d.zip"
if (\$failed.Count -gt 0) { exit 1 }
EOF
)
code=0
VAGRANT_CWD="$vm" vagrant winrm -s powershell -c "$ps" || code=$?

echo "test-windows: switching the VM off…"
VAGRANT_CWD="$vm" vagrant halt >/dev/null || true
if [ "$code" = 0 ]; then echo "test-windows: all good on Windows."; else echo "test-windows: something fails on Windows (above)."; fi
exit "$code"
