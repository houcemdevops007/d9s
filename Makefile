BINARY=d9s
# Use ./mainpkg for local macOS build workaround, or ./cmd/d9s for standard structure.
CMD=./mainpkg
GOFLAGS=-mod=mod
BUILD_DIR=build
VERSION=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags="-X github.com/houcemdevops007/d9s/pkg/version.Version=$(VERSION) -X github.com/houcemdevops007/d9s/pkg/version.GitCommit=$(GIT_COMMIT) -X github.com/houcemdevops007/d9s/pkg/version.BuildDate=$(BUILD_DATE)"

.PHONY: all build run test fmt lint clean build-all build-linux build-darwin package

all: build

build:
	@echo "Building $(BINARY) for local OS..."
	mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) $(CMD)
	@echo "✓ Built ./$(BINARY)"

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	@echo "Building $(BINARY) for Linux (amd64)..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(CMD)

build-linux-arm64:
	@echo "Building $(BINARY) for Linux (arm64)..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 $(CMD)

build-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
	@echo "Building $(BINARY) for macOS (amd64)..."
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(CMD)

build-darwin-arm64:
	@echo "Building $(BINARY) for macOS (arm64)..."
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(CMD)

build-all: build-linux build-darwin

package: build-all
	@echo "Packaging binaries..."
	tar -czf $(BUILD_DIR)/d9s-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY)-linux-amd64
	tar -czf $(BUILD_DIR)/d9s-linux-arm64.tar.gz -C $(BUILD_DIR) $(BINARY)-linux-arm64
	tar -czf $(BUILD_DIR)/d9s-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY)-darwin-amd64
	tar -czf $(BUILD_DIR)/d9s-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY)-darwin-arm64
	@echo "✓ Packages created in $(BUILD_DIR)/"

run:
	go run $(GOFLAGS) $(CMD)

test:
	go test $(GOFLAGS) ./...

fmt:
	gofmt -w .

lint:
	go vet $(GOFLAGS) ./...

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ Installed to /usr/local/bin/$(BINARY)"
