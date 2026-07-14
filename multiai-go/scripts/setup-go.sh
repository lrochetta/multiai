#!/usr/bin/env bash
# Setup script for multiai Go development (macOS/Linux)
set -euo pipefail

echo "multiai Go Setup"
echo "==============="
echo ""

# ── Install Go if missing ──────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
    echo "Go non trouve. Installation..."
    GO_VERSION="1.25.12"
    GO_LINUX_AMD64_SHA256="234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1"
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install go
    else
        archive="$(mktemp)"
        cleanup_go_download() { rm -f "$archive"; }
        trap cleanup_go_download EXIT
        url="https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
        curl --fail --silent --show-error --location \
            --proto '=https' --proto-redir '=https' \
            --output "$archive" "$url"
        expected="$GO_LINUX_AMD64_SHA256"
        if ! [[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]]; then
            echo "Checksum Go invalide." >&2
            exit 1
        fi
        if command -v sha256sum >/dev/null 2>&1; then
            actual="$(sha256sum "$archive" | awk '{print $1}')"
        elif command -v shasum >/dev/null 2>&1; then
            actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
        else
            echo "sha256sum ou shasum est requis." >&2
            exit 1
        fi
        if [ "${actual,,}" != "${expected,,}" ]; then
            echo "Checksum Go incorrect; installation refusee." >&2
            exit 1
        fi
        sudo tar -C /usr/local -xzf "$archive"
        cleanup_go_download
        trap - EXIT
        export PATH="/usr/local/go/bin:$PATH"
        echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
    fi
else
    echo "Go trouve : $(go version)"
fi

# ── Build ──────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "1/4 go mod tidy..."
go mod tidy

echo "2/4 go vet..."
go vet ./...

echo "3/4 go build..."
mkdir -p build
go build -o build/multiai ./cmd/multiai/
echo "  -> build/multiai"

echo "4/4 go test..."
go test -race -v ./...

echo ""
echo "Cross-compilation..."
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o build/multiai-linux-amd64   ./cmd/multiai/
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o build/multiai-darwin-amd64  ./cmd/multiai/
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o build/multiai-darwin-arm64  ./cmd/multiai/
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/multiai-windows-amd64.exe ./cmd/multiai/

echo ""
echo "Binaires generes :"
ls -lh build/
echo ""
echo "Setup termine avec succes !"
echo "Lance : ./build/multiai help"
