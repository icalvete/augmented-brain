#!/bin/sh
# release-build.sh — builds the release archives for every supported system.
#
#   scripts/release-build.sh 0.1.0 [output_dir]
#
# Writes augmented-brain_<version>_<os>_<arch>.tar.gz for Linux and macOS (amd64 and
# arm64) and checksums.txt with their SHA-256, into output_dir (default: dist/).
# Binaries are static (CGO off) and stripped (-s -w); the version is set at build time.
set -eu

VERSION="${1:?usage: $0 <version> [output_dir]}"
VERSION="${VERSION#v}"
OUT="${2:-dist}"
TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

for t in $TARGETS; do
  os="${t%/*}"
  arch="${t#*/}"
  name="augmented-brain_${VERSION}_${os}_${arch}"
  mkdir -p "$WORK/$name"
  (cd "$ROOT" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/icalvete/augmented-brain/internal/version.Version=$VERSION" \
    -o "$WORK/$name/augmented-brain" ./cmd/augmented-brain)
  cp "$ROOT/LICENSE" "$ROOT/README.md" "$WORK/$name/"
  tar -C "$WORK" -czf "$OUT/$name.tar.gz" "$name"
  echo "built $name.tar.gz"
done

cd "$OUT"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum augmented-brain_"${VERSION}"_*.tar.gz > checksums.txt
else
  shasum -a 256 augmented-brain_"${VERSION}"_*.tar.gz > checksums.txt
fi
echo "wrote checksums.txt"
