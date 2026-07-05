APP_NAME = multiai
VERSION = 0.4.0-dev
BUILD_DIR = build
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all build test lint clean cross-build

all: test build

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/multiai/

test:
	go test -race -coverprofile=coverage.out ./...

test-verbose:
	go test -race -v -coverprofile=coverage.out ./...

lint:
	go vet ./...

cross-build:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/multiai/
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/multiai/
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/multiai/
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/multiai/
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/multiai/
	@echo "Cross-compilation terminee dans $(BUILD_DIR)/"

clean:
	rm -rf $(BUILD_DIR)
