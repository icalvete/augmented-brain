#!/bin/sh
# privacy-check.sh — stops personal data and secrets from reaching this public repo.
#
#   scripts/privacy-check.sh --staged   # what is about to be committed (used by the pre-commit hook)
#   scripts/privacy-check.sh --all      # every tracked file (used in CI and before publishing)
#
# Three checks:
#   1. Secrets: high-confidence patterns (cloud keys, tokens, private keys).
#   2. Personal denylist: literal strings, case-insensitive, one per line, from the file in
#      $AB_DENYLIST (default: ~/.config/augmented-brain-dev/denylist). The denylist lives
#      OUTSIDE the repo on purpose: it contains the very data it protects. If it is missing,
#      this check is skipped with a warning (contributors and CI do not have it).
#   3. Commit identity: with --staged, the author name and email must not match the denylist.
#
# Exceptions: exact literal strings in $AB_ALLOW (default: next to the denylist, "allow") are
# removed before the denylist is applied, e.g. the repo URL or the maintainer's public identity.
#
# Exit code: 0 clean, 1 something found. Prints file:line, never the matched secret.
set -u

MODE="${1:---staged}"
DENYLIST="${AB_DENYLIST:-$HOME/.config/augmented-brain-dev/denylist}"
ALLOW="${AB_ALLOW:-$(dirname "$DENYLIST")/allow}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
found=0

# Allowed literals, without comments or blank lines.
if [ -f "$ALLOW" ]; then
  grep -v -e '^[[:space:]]*#' -e '^[[:space:]]*$' "$ALLOW" > "$TMP/allow"
else
  : > "$TMP/allow"
fi

# strip_allowed < in > out: removes every allowed literal (case-insensitive) from each line.
strip_allowed() {
  awk -v allowfile="$TMP/allow" '
    BEGIN { while ((getline a < allowfile) > 0) if (a != "") allowed[++n] = tolower(a) }
    {
      line = $0
      for (i = 1; i <= n; i++) {
        while ((p = index(tolower(line), allowed[i])) > 0)
          line = substr(line, 1, p - 1) substr(line, p + length(allowed[i]))
      }
      print line
    }'
}

# High-confidence secret patterns (extended regex).
cat > "$TMP/secrets" <<'EOF'
AKIA[0-9A-Z]{16}
ASIA[0-9A-Z]{16}
gh[pousr]_[A-Za-z0-9]{36}
github_pat_[A-Za-z0-9_]{22,}
glpat-[A-Za-z0-9_-]{20}
xox[abprs]-[A-Za-z0-9-]{10,}
sk-ant-[A-Za-z0-9_-]{20,}
AIza[0-9A-Za-z_-]{35}
[0-9]{8,10}:AA[A-Za-z0-9_-]{33}
BEGIN [A-Z ]*PRIVATE KEY
EOF

# Denylist without comments or blank lines.
if [ -f "$DENYLIST" ]; then
  grep -v -e '^[[:space:]]*#' -e '^[[:space:]]*$' "$DENYLIST" > "$TMP/deny"
else
  echo "privacy-check: no denylist at $DENYLIST; checking secrets only." >&2
  : > "$TMP/deny"
fi

# Files to check, and how to read each one.
case "$MODE" in
  --staged) git diff --cached --name-only --diff-filter=ACMR > "$TMP/files"; READ='git show :' ;;
  --all)    git ls-files > "$TMP/files"; READ='cat ' ;;
  *) echo "usage: $0 [--staged|--all]" >&2; exit 2 ;;
esac

while IFS= read -r f; do
  [ -n "$f" ] || continue
  if [ "$READ" = 'git show :' ]; then git show ":$f" > "$TMP/blob" 2>/dev/null; else cat "$f" > "$TMP/blob"; fi
  # Secrets are searched in every file, binaries included (-a): the patterns are long.
  if grep -a -n -E -f "$TMP/secrets" "$TMP/blob" > "$TMP/hits" 2>/dev/null; then
    cut -d: -f1 "$TMP/hits" | sed "s|^|  SECRET  $f:|"
    found=1
  fi
  # The denylist has short words that show up by chance inside images and fonts, so it
  # only looks at text files (and at every file name).
  if [ -s "$TMP/deny" ]; then
    if printf '%s\n' "$f" | grep -q -i -F -f "$TMP/deny"; then
      echo "  PERSONAL  $f (file name)"; found=1
    fi
    strip_allowed < "$TMP/blob" > "$TMP/blob.allowed"
    if grep -I -n -i -F -f "$TMP/deny" "$TMP/blob.allowed" > "$TMP/hits" 2>/dev/null; then
      sed "s|^\([0-9]*\):.*|  PERSONAL  $f:\1|" "$TMP/hits"
      found=1
    fi
  fi
done < "$TMP/files"

if [ "$MODE" = "--staged" ] && [ -s "$TMP/deny" ]; then
  ident="$(git var GIT_AUTHOR_IDENT 2>/dev/null) $(git var GIT_COMMITTER_IDENT 2>/dev/null)"
  if printf '%s\n' "$ident" | strip_allowed | grep -q -i -F -f "$TMP/deny"; then
    echo "  PERSONAL  commit identity (author or committer name/email); set a public one with: git config user.name / user.email"
    found=1
  fi
fi

if [ "$found" -ne 0 ]; then
  echo "privacy-check: blocked. Remove the data above (or fix the identity) and try again." >&2
  exit 1
fi
echo "privacy-check: clean."
