#!/bin/sh
# release-commit.sh — which commit of main a release is made from.
#
#   scripts/release-commit.sh [commit]     (default: HEAD)
#
# Prints the newest commit, walking main's own line (first parents: each pull request
# is one step), that set VERSION or changed that version's section of CHANGELOG.md or
# CHANGELOG.es.md. A pull request that merged later, with its line under Unreleased,
# is not part of this version: release.yml tags the commit printed here, not the tip
# of main at the moment someone clicks Run workflow. Git only, no network.
set -eu

head="$(git rev-parse --verify "${1:-HEAD}^{commit}")"
version="$(git show "$head:VERSION")"

# The lines under "## <version>" up to the next "## " heading, as in that commit.
section() {
  git show "$1:$2" 2>/dev/null | awk -v v="## $version" '$0==v{on=1;next} /^## /{on=0} on'
}

for c in $(git rev-list --first-parent --max-count=1000 "$head"); do
  p="$(git rev-parse --verify --quiet "$c^1" || true)"
  # The first commit of the repository, or the one that set this version.
  if [ -z "$p" ] || [ "$(git show "$p:VERSION" 2>/dev/null)" != "$version" ]; then
    echo "$c"; exit 0
  fi
  for f in CHANGELOG.md CHANGELOG.es.md; do
    if [ "$(section "$c" "$f")" != "$(section "$p" "$f")" ]; then
      echo "$c"; exit 0
    fi
  done
done
echo "release-commit.sh: VERSION $version was not set in the last 1000 commits" >&2
exit 1
