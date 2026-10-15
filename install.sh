#!/bin/sh
# install.sh — installs augmented brain on Linux or macOS. No root needed.
#
#   curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
#
# Downloads the archive for this system from the latest release, checks its SHA-256
# against the release's checksums.txt, and puts the program in ~/.local/bin (or
# $AB_INSTALL_DIR). Then run `augmented-brain init`.
#
# Settings, for testing: AB_VERSION (a version, default: latest) and AB_RELEASE_URL
# (where the release files are, default: the GitHub release).
set -eu

REPO="icalvete/augmented-brain"
VERSION="${AB_VERSION:-latest}"
DIR="${AB_INSTALL_DIR:-$HOME/.local/bin}"

fail() { echo "augmented brain: $*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "this system ($(uname -s)) is not supported yet: Linux and macOS only." ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "this processor ($(uname -m)) is not supported yet." ;;
esac

if [ -n "${AB_RELEASE_URL:-}" ]; then
  BASE="$AB_RELEASE_URL"
elif [ "$VERSION" = latest ]; then
  BASE="https://github.com/$REPO/releases/latest/download"
else
  BASE="https://github.com/$REPO/releases/download/v${VERSION#v}"
fi

fetch() { # fetch <url> <file>
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    fail "needs curl or wget to download."
  fi
}

if command -v sha256sum >/dev/null 2>&1; then
  sha() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  sha() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  fail "needs sha256sum or shasum to check the download."
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fetch "$BASE/checksums.txt" "$TMP/checksums.txt" || fail "cannot download $BASE/checksums.txt"
archive="$(grep "_${os}_${arch}\.tar\.gz\$" "$TMP/checksums.txt" | awk '{print $2}' | head -n 1)"
[ -n "$archive" ] || fail "the release has no archive for ${os}/${arch}."
fetch "$BASE/$archive" "$TMP/$archive" || fail "cannot download $archive"

want="$(grep " ${archive}\$" "$TMP/checksums.txt" | awk '{print $1}')"
got="$(sha "$TMP/$archive")"
[ -n "$want" ] && [ "$got" = "$want" ] || fail "the download does not match its checksum: not installed."

tar -xzf "$TMP/$archive" -C "$TMP"
mkdir -p "$DIR"
cp "$TMP/${archive%.tar.gz}/augmented-brain" "$DIR/augmented-brain.new"
chmod 755 "$DIR/augmented-brain.new"
mv "$DIR/augmented-brain.new" "$DIR/augmented-brain"

echo "augmented brain $("$DIR/augmented-brain" version) installed in $DIR"
case ":$PATH:" in
  *":$DIR:"*) ;;
  *) echo
     echo "$DIR is not in your PATH. Add this line to your shell profile (~/.bashrc, ~/.zshrc):"
     echo "  export PATH=\"$DIR:\$PATH\""
     echo "and open a new terminal." ;;
esac
echo
echo "Next: run  augmented-brain init"
