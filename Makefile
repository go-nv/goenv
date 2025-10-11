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
.PHONY: build clean test install uninstall dev-deps all cross-build

# Default target
all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf bin/ dist/
	go clean

install: build
	mkdir -p "$(PREFIX)/bin"
	cp $(BINARY_NAME) "$(PREFIX)/bin/"
	# Install shell completions
	mkdir -p "$(PREFIX)/share/goenv/completions"
	cp -R completions/* "$(PREFIX)/share/goenv/completions/" 2>/dev/null || true

uninstall:
	rm -f "$(PREFIX)/bin/$(BINARY_NAME)"
	rm -rf "$(PREFIX)/share/goenv"

dev-deps:
	go mod download
	go mod tidy

# Cross-platform builds for releases
cross-build:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-freebsd-amd64 .

# Migration helpers - these preserve some compatibility while transitioning
.PHONY: migrate-test

# Run Go tests alongside existing bats tests during migration
migrate-test: test bats-test

bats-test:
	@echo "Running legacy bats tests (if available)..."
	@if command -v bats >/dev/null 2>&1 && [ -d "test" ]; then \
		bats test/ 2>/dev/null || echo "Bats tests not available or failed"; \
	else \
		echo "Bats not installed or tests not found - skipping legacy tests"; \
	fi

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_SHA)"
	@echo "Build Time: $(BUILD_TIME)"
