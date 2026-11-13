#!/bin/sh
# release-build.sh — builds the release archives for every supported system.
#
#   scripts/release-build.sh 0.1.0 [output_dir]
#
# Writes augmented-brain_<version>_<os>_<arch>.tar.gz for Linux and macOS and
# augmented-brain_<version>_windows_<arch>.zip (with augmented-brain.exe) for Windows,
# amd64 and arm64 each, copies install.sh and install.ps1 next to them, and writes
# checksums.txt with the SHA-256 of all of them in sha256sum's format, into output_dir
# (default: dist/).
# Binaries are static (CGO off) and stripped (-s -w); the version is set at build time.
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
TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

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
done

# The installers go in the release as they are in this commit, and in checksums.txt.
cp "$ROOT/install.sh" "$ROOT/install.ps1" "$OUT/"

cd "$OUT"
set -- augmented-brain_"${VERSION}"_*.tar.gz augmented-brain_"${VERSION}"_*.zip install.sh install.ps1
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$@" > checksums.txt
else
  shasum -a 256 "$@" > checksums.txt
fi
echo "wrote checksums.txt"
