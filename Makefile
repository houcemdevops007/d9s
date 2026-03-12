BINARY=d9s
CMD=./cmd/d9s
GOFLAGS=-mod=mod

.PHONY: all build run test fmt lint clean

all: build

build:
	@echo "Building $(BINARY)..."
	go build $(GOFLAGS) -ldflags="-X github.com/houcemdevops007/d9s/pkg/version.Version=0.1.0 -X github.com/houcemdevops007/d9s/pkg/version.GitCommit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) -X github.com/houcemdevops007/d9s/pkg/version.BuildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o $(BINARY) $(CMD)
	@echo "✓ Built ./$(BINARY)"

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

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ Installed to /usr/local/bin/$(BINARY)"
