#!/usr/bin/env bash
# AI Code CLI Router — Configure les clés API (équivalent de config.cmd sur Windows)
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec pwsh -NoProfile -File "$DIR/code-router.ps1" -Configure "$@"
