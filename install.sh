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
#   NO_COLOR       Set to any non-empty value to disable styled output
#                  (also respects TERM=dumb and non-TTY stderr)

set -eu

REPO="${REPO:-captured-ventures/rememberize-cli}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="rememberize"

# -----------------------------------------------------------------------------
# Output styling — pure ANSI, no deps. Aesthetic mirrors the Go CLI's
# lipgloss tables (rounded borders, dim color-8 borders, subdued accents)
# so the install ceremony feels continuous with the post-install experience.
#
# Opt-outs (any one suffices):
#   stderr is not a TTY (piped to file)
#   $NO_COLOR is non-empty (https://no-color.org POSIX standard)
#   $TERM is "dumb"
# -----------------------------------------------------------------------------
USE_STYLE=1
if [ ! -t 2 ] || [ -n "${NO_COLOR:-}" ] || [ "${TERM:-}" = "dumb" ]; then
  USE_STYLE=0
fi

if [ "$USE_STYLE" = "1" ]; then
  C_RESET="$(printf '\033[0m')"
  C_BOLD="$(printf '\033[1m')"
  C_DIM="$(printf '\033[2m')"
  C_GRAY="$(printf '\033[38;5;8m')"
  C_GREEN="$(printf '\033[32m')"
  C_RED="$(printf '\033[31m')"
  C_CYAN="$(printf '\033[36m')"
  GLYPH_OK="✓"
  GLYPH_ERR="✗"
  GLYPH_STEP="›"
  BOX_TL="╭"; BOX_TR="╮"; BOX_BL="╰"; BOX_BR="╯"; BOX_H="─"; BOX_V="│"
else
  C_RESET=""; C_BOLD=""; C_DIM=""; C_GRAY=""; C_GREEN=""; C_RED=""; C_CYAN=""
  GLYPH_OK="[ok]"
  GLYPH_ERR="[err]"
  GLYPH_STEP=">"
  BOX_TL="+"; BOX_TR="+"; BOX_BL="+"; BOX_BR="+"; BOX_H="-"; BOX_V="|"
fi

# header_box prints a rounded title panel like the Go CLI's table headers.
header_box() {
  title="$1"
  # Width = title length + 4 (2-space pad each side). Add 2 for borders.
  pad=4
  title_len=${#title}
  inner=$((title_len + pad))
  bar=""
  i=0
  while [ "$i" -lt "$inner" ]; do
    bar="${bar}${BOX_H}"
    i=$((i + 1))
  done
  printf '\n%s%s%s%s%s\n' "$C_GRAY" "$BOX_TL" "$bar" "$BOX_TR" "$C_RESET" >&2
  printf '%s%s%s  %s%s%s  %s%s%s\n' "$C_GRAY" "$BOX_V" "$C_RESET" "$C_BOLD" "$title" "$C_RESET" "$C_GRAY" "$BOX_V" "$C_RESET" >&2
  printf '%s%s%s%s%s\n\n' "$C_GRAY" "$BOX_BL" "$bar" "$BOX_BR" "$C_RESET" >&2
}

step()    { printf '  %s%s%s %s\n' "$C_DIM" "$GLYPH_STEP" "$C_RESET" "$1" >&2; }
ok()      { printf '  %s%s%s %s\n' "$C_GREEN" "$GLYPH_OK" "$C_RESET" "$1" >&2; }
warn()    { printf '  %s%s%s %s\n' "$C_DIM" "!" "$C_RESET" "$1" >&2; }
err()     { printf '  %s%s%s %s%s%s\n' "$C_RED" "$GLYPH_ERR" "$C_RESET" "$C_BOLD" "$1" "$C_RESET" >&2; }
section() { printf '\n  %s%s%s\n' "$C_BOLD" "$1" "$C_RESET" >&2; }
cmd()     { printf '    %s%s%s\n' "$C_CYAN" "$1" "$C_RESET" >&2; }
hint()    { printf '    %s%s%s\n' "$C_DIM" "$1" "$C_RESET" >&2; }

die() {
  err "$*"
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

header_box "Installing rememberize CLI"

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
ok "Detected ${OS}/${ARCH}"

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
  step "Resolving latest release from github.com/${REPO}..."
  RELEASES_URL="https://api.github.com/repos/$REPO/releases/latest"
  TAG=$($FETCH "$RELEASES_URL" 2>/dev/null \
    | grep '"tag_name":' \
    | head -n 1 \
    | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/') || TAG=""
  if [ -z "$TAG" ]; then
    die "could not determine latest release tag (no releases published yet?)"
  fi
fi
ok "Version ${TAG}"

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

step "Downloading ${ARCHIVE}"
if ! $FETCH_OUT "$TMP_DIR/$ARCHIVE" "$URL"; then
  die "download failed: $URL"
fi
ok "Downloaded"

step "Extracting"
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
ok "Extracted"

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
ok "Installed to ${DEST}"

# Path warning if INSTALL_DIR not on PATH.
case ":${PATH:-}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) warn "${INSTALL_DIR} is not on your PATH" ;;
esac

# -----------------------------------------------------------------------------
# Next steps
# -----------------------------------------------------------------------------
section "Next steps"

case ":${PATH:-}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    hint "Add ${INSTALL_DIR} to your shell profile:"
    cmd "export PATH=\"\$HOME/.local/bin:\$PATH\""
    printf '\n' >&2
    ;;
esac

hint "Verify the install:"
cmd "rememberize --version"
printf '\n' >&2
hint "Pair this machine to your rememberize account:"
cmd "rememberize pair <code>"
hint "(get a code from https://rememberize.app/app/connections/new)"
printf '\n' >&2
