#!/bin/sh
# install.sh — Install the circuit CLI
# Usage: see README.md's Quick Start for the pinned-SHA install command.
set -eu

# --- Colors (disabled when not a terminal) ---
if [ -t 1 ]; then
  BOLD='\033[1m' GREEN='\033[32m' YELLOW='\033[33m' NC='\033[0m'
else
  BOLD='' GREEN='' YELLOW='' NC=''
fi

info() { printf '%b▶%b %s\n' "$BOLD" "$NC" "$1"; }
ok()   { printf '  %b✓%b %s\n' "$GREEN" "$NC" "$1"; }
warn() { printf '  %b!%b %s\n' "$YELLOW" "$NC" "$1" >&2; }
fail() { printf '  %b✗%b %s\n' "$YELLOW" "$NC" "$1" >&2; exit 1; }

VERSION="0.1.0"
REPO="punt-labs/circuit"
BINARY="circuit"

usage() {
  printf '%s\n' \
    'install.sh — install the circuit CLI to ~/.local/bin' \
    '' \
    'Usage: curl -fsSL .../install.sh | sh' \
    '' \
    'Options:' \
    '  -h, --help    Print this help and exit.' \
    '' \
    'Environment:' \
    '  CIRCUIT_LOCAL_BINARY=/path/to/binary' \
    '                Install a local binary instead of downloading' \
    '                (offline / air-gapped / testing).'
}

for arg in "$@"; do
  case "$arg" in
    -h|--help) usage; exit 0 ;;
    *)         printf 'install.sh: unknown option: %s\n' "$arg" >&2; usage >&2; exit 2 ;;
  esac
done

# --- Step 1: Prerequisites ---

info "Checking prerequisites..."

if command -v curl >/dev/null 2>&1; then
  ok "curl found"
else
  warn "curl not found — pre-built binary download will not work"
fi

if command -v sha256sum >/dev/null 2>&1; then
  SHA256="sha256sum"
  ok "sha256sum found"
elif command -v shasum >/dev/null 2>&1; then
  SHA256="shasum -a 256"
  ok "shasum found"
else
  SHA256=""
  warn "neither sha256sum nor shasum found — pre-built binary download will not work"
fi

if command -v go >/dev/null 2>&1; then
  GO_VERSION=$(go version | sed 's/.*go\([0-9]*\.[0-9]*\).*/\1/')
  ok "Go ${GO_VERSION} (fallback build)"
fi

if command -v git >/dev/null 2>&1; then
  ok "git found"
else
  warn "git not found — fallback build from source will not work"
fi

# --- Step 2: Install binary ---

info "Installing circuit binary..."

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

OS_RAW="$(uname -s)"
case "$OS_RAW" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)      OS="" ;;
esac
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       ARCH="" ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}"
RELEASE_URL="${CIRCUIT_RELEASE_URL:-https://github.com/${REPO}/releases/download/v${VERSION}}"
INSTALLED=0

TMPDIR_WORK="$(mktemp -d "${TMPDIR:-/tmp}/${BINARY}-install.XXXXXX")"
cleanup() { rm -rf "$TMPDIR_WORK"; }
trap cleanup EXIT INT TERM

# Copy a file into place atomically (temp file in the target dir, then mv).
# The copy must run "version" first, so a bad binary never replaces a
# working installation.
install_file() {
  TMPBIN="$(mktemp "${INSTALL_DIR}/${BINARY}.tmp.XXXXXX")"
  if ! cp "$1" "$TMPBIN" || ! chmod 0755 "$TMPBIN"; then
    rm -f "$TMPBIN"
    fail "failed to install $1 to ${INSTALL_DIR}/${BINARY}"
  fi
  if ! "$TMPBIN" version >/dev/null 2>&1; then
    rm -f "$TMPBIN"
    fail "$1 does not run 'version'; leaving ${INSTALL_DIR}/${BINARY} unchanged"
  fi
  mv "$TMPBIN" "${INSTALL_DIR}/${BINARY}"
}

if [ -n "${CIRCUIT_LOCAL_BINARY:-}" ]; then
  if [ ! -f "$CIRCUIT_LOCAL_BINARY" ] || [ ! -r "$CIRCUIT_LOCAL_BINARY" ]; then
    fail "CIRCUIT_LOCAL_BINARY is not a readable file: $CIRCUIT_LOCAL_BINARY"
  fi
  if [ ! -s "$CIRCUIT_LOCAL_BINARY" ]; then
    fail "CIRCUIT_LOCAL_BINARY is empty: $CIRCUIT_LOCAL_BINARY"
  fi
  install_file "$CIRCUIT_LOCAL_BINARY"
  ok "Installed local binary from $CIRCUIT_LOCAL_BINARY"
  INSTALLED=1
fi

# Download the pre-built binary and check it against the release checksums.
if [ "$INSTALLED" = "0" ] && [ -n "$OS" ] && [ -n "$ARCH" ] && [ -n "$SHA256" ] && command -v curl >/dev/null 2>&1; then
  if curl -fsSL -o "$TMPDIR_WORK/$ASSET" "$RELEASE_URL/$ASSET" \
    && curl -fsSL -o "$TMPDIR_WORK/checksums.txt" "$RELEASE_URL/checksums.txt"; then
    EXPECTED="$(awk -v f="$ASSET" '$2 == f { print $1 }' "$TMPDIR_WORK/checksums.txt")"
    ACTUAL="$($SHA256 "$TMPDIR_WORK/$ASSET" | awk '{ print $1 }')"
    if [ -z "$EXPECTED" ]; then
      fail "checksums.txt for v${VERSION} has no entry for $ASSET"
    fi
    if [ "$ACTUAL" != "$EXPECTED" ]; then
      fail "Checksum mismatch for $ASSET: expected $EXPECTED, got $ACTUAL"
    fi
    ok "SHA-256 verified for $ASSET"
    install_file "$TMPDIR_WORK/$ASSET"
    INSTALLED=1
  else
    warn "Download failed for $ASSET v${VERSION}, falling back to source build"
  fi
fi

# Fallback: build the tagged release from source.
if [ "$INSTALLED" = "0" ]; then
  if ! command -v go >/dev/null 2>&1; then
    fail "No pre-built binary (OS=${OS_RAW}, arch=$(uname -m)) and Go is not installed"
  fi
  if ! command -v git >/dev/null 2>&1; then
    fail "No pre-built binary and git is not installed for source build"
  fi
  warn "Pre-built binary not available, building from source..."
  if ! git clone --depth 1 --branch "v${VERSION}" "https://github.com/${REPO}.git" "$TMPDIR_WORK/src"; then
    fail "Tag v${VERSION} not found. This installer requires a tagged release."
  fi
  (cd "$TMPDIR_WORK/src" && CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o "$TMPDIR_WORK/$BINARY" ./cmd/circuit/) \
    || fail "go build failed"
  install_file "$TMPDIR_WORK/$BINARY"
fi

# --- Step 3: Verify ---

info "Verifying installation..."
INSTALLED_VERSION="$("$INSTALL_DIR/$BINARY" version)" || fail "$INSTALL_DIR/$BINARY version failed"
ok "$INSTALLED_VERSION"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    warn "$INSTALL_DIR is not on your PATH"
    # shellcheck disable=SC2016 # $PATH must stay literal in the instruction
    warn 'Add this to your shell profile: export PATH="$HOME/.local/bin:$PATH"'
    ;;
esac

printf '\n%b%b%s is ready!%b\n\n' "$GREEN" "$BOLD" "$BINARY" "$NC"
printf 'circuit reads machines from ./machines in the directory where you run it.\n'
printf 'Run "circuit --help" for the command list.\n\n'
