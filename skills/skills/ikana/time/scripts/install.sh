#!/usr/bin/env bash

set -euo pipefail

# Detect supported platform.
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *)
    echo "Unsupported OS: $OS. Supported OS values are: darwin, linux." >&2
    exit 1
    ;;
esac

case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64) ARCH="x64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

VERSION="${TEMPORAL_VERSION:-v0.1.0}"
FILE="temporal-${OS}-${ARCH}"

if [ "$VERSION" = "latest" ]; then
  RELEASE_URL="https://github.com/Ikana/temporal/releases/latest/download"
else
  RELEASE_URL="https://github.com/Ikana/temporal/releases/download/${VERSION}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "${RELEASE_URL}/${FILE}" -o "${TMP_DIR}/${FILE}"
curl -fsSL "${RELEASE_URL}/temporal-checksums.txt" -o "${TMP_DIR}/temporal-checksums.txt"

EXPECTED_SUM="$(
  awk -v file="$FILE" '
    $2 == file || $2 ~ "/" file "$" { print $1; exit }
  ' "${TMP_DIR}/temporal-checksums.txt"
)"
if [ -z "$EXPECTED_SUM" ]; then
  echo "Checksum entry not found for ${FILE} in temporal-checksums.txt" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL_SUM="$(sha256sum "${TMP_DIR}/${FILE}" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL_SUM="$(shasum -a 256 "${TMP_DIR}/${FILE}" | awk '{ print $1 }')"
else
  echo "No SHA-256 tool found (expected sha256sum or shasum)." >&2
  exit 1
fi

if [ "$ACTUAL_SUM" != "$EXPECTED_SUM" ]; then
  echo "Checksum mismatch for ${FILE}" >&2
  exit 1
fi

INSTALL_DIR="${TEMPORAL_INSTALL_DIR:-/usr/local/bin}"
if [ -z "${TEMPORAL_INSTALL_DIR:-}" ] && { [ ! -d "$INSTALL_DIR" ] || [ ! -w "$INSTALL_DIR" ]; }; then
  INSTALL_DIR="${HOME}/.local/bin"
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMP_DIR}/${FILE}" "${INSTALL_DIR}/temporal"
echo "Installed temporal ${VERSION} to ${INSTALL_DIR}/temporal"
