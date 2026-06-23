#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# AI Code CLI Router — Installateur macOS / Linux (bash)
# Usage : bash install.sh [dossier-installation]
# Default : ~/.local/share/multiai
# Author : Laurent Rochetta — https://follow.ovh/bio/laurent — https://rochetta.fr
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${1:-$HOME/.local/share/multiai}"

# ── Couleurs ──────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${CYAN}$*${NC}"; }
ok()    { echo -e "${GREEN}$*${NC}"; }
warn()  { echo -e "${YELLOW}$*${NC}"; }
error() { echo -e "${RED}$*${NC}" >&2; }

# ── Si pwsh est disponible, deléguer a install.ps1 ───────────────────────────
if command -v pwsh &>/dev/null; then
    info "pwsh detecte — utilisation de install.ps1..."
    exec pwsh -NoProfile -File "$SCRIPT_DIR/install.ps1" -InstallDir "$INSTALL_DIR"
fi

# ── pwsh absent : installation bash native ────────────────────────────────────
warn "pwsh (PowerShell Core) non detecte."
warn "Le routeur en necessite pour fonctionner."
echo ""
info "Installation de pwsh :"
if [[ "$(uname)" == "Darwin" ]]; then
    echo "  brew install powershell/tap/powershell"
    echo "  # ou : https://learn.microsoft.com/powershell/scripting/install/installing-powershell-on-macos"
else
    echo "  # Ubuntu / Debian :"
    echo "  sudo apt-get update && sudo apt-get install -y powershell"
    echo "  # ou : https://learn.microsoft.com/powershell/scripting/install/installing-powershell-on-linux"
fi
echo ""

# On continue quand meme l'installation des fichiers pour que tout soit pret
# quand l'utilisateur installe pwsh.
warn "Installation des fichiers sans pwsh — le routeur ne fonctionnera qu'apres avoir installe pwsh."
echo ""

# ── Copie des fichiers ────────────────────────────────────────────────────────
info "Installation dans : $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

while IFS= read -r -d '' file; do
    relative="${file#$SCRIPT_DIR/}"
    # Ignorer les .zip
    [[ "$relative" == *.zip ]] && continue

    dest="$INSTALL_DIR/$relative"
    dest_dir="$(dirname "$dest")"
    mkdir -p "$dest_dir"

    # Preserver les .env existants
    if [[ "$relative" == configs/profiles/*.env ]] && [[ -f "$dest" ]]; then
        cp "$file" "${dest}.new"
        warn "Profil existant preserve : $relative ; nouvelle version copiee en .new"
    else
        cp "$file" "$dest"
    fi
done < <(find "$SCRIPT_DIR" -type f -print0)

# ── Permissions ───────────────────────────────────────────────────────────────
find "$INSTALL_DIR" -name "*.sh" -type f -exec chmod +x {} \;

# Normaliser les fins de ligne CRLF->LF dans les .sh (si crees sous Windows)
if command -v sed &>/dev/null; then
    find "$INSTALL_DIR" -name "*.sh" -type f -exec sed -i 's/\r$//' {} \;
fi

ok "Fichiers installes."

# ── PATH ─────────────────────────────────────────────────────────────────────
detect_shell_profile() {
    local shell_name
    shell_name="$(basename "${SHELL:-bash}")"
    case "$shell_name" in
        zsh)  echo "$HOME/.zshrc" ;;
        bash)
            if [[ "$(uname)" == "Darwin" ]] && [[ -f "$HOME/.bash_profile" ]]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        *)    echo "$HOME/.profile" ;;
    esac
}

SHELL_PROFILE="$(detect_shell_profile)"
EXPORT_LINE="export PATH=\"\$PATH:$INSTALL_DIR\""

if grep -qF "$INSTALL_DIR" "$SHELL_PROFILE" 2>/dev/null; then
    ok "Deja present dans le PATH ($SHELL_PROFILE)."
else
    {
        echo ""
        echo "# AI Code CLI Router"
        echo "$EXPORT_LINE"
    } >> "$SHELL_PROFILE"
    ok "Ajoute au PATH dans : $SHELL_PROFILE"
fi

# ── Post-install ──────────────────────────────────────────────────────────────
RELOAD_CMD="source $SHELL_PROFILE"

echo ""
ok "Installation terminee."
echo ""
if command -v pwsh &>/dev/null; then
    info "Recharge ton shell puis utilise :"
else
    info "Apres avoir installe pwsh, recharge ton shell puis utilise :"
fi
echo "  $RELOAD_CMD"
echo ""
echo "  multiai.sh     # menu principal"
echo "  multiai.sh -Configure     # configurer les cles API"
echo "  multiai.sh -Bmad       # installer BMAD+ dans un projet"
echo ""
info "Configure tes cles (premiere etape recommandee) :"
echo "  bash $INSTALL_DIR/multiai.sh -Configure"
echo ""
info "Profils dans :"
echo "  $INSTALL_DIR/configs/profiles/"
