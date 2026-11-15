#!/bin/sh
# Runs install.sh with CIRCUIT_LOCAL_BINARY against the local build in a
# scratch HOME and checks the installed binary and the installer's errors.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/circuit-install-test.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT INT TERM

fail() { printf 'install_test: %s\n' "$1" >&2; exit 1; }

[ -x "$ROOT/circuit" ] || fail "run 'make build' first"

HOME="$SCRATCH/home" CIRCUIT_LOCAL_BINARY="$ROOT/circuit" sh "$ROOT/install.sh" >"$SCRATCH/out" 2>&1 \
  || { cat "$SCRATCH/out" >&2; fail "install.sh failed"; }
[ -x "$SCRATCH/home/.local/bin/circuit" ] || fail "binary not installed to ~/.local/bin"
"$SCRATCH/home/.local/bin/circuit" version | grep -q '^circuit ' || fail "installed binary does not report a version"

# Re-running is safe.
HOME="$SCRATCH/home" CIRCUIT_LOCAL_BINARY="$ROOT/circuit" sh "$ROOT/install.sh" >/dev/null 2>&1 \
  || fail "second install.sh run failed"

: >"$SCRATCH/empty"
if HOME="$SCRATCH/home2" CIRCUIT_LOCAL_BINARY="$SCRATCH/empty" sh "$ROOT/install.sh" >/dev/null 2>&1; then
  fail "install.sh accepted an empty CIRCUIT_LOCAL_BINARY"
fi

# A binary that does not run must not replace the working one.
printf '#!/bin/sh\nexit 1\n' >"$SCRATCH/broken"
if HOME="$SCRATCH/home" CIRCUIT_LOCAL_BINARY="$SCRATCH/broken" sh "$ROOT/install.sh" >/dev/null 2>&1; then
  fail "install.sh accepted a binary that does not run"
fi
"$SCRATCH/home/.local/bin/circuit" version | grep -q '^circuit ' || fail "broken binary replaced the installed one"
for leftover in "$SCRATCH"/home/.local/bin/circuit.tmp.*; do
  if [ -e "$leftover" ]; then fail "install.sh left a temp file behind: $leftover"; fi
done

set +e
HOME="$SCRATCH/home3" sh "$ROOT/install.sh" --bogus >/dev/null 2>&1
status=$?
set -e
[ "$status" -eq 2 ] || fail "unknown option exited $status, want 2"

printf 'install_test: ok\n'
