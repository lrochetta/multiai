#!/usr/bin/env bash
# multiai universal installer for macOS/Linux.
# Usage: curl -fsSL https://rochetta.fr/multiai/install.sh | bash
#
# Downloads the release archive for this platform, VERIFIES its sha256
# against the release checksums.txt, then installs the binary.
# Archive names follow the goreleaser template:
#   multiai_<version>_<os>_<arch>.tar.gz
set -euo pipefail

REPO="lrochetta/multiai"
VERSION="${MULTIAI_VERSION:-}"
INSTALL_DIR="${MULTIAI_INSTALL_DIR:-$HOME/.local/bin}"
BINARY="multiai"

case "$(uname -s)" in
    Darwin)  PLATFORM="darwin";;
    Linux)   PLATFORM="linux";;
    *)       echo "Unsupported OS: $(uname -s)"; exit 1;;
esac

case "$(uname -m)" in
    x86_64|amd64)  ARCH="amd64";;
    arm64|aarch64) ARCH="arm64";;
    *)             echo "Unsupported arch: $(uname -m)"; exit 1;;
esac

# Resolve the latest version from the GitHub redirect when not pinned.
if [ -z "${VERSION}" ]; then
    LATEST_URL="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
    VERSION="${LATEST_URL##*/v}"
    if [ -z "${VERSION}" ] || [ "${VERSION}" = "${LATEST_URL}" ]; then
        echo "Cannot resolve the latest version. Set MULTIAI_VERSION=x.y.z and retry."
        exit 1
    fi
fi

ARCHIVE="multiai_${VERSION}_${PLATFORM}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

echo "multiai ${VERSION} -- ${PLATFORM}/${ARCH}"
echo "Installing to ${INSTALL_DIR}..."

mkdir -p "${INSTALL_DIR}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${TMPDIR}/${ARCHIVE}"
curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMPDIR}/checksums.txt"

# Verify sha256 before touching anything.
(
    cd "${TMPDIR}"
    if command -v sha256sum >/dev/null 2>&1; then
        grep " ${ARCHIVE}\$" checksums.txt | sha256sum --check --status
    else
        grep " ${ARCHIVE}\$" checksums.txt | shasum -a 256 --check --status
    fi
)
echo "sha256 OK (verified against checksums.txt)"

tar xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

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
