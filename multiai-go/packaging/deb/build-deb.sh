#!/usr/bin/env bash
# LOCAL FALLBACK ONLY — the official .deb packages are built by GoReleaser
# (nfpms section of multiai-go/.goreleaser.yaml) and attached to each GitHub
# release with their sha256 in checksums.txt. Use this script only for local
# testing without a release.
#
# Usage: packaging/deb/build-deb.sh <version> [amd64|arm64]
# Expects the binary at build/multiai-linux-<arch> (see scripts/setup-go.sh).
set -euo pipefail

VERSION="${1:?usage: $0 <version> [amd64|arm64]}"
ARCH="${2:-amd64}"
BINARY="build/multiai-linux-${ARCH}"
DEB_NAME="multiai_${VERSION}_${ARCH}.deb"
BUILD_DIR="build/deb"

if [ ! -f "${BINARY}" ]; then
    echo "Binary ${BINARY} not found. Build it first:"
    echo "  CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -trimpath -ldflags '-s -w -X main.version=${VERSION}' -o ${BINARY} ./cmd/multiai/"
    exit 1
fi

echo "Building ${DEB_NAME} (local fallback build)..."

# Clean
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/usr/share/bash-completion/completions"
mkdir -p "${BUILD_DIR}/usr/share/zsh/site-functions"
mkdir -p "${BUILD_DIR}/usr/share/fish/completions"
mkdir -p "${BUILD_DIR}/usr/share/doc/multiai"

# Copy binary
cp "${BINARY}" "${BUILD_DIR}/usr/bin/multiai"
chmod 755 "${BUILD_DIR}/usr/bin/multiai"

# Copy control (Version/Architecture are stamped here)
sed "s/Version: .*/Version: ${VERSION}/; s/Architecture: .*/Architecture: ${ARCH}/" packaging/deb/control > "${BUILD_DIR}/DEBIAN/control"
cp packaging/deb/postinst "${BUILD_DIR}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"

# Build
dpkg-deb --build "${BUILD_DIR}" "build/${DEB_NAME}"

echo "Package cree : build/${DEB_NAME}"
