.PHONY: build test lint clean fmt

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS = -X github.com/koblas/mushu/internal/version.Version=$(VERSION) \
          -X github.com/koblas/mushu/internal/version.Commit=$(COMMIT) \
          -X github.com/koblas/mushu/internal/version.BuildTime=$(BUILD_TIME)

# Build the mushu binary
build:
	go build -ldflags "$(LDFLAGS)" -o bin/mushu ./cmd/main.go

# Build with debug information
build-debug:
	go build -ldflags "$(LDFLAGS)" -gcflags="all=-N -l" -o bin/mushu ./cmd/main.go

# Build for release
build-release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS) -s -w" -o bin/mushu-linux-amd64 ./cmd/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS) -s -w" -o bin/mushu-darwin-amd64 ./cmd/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS) -s -w" -o bin/mushu-darwin-arm64 ./cmd/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS) -s -w" -o bin/mushu-windows-amd64.exe ./cmd/main.go

# Run tests
test:
	go test ./...

# Run golangci-lint
lint:
	golangci-lint run

# Format code with gofumpt
fmt:
	gofumpt -w .

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all checks (test + lint)
check: test lint

# Build and test
all: deps fmt lint test build

