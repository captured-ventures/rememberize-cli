#!/bin/sh
# rememberize CLI installer
#
# Usage:
#   curl -sSL https://rememberize.app/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/captured-ventures/rememberize-cli/main/install.sh | sh
#
# Environment variables:
#   INSTALL_DIR    Where to install the binary (default: $HOME/.local/bin)
#   VERSION        Specific version/tag to install (default: latest)
#   REPO           Override repo slug (default: captured-ventures/rememberize-cli)

set -eu

REPO="${REPO:-captured-ventures/rememberize-cli}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="rememberize"

log() {
  printf '%s\n' "$*" >&2
}

die() {
  log "error: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need_cmd uname
need_cmd mkdir
need_cmd mv
need_cmd rm
need_cmd chmod

# -----------------------------------------------------------------------------
# Detect OS
# -----------------------------------------------------------------------------
detect_os() {
  os_raw=$(uname -s 2>/dev/null || echo unknown)
  case "$os_raw" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT) echo "windows" ;;
    *) die "unsupported OS: $os_raw" ;;
  esac
}

# -----------------------------------------------------------------------------
# Detect arch
# -----------------------------------------------------------------------------
detect_arch() {
  arch_raw=$(uname -m 2>/dev/null || echo unknown)
  case "$arch_raw" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) die "unsupported architecture: $arch_raw" ;;
  esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "windows" ] && [ "$ARCH" = "arm64" ]; then
  die "windows arm64 is not currently published"
fi

# -----------------------------------------------------------------------------
# Pick a downloader
# -----------------------------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
  FETCH_OUT="curl -fsSL -o"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
  FETCH_OUT="wget -qO"
else
  die "need curl or wget installed"
fi

# -----------------------------------------------------------------------------
# Resolve version
# -----------------------------------------------------------------------------
TAG="${VERSION:-}"
if [ -z "$TAG" ]; then
  log "resolving latest release from github.com/$REPO..."
  RELEASES_URL="https://api.github.com/repos/$REPO/releases/latest"
  # Extract "tag_name": "vX.Y.Z" without jq.
  TAG=$($FETCH "$RELEASES_URL" 2>/dev/null \
    | grep '"tag_name":' \
    | head -n 1 \
    | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/') || TAG=""
  if [ -z "$TAG" ]; then
    die "could not determine latest release tag (no releases published yet?)"
  fi
fi

# Strip leading v for archive name
VERSION_NUM=$(printf '%s' "$TAG" | sed 's/^v//')

# -----------------------------------------------------------------------------
# Build download URL
# -----------------------------------------------------------------------------
if [ "$OS" = "windows" ]; then
  EXT="zip"
else
  EXT="tar.gz"
fi

ARCHIVE="rememberize_${VERSION_NUM}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

# -----------------------------------------------------------------------------
# Download + extract
# -----------------------------------------------------------------------------
TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t rememberize)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

log "downloading $URL"
if ! $FETCH_OUT "$TMP_DIR/$ARCHIVE" "$URL"; then
  die "download failed: $URL"
fi

log "extracting $ARCHIVE"
case "$EXT" in
  tar.gz)
    need_cmd tar
    tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR" || die "extract failed"
    ;;
  zip)
    if command -v unzip >/dev/null 2>&1; then
      unzip -q "$TMP_DIR/$ARCHIVE" -d "$TMP_DIR" || die "extract failed"
    else
      die "unzip required to install on windows"
    fi
    ;;
esac

# Locate binary in extracted tree
BIN_SRC=""
for candidate in \
  "$TMP_DIR/$BIN_NAME" \
  "$TMP_DIR/$BIN_NAME.exe" \
  "$TMP_DIR/rememberize_${VERSION_NUM}_${OS}_${ARCH}/$BIN_NAME" \
  "$TMP_DIR/rememberize_${VERSION_NUM}_${OS}_${ARCH}/$BIN_NAME.exe"; do
  if [ -f "$candidate" ]; then
    BIN_SRC="$candidate"
    break
  fi
done

if [ -z "$BIN_SRC" ]; then
  die "could not find $BIN_NAME binary in extracted archive"
fi

# -----------------------------------------------------------------------------
# Install
# -----------------------------------------------------------------------------
mkdir -p "$INSTALL_DIR" || die "could not create $INSTALL_DIR"

DEST="$INSTALL_DIR/$BIN_NAME"
if [ "$OS" = "windows" ]; then
  DEST="$INSTALL_DIR/$BIN_NAME.exe"
fi

mv "$BIN_SRC" "$DEST" || die "could not move binary to $DEST"
chmod +x "$DEST" 2>/dev/null || true

# -----------------------------------------------------------------------------
# Report
# -----------------------------------------------------------------------------
cat >&2 <<EOF

Installed rememberize ${TAG} to ${DEST}

If ${INSTALL_DIR} is not on your PATH, add it to your shell profile:
  export PATH="\$HOME/.local/bin:\$PATH"

Verify the install:
  rememberize --version

Get started by pairing this machine to your rememberize account:
  rememberize pair <code>

EOF
