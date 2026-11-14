#!/bin/sh
# release-checks.sh — whether every required check passed on the commit to publish.
#
#   GH_TOKEN=… scripts/release-checks.sh <owner/repo> <commit>
#
# The required checks are the ones main's ruleset names, plus the macOS ones (release_only
# below), which only a release waits for. Each must have passed: a check
# still running, missing, failed or cancelled stops the release. A merge commit of a
# pull request counts its pull request's last commit too when both hold the same files
# (main takes only pull requests up to date with it, so they do): a workflow that runs
# on main only for some files leaves its required check on the pull request alone.
# A check run again counts by its newest run. Needs git (the commit fetched) and gh.
set -eu

repo="${1:?usage: $0 <owner/repo> <commit>}"
commit="$(git rev-parse --verify "${2:?usage: $0 <owner/repo> <commit>}^{commit}")"
main="$(gh api "repos/$repo" --jq .default_branch)"

required="$(gh api "repos/$repo/rules/branches/$main" \
  --jq '.[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context' | sort -u)"
if [ -z "$required" ]; then
  echo "The ruleset of $main names no required checks: nothing to gate the release on."; exit 1
fi
# A release also needs the checks a pull request does not wait for: macOS is slow, so
# the ruleset leaves it out and a pull request merges without it, but CI still runs it
# on main after each merge and no version goes out until it has passed there.
release_only="test (macos-latest)
compat (macos-latest)"
required="$(printf '%s\n%s\n' "$required" "$release_only" | sort -u)"

shas="$commit"
pr="$(git rev-parse --verify --quiet "$commit^2" || true)"
if [ -n "$pr" ] && [ "$(git rev-parse "$commit^{tree}")" = "$(git rev-parse "$pr^{tree}")" ]; then
  shas="$shas $pr"
fi

passed=""
for s in $shas; do
  passed="$passed
$(gh api "repos/$repo/commits/$s/check-runs" --paginate --jq '.check_runs[] | [.id, .name, (.conclusion // .status)] | @tsv' |
    sort -n | awk -F'\t' '{last[$2]=$3} END {for (n in last) if (last[n]=="success") print n}')"
done

missing="$(echo "$required" | while IFS= read -r c; do
  echo "$passed" | grep -Fxq -- "$c" || echo "  $c"
done)"
if [ -n "$missing" ]; then
  echo "Not every required check has passed on $commit${pr:+ (or $pr)}:"
  echo "$missing"
  echo "Wait until CI is green there, then run the release again."
  exit 1
fi
echo "Every required check passed on $commit."
