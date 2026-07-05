#!/usr/bin/env bash
# Pin packaging/aur/PKGBUILD and .SRCINFO to the real sha256 of a released
# source tarball, so no SKIP/placeholder ever reaches the AUR.
#
# Usage: scripts/update-aur-checksums.sh <version>     (e.g. 0.4.0)
# Run AFTER the v<version> tag exists on GitHub.
set -euo pipefail

VERSION="${1:?usage: $0 <version> (e.g. 0.4.0)}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AUR_DIR="${ROOT}/packaging/aur"
URL="https://github.com/lrochetta/multiai/archive/refs/tags/v${VERSION}.tar.gz"

TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT

echo "Downloading ${URL}..."
curl -fsSL "${URL}" -o "${TMP}"
SHA="$(sha256sum "${TMP}" | awk '{print $1}')"
echo "sha256: ${SHA}"

sed -i -E "s/^pkgver=.*/pkgver=${VERSION}/" "${AUR_DIR}/PKGBUILD"
sed -i -E "s/^sha256sums=.*/sha256sums=('${SHA}')/" "${AUR_DIR}/PKGBUILD"

cat > "${AUR_DIR}/.SRCINFO" <<EOF
pkgbase = multiai
	pkgdesc = Route multiple AI CLIs (Claude Code, Codex, OpenCode) with isolated env profiles
	pkgver = ${VERSION}
	pkgrel = 1
	url = https://rochetta.fr
	arch = x86_64
	arch = aarch64
	license = MIT
	makedepends = go
	source = multiai-${VERSION}.tar.gz::${URL}
	sha256sums = ${SHA}

pkgname = multiai
EOF

echo "PKGBUILD and .SRCINFO pinned to v${VERSION} (${SHA})."
echo "Verify with: makepkg --verifysource -p ${AUR_DIR}/PKGBUILD"
