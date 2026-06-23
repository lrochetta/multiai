#!/usr/bin/env bash
# AI Code CLI Router — Lance le menu BMAD+ (équivalent de bmad.cmd sur Windows)
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec pwsh -NoProfile -File "$DIR/code-router.ps1" -Bmad "$@"
