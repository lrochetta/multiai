#!/usr/bin/env bash
# multiai universal installer for macOS/Linux
# Usage: curl -fsSL https://rochetta.fr/multiai/install.sh | bash
set -euo pipefail

VERSION="${MULTIAI_VERSION:-0.5.0}"
INSTALL_DIR="${MULTIAI_INSTALL_DIR:-$HOME/.local/bin}"
BINARY="multiai"

case "$(uname -s)" in
    Darwin)  PLATFORM="darwin";;
    Linux)   PLATFORM="linux";;
    *)       echo "Unsupported OS: $(uname -s)"; exit 1;;
esac

case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64";;
    arm64|aarch64) ARCH="arm64";;
    *)            echo "Unsupported arch: $(uname -m)"; exit 1;;
esac

TARGET="${PLATFORM}-${ARCH}"
ARCHIVE="multiai_${VERSION}_${TARGET}.tar.gz"
URL="https://github.com/lrochetta/multiai/releases/download/v${VERSION}/${ARCHIVE}"

echo "multiai ${VERSION} -- ${TARGET}"
echo "Installing to ${INSTALL_DIR}..."

mkdir -p "${INSTALL_DIR}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf ${TMPDIR}' EXIT

curl -fsSL "${URL}" | tar xz -C "${TMPDIR}"

if [ -f "${TMPDIR}/${BINARY}" ]; then
    cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    echo "multiai installed: ${INSTALL_DIR}/${BINARY}"
else
    echo "Binary not found in archive"
    exit 1
fi

# PATH check
if ! echo "${PATH}" | grep -q "${INSTALL_DIR}"; then
    echo ""
    echo "Add to your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    echo "Or add this line to ~/.bashrc / ~/.zshrc"
fi

"${INSTALL_DIR}/${BINARY}" version
echo "Run: ${INSTALL_DIR}/${BINARY} help"
