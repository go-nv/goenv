# Go-based goenv Makefile

# Build variables
BINARY_NAME = goenv
VERSION ?= $(shell cat APP_VERSION 2>/dev/null || echo "dev")
COMMIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT_SHA) -X main.buildTime=$(BUILD_TIME)"

# Default installation prefix
export PREFIX ?= /usr/local

# Build targets
.PHONY: build clean test install uninstall dev-deps all cross-build generate-embedded test-windows release snapshot
.DEFAULT=build

# Default target
all: build

# Generate embedded versions from API (run before releases) 
generate-embedded:
	go run scripts/build-tool/main.go -task=generate-embedded

build:
	go run scripts/build-tool/main.go -task=build

build-swap:: build swap

test:
	go run scripts/build-tool/main.go -task=test

# Test Windows compatibility (can run on any OS)
test-windows:
	go run scripts/build-tool/main.go -task=test-windows

clean:
	go run scripts/build-tool/main.go -task=clean

install: build
	go run scripts/build-tool/main.go -task=install

uninstall:
	go run scripts/build-tool/main.go -task=uninstall

dev-deps:
	go run scripts/build-tool/main.go -task=dev-deps

# Cross-platform builds for releases
cross-build: generate-embedded
	go run scripts/build-tool/main.go -task=cross-build

# Migration helpers - these preserve some compatibility while transitioning
.PHONY: migrate-test

# Run Go tests alongside existing bats tests during migration
migrate-test:
	go run scripts/build-tool/main.go -task=migrate-test

bats-test:
	go run scripts/build-tool/main.go -task=bats-test

# Show version information
version:
	go run scripts/build-tool/main.go -task=version

# Cross-platform build tool (delegates to Go-based tool)
build-tool:
	go run scripts/build-tool/main.go -task=$(TASK)

# GoReleaser targets
release:
	go run scripts/build-tool/main.go -task=release

snapshot:
	go run scripts/build-tool/main.go -task=snapshot

swap:
	go run ./scripts/swap/main.go go
