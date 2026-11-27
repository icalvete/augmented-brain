#!/bin/sh
# release-build.sh — builds the release archives for every supported system.
#
#   scripts/release-build.sh 0.1.0 [output_dir]
#
# Writes augmented-brain_<version>_<os>_<arch>.tar.gz for Linux and macOS and
# augmented-brain_<version>_windows_<arch>.zip (with augmented-brain.exe) for Windows,
# amd64 and arm64 each, augmented-brain-telegram_<version>_<os>_<arch>.tgz for Linux and
# macOS, copies install.sh and install.ps1 next to them, and writes
# checksums.txt with the SHA-256 of all of them in sha256sum's format, into output_dir
# (default: dist/).
# Binaries are static (CGO off) and stripped (-s -w); the version is set at build time.
# AB_RELEASE_TARGETS ("darwin/arm64 …") builds only those: the install checks need just
# this machine's, and building all six took a slow Intel Mac most of CI's 10 minutes.
# AB_RELEASE_PROGRAMS ("augmented-brain augmented-brain-telegram", the default) builds
# only those programs; empty, none. AB_RELEASE_CHECKSUMS=0 leaves out the installers and
# checksums.txt. Telegram's program takes minutes for each system on CI's runners: CI and
# the release build it in jobs of its own, side by side, and a last run with
# AB_RELEASE_PROGRAMS="" over all the archives writes checksums.txt.
#
# The same commit and the same Go give the same files, byte for byte, wherever they are
# built: no paths (-trimpath), no git stamp (-buildvcs=false: it would depend on the
# tags the clone has), and archives with fixed dates, owners and order. The date is the
# commit's (SOURCE_DATE_EPOCH, when set, wins). With GNU tar and zip (Linux, CI); other
# tars still build working archives, only not reproducible ones.
set -eu

VERSION="${1:?usage: $0 <version> [output_dir]}"
VERSION="${VERSION#v}"
OUT="${2:-dist}"
TARGETS="${AB_RELEASE_TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"
PROGRAMS="${AB_RELEASE_PROGRAMS-augmented-brain augmented-brain-telegram}"
has() { case " $PROGRAMS " in *" $1 "*) return 0 ;; esac; return 1; }

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

EPOCH="${SOURCE_DATE_EPOCH:-$(git -C "$ROOT" log -1 --format=%ct 2>/dev/null || date +%s)}"
export TZ=UTC   # zip keeps local times

# stamp gives every file and folder under $1 the date $EPOCH.
stamp() {
  if touch -d "@$EPOCH" "$1" 2>/dev/null; then
    find "$1" -exec touch -h -d "@$EPOCH" {} +
  else
    t="$(date -u -r "$EPOCH" +%Y%m%d%H%M.%S)"
    find "$1" -exec touch -h -t "$t" {} +
  fi
}

for t in $TARGETS; do
  os="${t%/*}"
  arch="${t#*/}"
  if has augmented-brain; then
    name="augmented-brain_${VERSION}_${os}_${arch}"
    exe=augmented-brain
    [ "$os" = windows ] && exe=augmented-brain.exe
    mkdir -p "$WORK/$name"
    (cd "$ROOT" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X github.com/icalvete/augmented-brain/internal/version.Version=$VERSION" \
      -o "$WORK/$name/$exe" ./cmd/augmented-brain)
    cp "$ROOT/LICENSE" "$ROOT/THIRD_PARTY_NOTICES.md" "$ROOT/README.md" "$WORK/$name/"
    chmod 755 "$WORK/$name" "$WORK/$name/$exe"
    chmod 644 "$WORK/$name/LICENSE" "$WORK/$name/THIRD_PARTY_NOTICES.md" "$WORK/$name/README.md"
    stamp "$WORK/$name"
    files="$(cd "$WORK" && find "$name" | LC_ALL=C sort)"
    if [ "$os" = windows ]; then
      # Windows opens a .zip by itself; install.ps1 unpacks it with Expand-Archive.
      # -X: no extra fields (owner ids, other dates); -D: no folder entries.
      rm -f "$OUT/$name.zip"
      # shellcheck disable=SC2086 # one name per line, none with spaces
      (cd "$WORK" && zip -qX -D "$OUT/$name.zip" $files)
      echo "built $name.zip"
    elif tar --version 2>/dev/null | grep -q 'GNU tar'; then
      tar -C "$WORK" --sort=name --owner=0 --group=0 --numeric-owner --format=gnu \
        --mtime="@$EPOCH" -cf - "$name" | gzip -9n > "$OUT/$name.tar.gz"
      echo "built $name.tar.gz"
    else
      tar -C "$WORK" -cf - "$name" | gzip -9n > "$OUT/$name.tar.gz"
      echo "built $name.tar.gz"
    fi
  fi
  # Telegram's program, on its own (its library triples a program's size): augmented
  # brain downloads it only when someone turns Telegram on. Linux and macOS: it keeps
  # its session in their keychains. .tgz, never .tar.gz: installers pick augmented-brain's
  # own archive by "_<os>_<arch>.tar.gz".
  has augmented-brain-telegram || continue
  [ "$os" = windows ] && continue
  tname="augmented-brain-telegram_${VERSION}_${os}_${arch}"
  mkdir -p "$WORK/$tname"
  (cd "$ROOT" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X github.com/icalvete/augmented-brain/internal/version.Version=$VERSION" \
    -o "$WORK/$tname/augmented-brain-telegram" ./cmd/augmented-brain-telegram)
  cp "$ROOT/LICENSE" "$ROOT/THIRD_PARTY_NOTICES.md" "$WORK/$tname/"
  chmod 755 "$WORK/$tname" "$WORK/$tname/augmented-brain-telegram"
  chmod 644 "$WORK/$tname/LICENSE" "$WORK/$tname/THIRD_PARTY_NOTICES.md"
  stamp "$WORK/$tname"
  if tar --version 2>/dev/null | grep -q 'GNU tar'; then
    tar -C "$WORK" --sort=name --owner=0 --group=0 --numeric-owner --format=gnu \
      --mtime="@$EPOCH" -cf - "$tname" | gzip -9n > "$OUT/$tname.tgz"
  else
    tar -C "$WORK" -cf - "$tname" | gzip -9n > "$OUT/$tname.tgz"
  fi
  echo "built $tname.tgz"
done

[ "${AB_RELEASE_CHECKSUMS:-1}" = 0 ] && exit 0

# The installers go in the release as they are in this commit, and in checksums.txt.
cp "$ROOT/install.sh" "$ROOT/install.ps1" "$OUT/"

cd "$OUT"
set --
for f in augmented-brain_"${VERSION}"_*.tar.gz augmented-brain_"${VERSION}"_*.zip augmented-brain-telegram_"${VERSION}"_*.tgz; do
  [ -e "$f" ] && set -- "$@" "$f"   # with AB_RELEASE_TARGETS, maybe no .zip at all
done
set -- "$@" install.sh install.ps1
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$@" > checksums.txt
else
  shasum -a 256 "$@" > checksums.txt
fi
echo "wrote checksums.txt"
