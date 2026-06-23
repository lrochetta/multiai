#!/usr/bin/env bash
# AI Code CLI Router — Wrapper Unix
# Usage : multiai [args...]
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec pwsh -NoProfile -File "$DIR/code-router.ps1" "$@"
