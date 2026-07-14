#!/usr/bin/env bash
# multiai -- Cross-platform installer for Linux/macOS.
# Usage:
#   bash ./install.sh
#   bash ./install.sh -v 0.5.0
#   MULTIAI_SKIP_CHECKSUM=1 bash ./install.sh
#
# Downloads the release archive from GitHub Releases, verifies SHA256, and
# installs the binary to ~/.local/bin (or $MULTIAI_INSTALL_DIR).
#
# Environment variables:
#   MULTIAI_VERSION       — Pin a specific version (default: latest from GitHub)
#   MULTIAI_INSTALL_DIR   — Install directory  (default: $HOME/.local/bin)
#   MULTIAI_SKIP_CHECKSUM — Set to 1 to skip SHA256 verification (not recommended)
#
# Archive naming convention (must match .goreleaser.yaml):
#   multiai_<version>_<os>_<arch>.tar.gz

set -euo pipefail

# ── Configuration defaults ──────────────────────────────────────────────────
REPO="lrochetta/multiai"
VERSION="${MULTIAI_VERSION:-}"
INSTALL_DIR="${MULTIAI_INSTALL_DIR:-${HOME}/.local/bin}"
SKIP_CHECKSUM="${MULTIAI_SKIP_CHECKSUM:-0}"
BINARY="multiai"

# ── Color output (disabled when not a terminal or NO_COLOR is set) ──────────
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED=''; GREEN=''; YELLOW=''; CYAN=''; BOLD=''; NC=''
fi

info()  { printf "${CYAN}%s${NC}\n" "$*"; }
ok()    { printf "${GREEN}%s${NC}\n" "$*"; }
warn()  { printf "${YELLOW}%s${NC}\n" "$*" >&2; }
err()   { printf "${RED}%s${NC}\n" "$*" >&2; }

# ── Cleanup handler ─────────────────────────────────────────────────────────
TMPDIR=""
cleanup() {
    if [ -n "${TMPDIR}" ] && [ -d "${TMPDIR}" ]; then
        rm -rf "${TMPDIR}"
    fi
}
trap cleanup EXIT INT TERM

# ── Die helper ──────────────────────────────────────────────────────────────
die() {
    err "$*"
    exit 1
}

# ── Detect platform ─────────────────────────────────────────────────────────
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Darwin)  os="darwin" ;;
        Linux)   os="linux"  ;;
        *)       die "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)          arch="amd64" ;;
        arm64|aarch64)         arch="arm64" ;;
        *)                     die "Unsupported architecture: $(uname -m). Supported: amd64, arm64." ;;
    esac

    echo "${os}_${arch}"
}

# ── Find a command among alternatives ───────────────────────────────────────
find_cmd() {
    local cmd1="$1" cmd2="$2"
    if command -v "${cmd1}" >/dev/null 2>&1; then
        echo "${cmd1}"
    elif [ -n "${cmd2}" ] && command -v "${cmd2}" >/dev/null 2>&1; then
        echo "${cmd2}"
    else
        return 1
    fi
}

# ── Detect available download tool ──────────────────────────────────────────
detect_downloader() {
    find_cmd "curl" "wget" || die "Neither curl nor wget found.
  Install one of them (e.g. apt install curl, brew install curl) and retry."
}

# ── Detect checksum tool ────────────────────────────────────────────────────
detect_checksum_tool() {
    find_cmd "sha256sum" "shasum" || return 1
}

# ── Download helper ─────────────────────────────────────────────────────────
download() {
    local url="$1" dest="$2" tool
    tool=$(detect_downloader)
    if [ "${tool}" = "curl" ]; then
        curl -fsSL "${url}" -o "${dest}" || return 1
    else
        wget -q "${url}" -O "${dest}" || return 1
    fi
}

# ── Resolve latest version via GitHub API ───────────────────────────────────
resolve_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local dl
    dl=$(detect_downloader)

    if [ "${dl}" = "curl" ]; then
        curl -fsSL "${api_url}" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "v//;s/".*//' || return 1
    else
        wget -q -O- "${api_url}" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "v//;s/".*//' || return 1
    fi
}

# ── Resolve latest version via redirect (fallback if API fails) ────────────
resolve_latest_version_redirect() {
    local latest_url dl
    dl=$(detect_downloader)

    if [ "${dl}" = "curl" ]; then
        latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" 2>/dev/null)" || return 1
        echo "${latest_url##*/v}"
    else
        # wget fallback: parse the redirect location header
        latest_url="$(wget --spider --max-redirect=0 "https://github.com/${REPO}/releases/latest" 2>&1 | grep '^Location:' | tail -1 | sed 's/^Location: //;s/\r//;s/^[[:space:]]*//')" || return 1
        echo "${latest_url##*/v}"
    fi
}

# ── Main installation logic ─────────────────────────────────────────────────
main() {
    local platform os arch
    platform=$(detect_platform)
    os="${platform%_*}"
    arch="${platform#*_}"

    info "multiai installer -- ${BOLD}${os}/${arch}${NC}"

    # ── Resolve version ──────────────────────────────────────────────────
    if [ -z "${VERSION}" ]; then
        info "Resolving latest version from GitHub..."
        VERSION=$(resolve_latest_version || resolve_latest_version_redirect || true)
        if [ -z "${VERSION}" ]; then
            die "Cannot determine the latest version.
  • Set MULTIAI_VERSION=x.y.z to pin a specific version.
  • Check your internet connection.
  • Ensure github.com is reachable."
        fi
        ok "Latest version: ${VERSION} (v${VERSION#v})"
    else
        info "Using specified version: ${VERSION}"
    fi

    # Strip leading "v" if present
    VERSION="${VERSION#v}"

    local archive="multiai_${VERSION}_${os}_${arch}.tar.gz"
    local base_url="https://github.com/${REPO}/releases/download/v${VERSION}"

    # ── Create temp directory ────────────────────────────────────────────
    TMPDIR="$(mktemp -d 2>/dev/null || mktemp -d -t multiai-install.XXXXXXXXXX)"

    # ── Download archive ─────────────────────────────────────────────────
    info "Downloading ${archive}..."
    download "${base_url}/${archive}" "${TMPDIR}/${archive}" || \
        die "Failed to download ${archive}
  Check that version v${VERSION} exists at:
  ${base_url}/${archive}"

    # ── Download checksums ───────────────────────────────────────────────
    info "Downloading checksums.txt..."
    download "${base_url}/checksums.txt" "${TMPDIR}/checksums.txt" || \
        warn "checksums.txt not found — ${BOLD}SHA256 verification skipped${NC}"

    # ── Verify SHA256 ────────────────────────────────────────────────────
    if [ "${SKIP_CHECKSUM}" != "1" ] && [ -f "${TMPDIR}/checksums.txt" ]; then
        info "Verifying SHA256 checksum..."
        local ck_tool
        ck_tool=$(detect_checksum_tool) || die "No checksum verification tool found.
  Install coreutils (sha256sum) or Perl (shasum).
  To skip verification: set MULTIAI_SKIP_CHECKSUM=1 (not recommended)."

        (
            cd "${TMPDIR}"
            if [ "${ck_tool}" = "sha256sum" ]; then
                grep " ${archive}\$" checksums.txt | sha256sum --check --status 2>/dev/null
            else
                grep " ${archive}\$" checksums.txt | shasum -a 256 --check --status 2>/dev/null
            fi
        ) || die "SHA256 mismatch!
  The downloaded file does NOT match the expected checksum.
  This could mean:
    • The download was corrupted (retry with MULTIAI_VERSION=${VERSION}).
    • The release was tampered with.
  If you trust the source, set MULTIAI_SKIP_CHECKSUM=1 to skip verification."

        ok "SHA256 verified against checksums.txt"

    elif [ "${SKIP_CHECKSUM}" = "1" ]; then
        warn "SKIP_CHECKSUM is set — skipping SHA256 verification (not recommended)"
    fi

    # ── Extract archive ──────────────────────────────────────────────────
    info "Extracting archive..."
    if command -v tar >/dev/null 2>&1; then
        tar xzf "${TMPDIR}/${archive}" -C "${TMPDIR}" || \
            die "Failed to extract ${archive}. Is the file a valid tar.gz?"
    else
        die "tar is required to extract the archive but was not found."
    fi

    # ── Install binary ───────────────────────────────────────────────────
    if [ ! -f "${TMPDIR}/${BINARY}" ]; then
        die "Binary '${BINARY}' not found in the extracted archive.
  The archive may be corrupted or the release format has changed."
    fi

    mkdir -p "${INSTALL_DIR}" || die "Cannot create install directory: ${INSTALL_DIR}"
    cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}" || \
        die "Cannot copy binary to ${INSTALL_DIR}/${BINARY}
  Check that you have write permission to ${INSTALL_DIR}."
    chmod +x "${INSTALL_DIR}/${BINARY}" || \
        warn "Could not set executable permission on ${INSTALL_DIR}/${BINARY}"

    ok "multiai ${VERSION} installed: ${INSTALL_DIR}/${BINARY}"

    # ── PATH check ───────────────────────────────────────────────────────
    if ! echo ":${PATH-}:" | grep -q ":${INSTALL_DIR}:"; then
        warn "${INSTALL_DIR} is not in your PATH"
        printf '\n'
        printf 'Add the following line to your shell profile (~/.bashrc, ~/.zshrc, ~/.profile):\n\n'
        printf "  ${BOLD}export PATH=\"%s:\$PATH\"${NC}\n\n" "${INSTALL_DIR}"
        printf 'Then reload with:\n'
        printf "  ${BOLD}source ~/.bashrc${NC}  (or restart your terminal)\n"
        printf '\n'
    fi

    # ── Verify installation ──────────────────────────────────────────────
    if [ -x "${INSTALL_DIR}/${BINARY}" ]; then
        info "Verifying installation..."
        "${INSTALL_DIR}/${BINARY}" version 2>/dev/null || \
            warn "Binary installed but '${BINARY} version' failed to run."
    fi

    echo ""
    ok "Installation complete! Run:  ${BOLD}${BINARY} help${NC}"
}

main "$@"
