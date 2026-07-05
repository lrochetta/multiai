# multiai

Route multiple AI CLIs (Claude Code, Codex CLI, OpenCode) through isolated
environment profiles. One command, 37+ provider profiles, API keys kept in
your OS credential store.

> **Status: pre-release (`0.4.0-dev`) — not published yet.**
>
> The `multiai` package currently on npm ships the PowerShell implementation
> (v0.3.x). *This* package is its next generation: it installs the native Go
> binary. It takes over the `multiai` name on npm only once the Go
> implementation reaches feature parity with v0.3.0, published manually by
> the maintainer. Until then, install the current version with
> `npm install -g multiai@0.3`.

## What `npm install -g multiai` does (this package)

1. Downloads the archive for your platform from the matching GitHub release
   (`windows/amd64`, `darwin/amd64+arm64`, `linux/amd64+arm64`).
2. **Verifies its SHA256** against the release `checksums.txt` — a mismatch
   aborts the install before anything is extracted.
3. Installs the binary inside the package and exposes it through a Node shim
   (`multiai` on your PATH).

The `checksums.txt` itself is signed with Cosign (keyless, GitHub Actions
OIDC). To verify the whole chain manually:

```sh
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/lrochetta/multiai' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

The npm installer does **not** run cosign for you (it only checks SHA256);
run the commands above if you want signature-level assurance.

## Environment variables

| Variable | Effect |
|---|---|
| `MULTIAI_SKIP_DOWNLOAD=1` | Skip the binary download (CI, offline). |
| `MULTIAI_INSTALL_DIR=path` | Also copy the verified binary to `path`. |

## Usage

```sh
multiai            # interactive menu
multiai launch -p ds
multiai list --json
multiai config
multiai help
```

Full documentation: https://github.com/lrochetta/multiai

## License

MIT
