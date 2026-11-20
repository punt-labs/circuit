#!/bin/sh
# Tests local installation and download failures in scratch HOMEs.
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

offline_bin="$SCRATCH/offline-bin"
mkdir "$offline_bin"
for tool in awk mkdir mktemp rm; do
  ln -s "$(command -v "$tool")" "$offline_bin/$tool"
done
if command -v sha256sum >/dev/null 2>&1; then
  ln -s "$(command -v sha256sum)" "$offline_bin/sha256sum"
else
  ln -s "$(command -v shasum)" "$offline_bin/shasum"
fi

cat >"$offline_bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) printf 'Linux\n' ;;
  -m) printf 'x86_64\n' ;;
  *) exit 1 ;;
esac
EOF
cat >"$offline_bin/curl" <<'EOF'
#!/bin/sh
[ "$CIRCUIT_TEST_DOWNLOAD" = mismatch ] || exit 22
case "${4##*/}" in
  circuit-linux-amd64) printf 'test binary\n' >"$3" ;;
  checksums.txt) printf '%064d  circuit-linux-amd64\n' 0 >"$3" ;;
  *) exit 22 ;;
esac
EOF
chmod +x "$offline_bin/uname" "$offline_bin/curl"

check_download_failure() {
  scenario="$1"
  shift
  home="$SCRATCH/$scenario"
  if HOME="$home" PATH="$offline_bin" CIRCUIT_LOCAL_BINARY='' CIRCUIT_TEST_DOWNLOAD="$scenario" \
    /bin/sh "$ROOT/install.sh" >"$SCRATCH/out" 2>&1; then
    fail "$scenario unexpectedly succeeded"
  fi
  for message in "$@"; do
    grep -Fq "$message" "$SCRATCH/out" \
      || { cat "$SCRATCH/out" >&2; fail "$scenario did not report: $message"; }
  done
  [ ! -e "$home/.local/bin/circuit" ] || fail "$scenario left an installed binary"
}

check_download_failure mismatch "Checksum mismatch for circuit-linux-amd64"
check_download_failure unavailable "falling back to source build" "Go is not installed"

printf 'install_test: ok\n'
