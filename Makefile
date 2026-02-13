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
.PHONY: test-verbose test-report test-watch test-coverage test-quick check-gotestsum
.DEFAULT=build

# Default target
all: build

# Generate embedded versions from API (run before releases) 
generate-embedded:
	go run scripts/build-tool/main.go -task=generate-embedded

build:
	go run scripts/build-tool/main.go -task=build

build-swap:: build swap

# alias for build-swap
bs: build-swap

test:
	unset GOENV_DEBUG && go run scripts/build-tool/main.go -task=test

# Enhanced test targets using gotestsum (install with: go install gotest.tools/gotestsum@latest)
check-gotestsum:
	@go run gotest.tools/gotestsum@latest --version > /dev/null 2>&1 || (echo "⚠️  gotestsum not available. It will be downloaded on first use." && echo "   Or install with: go install gotest.tools/gotestsum@latest")

# Run tests with clean, readable output (failures saved to .test-results/failures.txt)
test-quick:
	@echo "Running tests with gotestsum..."
	@mkdir -p .test-results
	@unset GOENV_DEBUG && go run gotest.tools/gotestsum@latest --format testname --jsonfile .test-results/test-output.json -- -race ./... 2>&1 | tee .test-results/full-output.log || true
	@if [ -f .test-results/test-output.json ]; then \
		go run gotest.tools/gotestsum@latest tool slowest --jsonfile .test-results/test-output.json --threshold 1s 2>/dev/null || true; \
		grep -A 20 '"Action":"fail"' .test-results/test-output.json | grep -E '"(Package|Test|Output)"' > .test-results/failures.txt 2>/dev/null || echo "No failures detected" > .test-results/failures.txt; \
	fi

# Verbose test output showing all details
test-verbose:
	@mkdir -p .test-results
	@unset GOENV_DEBUG && go run gotest.tools/gotestsum@latest --format standard-verbose --jsonfile .test-results/test-output.json -- -race ./... 2>&1 | tee .test-results/full-output.log

# Generate test reports (JUnit XML + HTML coverage + failures summary)
test-report:
	@echo "Running tests with report generation..."
	@mkdir -p .test-results
	@unset GOENV_DEBUG && go run gotest.tools/gotestsum@latest --junitfile .test-results/junit.xml --jsonfile .test-results/test-output.json --format testname -- -race -coverprofile=.test-results/coverage.out ./... 2>&1 | tee .test-results/full-output.log || true
	@go tool cover -html=.test-results/coverage.out -o .test-results/coverage.html 2>/dev/null || true
	@if [ -f .test-results/test-output.json ]; then \
		echo "\n=== Test Failures ===" > .test-results/failures.txt; \
		go run gotest.tools/gotestsum@latest tool slowest --jsonfile .test-results/test-output.json --threshold 1s 2>/dev/null || true; \
		grep '"Action":"fail"' .test-results/test-output.json | jq -r '"\(.Package).\(.Test): \(.Output)"' 2>/dev/null >> .test-results/failures.txt || echo "No failures detected" >> .test-results/failures.txt; \
	fi
	@echo "✓ Coverage report: .test-results/coverage.html"
	@echo "✓ JUnit report: .test-results/junit.xml"
	@echo "✓ Failures summary: .test-results/failures.txt"
	@echo "✓ Full output: .test-results/full-output.log"

# Debug mode: Only show failures in terminal, save all to files
test-debug:
	@echo "Running tests in debug mode (failures only)..."
	@mkdir -p .test-results
	@unset GOENV_DEBUG && go run gotest.tools/gotestsum@latest --format dots --jsonfile .test-results/test-output.json -- -race ./... > .test-results/full-output.log 2>&1 || true
	@echo "\n=== Failed Tests ===" | tee .test-results/failures.txt
	@grep -h "FAIL" .test-results/full-output.log | head -50 | tee -a .test-results/failures.txt || echo "✓ All tests passed"
	@echo "\nFull details: .test-results/full-output.log"
	@echo "JSON output: .test-results/test-output.json"

# Watch mode for development (reruns tests on file changes)
test-watch:
	@echo "Starting test watcher..."
	@unset GOENV_DEBUG && go run gotest.tools/gotestsum@latest --watch --format testname

# Quick coverage summary
test-coverage:
	@unset GOENV_DEBUG && go test -coverprofile=coverage.out ./... > /dev/null
	@go tool cover -func=coverage.out | grep total
	@rm coverage.out

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

restore:
	go run ./scripts/swap/main.go bash

swap:
	go run ./scripts/swap/main.go go
