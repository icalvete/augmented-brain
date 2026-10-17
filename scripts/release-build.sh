#!/bin/sh
# release-build.sh — builds the release archives for every supported system.
#
#   scripts/release-build.sh 0.1.0 [output_dir]
#
# Writes augmented-brain_<version>_<os>_<arch>.tar.gz for Linux and macOS and
# augmented-brain_<version>_windows_<arch>.zip (with augmented-brain.exe) for Windows,
# amd64 and arm64 each, and checksums.txt with their SHA-256 in sha256sum's format,
# into output_dir (default: dist/).
# Binaries are static (CGO off) and stripped (-s -w); the version is set at build time.
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

for t in $TARGETS; do
  os="${t%/*}"
  arch="${t#*/}"
  name="augmented-brain_${VERSION}_${os}_${arch}"
  exe=augmented-brain
  [ "$os" = windows ] && exe=augmented-brain.exe
  mkdir -p "$WORK/$name"
  (cd "$ROOT" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/icalvete/augmented-brain/internal/version.Version=$VERSION" \
    -o "$WORK/$name/$exe" ./cmd/augmented-brain)
  cp "$ROOT/LICENSE" "$ROOT/README.md" "$WORK/$name/"
  if [ "$os" = windows ]; then
    # Windows opens a .zip by itself; install.ps1 unpacks it with Expand-Archive.
    (cd "$WORK" && zip -qr "$OUT/$name.zip" "$name")
    echo "built $name.zip"
  else
    tar -C "$WORK" -czf "$OUT/$name.tar.gz" "$name"
    echo "built $name.tar.gz"
  fi
done

cd "$OUT"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum augmented-brain_"${VERSION}"_*.tar.gz augmented-brain_"${VERSION}"_*.zip > checksums.txt
else
  shasum -a 256 augmented-brain_"${VERSION}"_*.tar.gz augmented-brain_"${VERSION}"_*.zip > checksums.txt
fi
echo "wrote checksums.txt"
