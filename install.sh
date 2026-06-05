#!/bin/sh

set -eu

REPO="alexhughson/exa-cli"
BINARY_NAME="exa-cli"

usage() {
  cat <<'EOF'
Install exa-cli from GitHub Releases.

Usage:
  sh install.sh [--version vX.Y.Z] [--bin-dir DIR]

Options:
  --version   Install a specific release tag. Defaults to the latest release.
  --bin-dir   Install into this directory. Defaults to /usr/local/bin when writable,
              otherwise ~/.local/bin.
  --help      Show this help text.
EOF
}

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "error: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

download_to() {
  url=$1
  dest=$2

  if need_cmd curl; then
    curl -fsSL "$url" -o "$dest"
    return
  fi
  if need_cmd wget; then
    wget -qO "$dest" "$url"
    return
  fi

  fail "curl or wget is required"
}

fetch_text() {
  url=$1

  if need_cmd curl; then
    curl -fsSL "$url"
    return
  fi
  if need_cmd wget; then
    wget -qO- "$url"
    return
  fi

  fail "curl or wget is required"
}

resolve_version() {
  api_url="https://api.github.com/repos/${REPO}/releases/latest"
  json=$(fetch_text "$api_url")
  version=$(printf '%s' "$json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
  if [ -z "$version" ]; then
    fail "could not determine latest release from GitHub"
  fi
  printf '%s\n' "$version"
}

detect_os() {
  case $(uname -s) in
    Darwin) printf 'darwin\n' ;;
    Linux) printf 'linux\n' ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac
}

detect_arch() {
  case $(uname -m) in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

default_bin_dir() {
  if [ -w /usr/local/bin ]; then
    printf '/usr/local/bin\n'
  else
    printf '%s/.local/bin\n' "$HOME"
  fi
}

checksum_file() {
  file=$1

  if need_cmd sha256sum; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  if need_cmd shasum; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  if need_cmd openssl; then
    openssl dgst -sha256 "$file" | awk '{print $NF}'
    return
  fi

  return 1
}

verify_checksum() {
  asset_name=$1
  asset_path=$2
  checksums_path=$3

  expected=$(awk -v name="$asset_name" '$2 == name { print $1 }' "$checksums_path")
  if [ -z "$expected" ]; then
    fail "could not find checksum for $asset_name"
  fi

  if actual=$(checksum_file "$asset_path"); then
    if [ "$actual" != "$expected" ]; then
      fail "checksum mismatch for $asset_name"
    fi
    log "verified checksum for $asset_name"
    return
  fi

  log "warning: no SHA-256 tool found; skipping checksum verification"
}

extract_archive() {
  archive_path=$1
  output_dir=$2

  tar -xzf "$archive_path" -C "$output_dir"
}

VERSION=""
BIN_DIR=""

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      [ $# -ge 2 ] || fail "--version requires a value"
      VERSION=$2
      shift 2
      ;;
    --bin-dir)
      [ $# -ge 2 ] || fail "--bin-dir requires a value"
      BIN_DIR=$2
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

if [ -z "$VERSION" ]; then
  VERSION=$(resolve_version)
fi
if [ -z "$BIN_DIR" ]; then
  BIN_DIR=$(default_bin_dir)
fi

OS=$(detect_os)
ARCH=$(detect_arch)
ASSET_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
ASSET_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT INT TERM

ARCHIVE_PATH="${WORK_DIR}/${ASSET_NAME}"
CHECKSUMS_PATH="${WORK_DIR}/checksums.txt"
EXTRACT_DIR="${WORK_DIR}/extract"

mkdir -p "$EXTRACT_DIR"

log "installing ${BINARY_NAME} ${VERSION} for ${OS}/${ARCH}"
download_to "$ASSET_URL" "$ARCHIVE_PATH"
download_to "$CHECKSUMS_URL" "$CHECKSUMS_PATH"
verify_checksum "$ASSET_NAME" "$ARCHIVE_PATH" "$CHECKSUMS_PATH"
extract_archive "$ARCHIVE_PATH" "$EXTRACT_DIR"

BIN_PATH="${EXTRACT_DIR}/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}/${BINARY_NAME}"
[ -f "$BIN_PATH" ] || fail "binary not found in downloaded archive"

mkdir -p "$BIN_DIR"
install -m 0755 "$BIN_PATH" "${BIN_DIR}/${BINARY_NAME}"

log "installed ${BINARY_NAME} to ${BIN_DIR}/${BINARY_NAME}"

case ":$PATH:" in
  *":${BIN_DIR}:"*) ;;
  *)
    log "warning: ${BIN_DIR} is not on your PATH"
    ;;
esac
