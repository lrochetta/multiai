#!/usr/bin/env bash
# Generate APT repository structure from .deb files.
#
# Usage:
#   ./scripts/generate-apt-repo.sh \
#     --debs <deb-dir> \
#     --repo <repo-dir> \
#     --suite <suite-name> \
#     --gpg-key <key-id>
#
# Required tools:
#   - dpkg-dev (dpkg-scanpackages)
#   - apt-utils (apt-ftparchive)
#   - gnupg    (gpg)
#
# Example:
#   ./scripts/generate-apt-repo.sh \
#     --debs ./dist \
#     --repo ./apt \
#     --suite stable \
#     --gpg-key "laurent@rochetta.fr"
#
# Environment:
#   APT_GPG_PASSPHRASE  — GPG passphrase (default: empty for batch CI usage)
#   GPG_KEYRING         — Path to keyring file (default: agent's default keyring)

set -euo pipefail

# ── Parse arguments ───────────────────────────────────────────────────────────
DEBS_DIR=""
REPO_DIR=""
SUITE="stable"
GPG_KEY=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --debs)     DEBS_DIR="$2"; shift 2 ;;
        --repo)     REPO_DIR="$2"; shift 2 ;;
        --suite)    SUITE="$2";    shift 2 ;;
        --gpg-key)  GPG_KEY="$2";  shift 2 ;;
        --help|-h)  sed -n '/^# /,/^$/p' "$0" | head -n -1; exit 0 ;;
        *)          echo "Error: unknown argument $1"; exit 1 ;;
    esac
done

if [ -z "$DEBS_DIR" ] || [ -z "$REPO_DIR" ]; then
    echo "Error: --debs and --repo are required"
    echo "Usage: $0 --debs <deb-dir> --repo <repo-dir> [--suite <name>] [--gpg-key <key-id>]"
    exit 1
fi

if [ ! -d "$DEBS_DIR" ]; then
    echo "Error: debs directory does not exist: $DEBS_DIR"
    exit 1
fi

# ── Configuration ─────────────────────────────────────────────────────────────
ORIG_DIR="$(pwd)"
# Resolve absolute paths
DEBS_DIR="$(cd "$DEBS_DIR" && pwd)"
REPO_DIR="$(cd "$(dirname "$REPO_DIR")" && pwd)/$(basename "$REPO_DIR")"

POOL_DIR="$REPO_DIR/pool/main/m/multiai"
DISTS_DIR="$REPO_DIR/dists/$SUITE"

# ── Helper functions ──────────────────────────────────────────────────────────
info()  { echo "   $*"; }
ok()    { echo "  [OK] $*"; }
err()   { echo " [ERR] $*" >&2; }

die() {
    err "$*"
    exit 1
}

check_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        die "Required tool '$1' not found. Install it with: sudo apt install $2"
    fi
}

# ── Preflight checks ──────────────────────────────────────────────────────────
check_tool "dpkg-scanpackages" "dpkg-dev"
check_tool "apt-ftparchive"    "apt-utils"
check_tool "gpg"               "gnupg"

# ── Step 1: Populate pool ─────────────────────────────────────────────────────
echo ""
info "Step 1/5 — Copying .deb files to pool..."
mkdir -p "$POOL_DIR"

deb_count=0
for deb in "$DEBS_DIR"/*.deb; do
    [ -f "$deb" ] || continue
    cp -v "$deb" "$POOL_DIR/"
    deb_count=$((deb_count + 1))
done

if [ "$deb_count" -eq 0 ]; then
    die "No .deb files found in $DEBS_DIR"
fi
ok "$deb_count .deb file(s) copied to $POOL_DIR"

# ── Step 2: Detect architectures ──────────────────────────────────────────────
echo ""
info "Step 2/5 — Detecting architectures..."
ARCHS=()
for deb in "$POOL_DIR"/*.deb; do
    [ -f "$deb" ] || continue
    arch=$(dpkg --info "$deb" 2>/dev/null | grep -E '^ Architecture:' | awk '{print $2}')
    if [ -n "$arch" ]; then
        found=0
        for a in "${ARCHS[@]}"; do
            [ "$a" = "$arch" ] && found=1 && break
        done
        [ "$found" -eq 0 ] && ARCHS+=("$arch")
    fi
done

if [ "${#ARCHS[@]}" -eq 0 ]; then
    die "Could not detect any architectures from .deb files"
fi

ok "Detected architectures: ${ARCHS[*]}"

# ── Step 3: Generate Packages files ───────────────────────────────────────────
echo ""
info "Step 3/5 — Generating Packages files..."
cd "$REPO_DIR"

for arch in "${ARCHS[@]}"; do
    arch_dir="$DISTS_DIR/main/binary-$arch"
    mkdir -p "$arch_dir"

    info "  Generating Packages (${arch})..."
    dpkg-scanpackages --arch "$arch" pool/ /dev/null 2>/dev/null > "$arch_dir/Packages" || \
        die "dpkg-scanpackages failed for arch $arch"

    info "  Compressing Packages (${arch})..."
    gzip -kf "$arch_dir/Packages" 2>/dev/null || \
        gzip -f "$arch_dir/Packages" 2>/dev/null

    pkg_count=$(wc -l < "$arch_dir/Packages" 2>/dev/null || echo 0)
    ok "binary-$arch: Packages + Packages.gz generated ($((pkg_count / 2)) packages)"
done

# ── Step 4: Generate Release file ─────────────────────────────────────────────
echo ""
info "Step 4/5 — Generating Release file..."

# Create apt-ftparchive config
APT_CONF=$(mktemp)
cat > "$APT_CONF" << APTCONF
APT::FTPArchive::Release {
  Origin "multiai";
  Label "multiai APT Repository";
  Suite "$SUITE";
  Codename "$SUITE";
  Architectures "${ARCHS[*]}";
  Components "main";
  Description "APT repository for multiai — Route multiple AI CLIs with isolated env profiles";
  Date "$(date -Ru 2>/dev/null || date)";
};
APTCONF

apt-ftparchive release -c "$APT_CONF" "$DISTS_DIR" > "$DISTS_DIR/Release" 2>/dev/null || \
    die "apt-ftparchive release failed"
rm -f "$APT_CONF"

ok "Release generated at $DISTS_DIR/Release"

# ── Step 5: Sign Release file ─────────────────────────────────────────────────
echo ""
info "Step 5/5 — Signing Release file..."

if [ -n "$GPG_KEY" ]; then
    GPG_OPTS=("--default-key" "$GPG_KEY" "--armor" "--batch" "--yes")
    if [ -n "${APT_GPG_PASSPHRASE:-}" ]; then
        GPG_OPTS+=("--passphrase" "$APT_GPG_PASSPHRASE")
    fi

    info "  Creating Release.gpg (detached signature)..."
    gpg "${GPG_OPTS[@]}" --detach-sign \
        -o "$DISTS_DIR/Release.gpg" \
        "$DISTS_DIR/Release" 2>/dev/null || \
        die "Failed to create Release.gpg"

    info "  Creating InRelease (clearsigned)..."
    gpg "${GPG_OPTS[@]}" --clearsign \
        -o "$DISTS_DIR/InRelease" \
        "$DISTS_DIR/Release" 2>/dev/null || \
        die "Failed to create InRelease"

    ok "Release signed with GPG key: $GPG_KEY"
else
    warn "No GPG key specified — skipping signing (InRelease and Release.gpg not created)"
    warn "Users will need to use '--allow-insecure=yes' to use this repository"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  APT repository generated                                      │"
echo "│                                                                 │"
echo "│  Repository root: $REPO_DIR"
echo "│  Suite:           $SUITE"
echo "│  Architectures:   ${ARCHS[*]}"
echo "│  Packages:        $deb_count .deb files"
echo "│                                                                 │"
if [ -n "$GPG_KEY" ]; then
echo "│  Signed with:     $GPG_KEY"
fi
echo "│                                                                 │"
echo "│  To publish, push $REPO_DIR to gh-pages branch:               │"
echo "│    cd $REPO_DIR && git add -A && git commit ...         │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""

cd "$ORIG_DIR"
