# Scoop bucket — multiai

The Scoop manifest for multiai is **generated at release time** by GoReleaser.
It is post-processed by `packaging/scoop/patch-manifest.sh` to inject
`checkver` and `autoupdate` fields required by Scoop's auto-update mechanism.

## How it works

1. On every `v*` tag push, the release workflow runs `goreleaser release`.
2. GoReleaser generates a base manifest into `dist/scoop/multiai.json` with:
   - Version, URL, and SHA256 hash of the Windows amd64 archive
   - Homepage, license, and description from `.goreleaser.yaml`
3. The workflow then runs `patch-manifest.sh` which injects:
   - `checkver` — points to `lrochetta/multiai` GitHub releases so Scoop
     knows when a new version is available
   - `autoupdate` — URL template and hash extraction from `checksums.txt`
     so `scoop update multiai` works without manual intervention

## Generated manifest location

```
multiai-go/dist/scoop/multiai.json
```

## Installation

Users install multiai via Scoop with:

```powershell
scoop bucket add multiai https://github.com/lrochetta/scoop-bucket
scoop install multiai
```

## Deployment

### Manual (first releases)

1. After a release run completes, download the generated `multiai.json` from
   the workflow's artifacts (or copy it from your local `dist/scoop/` after
   running `goreleaser release --clean --skip-upload`).
2. Push it to the `lrochetta/scoop-bucket` repository:
   ```powershell
   git clone https://github.com/lrochetta/scoop-bucket.git
   copy multiai.json scoop-bucket/
   cd scoop-bucket
   git add multiai.json
   git commit -m "multiai vX.Y.Z"
   git push
   ```

### Automated (requires PAT)

1. Create a classic GitHub PAT with `repo` scope from an account that has
   push access to `lrochetta/scoop-bucket`.
2. Add it as a repository secret named `TAP_GITHUB_TOKEN` in the multiai
   repository settings (Settings > Secrets and variables > Actions).
3. In `.goreleaser.yaml`, set `skip_upload: false` for the `scoops` section.
4. The next release will automatically push the generated (and patched)
   manifest to `lrochetta/scoop-bucket`.

## Checkver and autoupdate details

The patched manifest includes:

```json
{
  "checkver": {
    "github": "https://github.com/lrochetta/multiai",
    "regex": "tag/([\\\\d.]+)"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/lrochetta/multiai/releases/download/v$version/multiai_v$version_windows_amd64.zip",
        "hash": {
          "url": "https://github.com/lrochetta/multiai/releases/download/v$version/checksums.txt",
          "regex": "$sha256[\\\\s]+multiai_v$version_windows_amd64.zip"
        }
      }
    }
  }
}
```

- **checkver**: fetches the latest release tag from `lrochetta/multiai` via the
  GitHub API and extracts the version number.
- **autoupdate**: on `scoop update multiai`, Scoop replaces `$version` with
  the latest version and fetches both the binary archive and its SHA256 from
  `checksums.txt` for integrity verification.

## Archive naming

The Windows amd64 archive follows the GoReleaser `name_template`:

```
multiai_<version>_windows_amd64.zip
```

Example: `multiai_v0.5.0_windows_amd64.zip`

## 64-bit only

Only the `amd64` architecture is published. There are no 32-bit builds, and
`windows/arm64` is excluded from the build matrix (not a supported target).
