#!/usr/bin/env bash
# Build a .deb package for multiai
set -euo pipefail

VERSION="${1:-0.5.0}"
ARCH="${2:-amd64}"
BINARY="build/multiai-linux-${ARCH}"
DEB_NAME="multiai_${VERSION}_${ARCH}.deb"
BUILD_DIR="build/deb"

echo "Building ${DEB_NAME}..."

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

# Copy control
sed "s/Version: .*/Version: ${VERSION}/; s/Architecture: .*/Architecture: ${ARCH}/" packaging/deb/control > "${BUILD_DIR}/DEBIAN/control"
cp packaging/deb/postinst "${BUILD_DIR}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"

# Build
dpkg-deb --build "${BUILD_DIR}" "build/${DEB_NAME}"

echo "Package cree : build/${DEB_NAME}"
