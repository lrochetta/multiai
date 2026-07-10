#!/usr/bin/env bash
# ==========================================================================
# patch-manifest.sh — Inject checkver + autoupdate into the GoReleaser-generated
# Scoop manifest (dist/scoop/multiai.json).
#
# GoReleaser's built-in Scoop pipe does not emit checkver or autoupdate
# fields.  This script patches them in so Scoop can auto-update.
#
# Usage: ./packaging/scoop/patch-manifest.sh [path/to/manifest.json]
#   Default: dist/scoop/multiai.json (relative to the Go module root).
#
# Called by the release workflow after `goreleaser release`.
# ==========================================================================

set -euo pipefail

MANIFEST="${1:-dist/scoop/multiai.json}"

if [ ! -f "$MANIFEST" ]; then
  echo "❌ Manifest not found: $MANIFEST"
  exit 1
fi

# Use jq to inject checkver + autoupdate into the existing manifest.
# The archive URL pattern matches the GoReleaser name_template:
#   multiai_<version>_windows_amd64.zip  (version = tag, e.g. v0.5.0)
jq \
  --arg homepage "https://github.com/lrochetta/multiai" \
  '. + {
    "checkver": {
      "github": $homepage,
      "regex": "tag/([\\d.]+)"
    },
    "autoupdate": {
      "architecture": {
        "64bit": {
          "url": "https://github.com/lrochetta/multiai/releases/download/v$version/multiai_v$version_windows_amd64.zip",
          "hash": {
            "url": "https://github.com/lrochetta/multiai/releases/download/v$version/checksums.txt",
            "regex": "$sha256[\\s]+multiai_v$version_windows_amd64.zip"
          }
        }
      }
    }
  }' "$MANIFEST" > "${MANIFEST}.tmp" && mv "${MANIFEST}.tmp" "$MANIFEST"

echo "✅ Patched $MANIFEST with checkver + autoupdate"
