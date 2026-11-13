#!/bin/sh
# guard-coverage.sh — the guard's tests must not cover less than they did.
#
#   scripts/guard-coverage.sh
#
# The guard decides what the assistant may run, so a rule no test runs can break in the
# next change without anything failing. This runs the tests of internal/guard with
# coverage and fails when the share of statements they run falls below the floor in
# internal/guard/coverage-floor. When a change raises it, raise the floor with it.
# Measured on Linux: CI runs it there only, since a few lines are for one system alone.
set -eu
cd "$(git rev-parse --show-toplevel)"
floor=$(cat internal/guard/coverage-floor)
out=$(go test -count=1 -cover ./internal/guard 2>&1) || { printf '%s\n' "$out"; exit 1; }
got=$(printf '%s\n' "$out" | sed -n 's/.*coverage: \([0-9.]*\)% of statements.*/\1/p')
if [ -z "$got" ]; then
  printf 'guard-coverage: no coverage in the output:\n%s\n' "$out"
  exit 1
fi
if awk -v g="$got" -v f="$floor" 'BEGIN { exit !(g + 0 < f + 0) }'; then
  echo "guard-coverage: the guard's tests cover $got% of it, below the floor of $floor%."
  echo "Add a test for the rule the change left uncovered."
  exit 1
fi
echo "guard-coverage: $got% (floor $floor%)"
