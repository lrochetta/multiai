#!/usr/bin/env bash
# Setup script for multiai Go development (macOS/Linux)
set -euo pipefail

echo "multiai Go Setup"
echo "==============="
echo ""

# ── Install Go if missing ──────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
    echo "Go non trouve. Installation..."
    GO_VERSION="1.23.2"
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install go
    else
        curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | sudo tar -C /usr/local -xz
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
