# GOPATH Integration Guide

goenv automatically integrates with GOPATH to manage binaries installed via `go install`, ensuring that tools installed with different Go versions remain isolated and accessible.

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Configuration](#configuration)
- [Common Use Cases](#common-use-cases)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

When you install tools using `go install`, they are placed in your GOPATH bin directory. goenv's GOPATH integration:

1. **Automatically creates shims** for GOPATH-installed binaries
2. **Isolates tools per Go version** - each version has its own GOPATH
3. **Makes version switching seamless** - tools switch with Go versions
4. **Prevents version conflicts** - no mixing of Go modules from different versions

## How It Works

### Default Behavior

By default, goenv manages GOPATH automatically:

```bash
# GOPATH structure (default)
$HOME/go/
  ├── 1.21.5/
  │   ├── bin/           # Tools installed with Go 1.21.5
  │   ├── pkg/
  │   └── src/
  ├── 1.22.5/
  │   ├── bin/           # Tools installed with Go 1.22.5
  │   ├── pkg/
  │   └── src/
  └── 1.23.2/
      ├── bin/           # Tools installed with Go 1.23.2
      ├── pkg/
      └── src/
```

### Shim Creation

When you run `goenv rehash`, goenv scans both:

- Version binaries: `$GOENV_ROOT/versions/{version}/bin/*`
- GOPATH binaries: `$GOENV_GOPATH_PREFIX/{version}/bin/*`

This creates shims for all discovered binaries, making them available via `goenv exec` and in your PATH.

### Example Workflow

```bash
# Switch to Go 1.22.5
$ goenv use 1.22.5

# Install a tool
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# Installs to: $HOME/go/1.22.5/bin/golangci-lint

# Rehash to create shim
$ goenv rehash

# Tool is now available
$ golangci-lint version
golangci-lint has version 1.55.2

# Switch to Go 1.21.5
$ goenv use 1.21.5

# The tool from 1.22.5's GOPATH is no longer accessible
$ golangci-lint version
goenv: golangci-lint: command not found

# Install the tool for this version
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# Installs to: $HOME/go/1.21.5/bin/golangci-lint

$ goenv rehash
$ golangci-lint version
golangci-lint has version 1.55.2
```

## Configuration

### Environment Variables

| Variable               | Default    | Description                                       |
| ---------------------- | ---------- | ------------------------------------------------- |
| `GOENV_DISABLE_GOPATH` | `0`        | Set to `1` to disable GOPATH integration entirely |
| `GOENV_GOPATH_PREFIX`  | `$HOME/go` | Base directory for version-specific GOPATHs       |

### Disabling GOPATH Integration

If you want to manage GOPATH yourself:

```bash
# In your shell config (~/.bashrc, ~/.zshrc)
export GOENV_DISABLE_GOPATH=1

# goenv will no longer scan GOPATH for binaries
```

### Custom GOPATH Location

To use a different GOPATH location:

```bash
# In your shell config
export GOENV_GOPATH_PREFIX=/custom/path

# GOPATHs will be:
# /custom/path/1.21.5/bin
# /custom/path/1.22.5/bin
# etc.
```

## Common Use Cases

### Use Case 1: Development Tools

Install development tools per Go version:

```bash
# Development with Go 1.22.5
$ goenv use 1.22.5
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
$ go install golang.org/x/tools/cmd/goimports@latest
$ go install github.com/cosmtrek/air@latest
$ goenv rehash

# These tools are now available and tied to Go 1.22.5
```

### Use Case 2: Testing Across Versions

Test tools with different Go versions:

```bash
# Test with Go 1.21.5
$ goenv use 1.21.5
$ go install ./cmd/mytool
$ mytool --version
mytool version 1.0.0 (go1.21.5)

# Test with Go 1.22.5
$ goenv use 1.22.5
$ go install ./cmd/mytool
$ mytool --version
mytool version 1.0.0 (go1.22.5)
```

### Use Case 3: Project-Specific Tools

Each project can have its own tool versions:

```bash
# Project A uses Go 1.21.5 and golangci-lint v1.54
$ cd ~/projects/project-a
$ goenv use 1.21.5
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
$ goenv rehash

# Project B uses Go 1.22.5 and golangci-lint v1.55
$ cd ~/projects/project-b
$ goenv use 1.22.5
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
$ goenv rehash

# Tools switch automatically with Go version
$ cd ~/projects/project-a
$ golangci-lint version  # v1.54.2 with Go 1.21.5

$ cd ~/projects/project-b
$ golangci-lint version  # v1.55.2 with Go 1.22.5
```

### Use Case 4: CI/CD Pipelines

Ensure consistent tool versions in CI:

```yaml
# .github/workflows/lint.yml
steps:
  - name: Set up Go
    run: |
      goenv use 1.22.5
      go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
      goenv rehash

  - name: Lint
    run: golangci-lint run
```

## Troubleshooting

### Tool Not Found After Installation

**Problem:** Installed tool with `go install` but it's not available.

**Solution:**

```bash
# Ensure you've rehashed after installation
$ goenv rehash

# Verify the tool is in GOPATH
$ ls $GOENV_GOPATH_PREFIX/$(goenv version-name)/bin/

# Check that GOPATH integration is enabled
$ echo $GOENV_DISABLE_GOPATH
# Should be empty or 0
```

### Wrong Tool Version Running

**Problem:** Running tool from wrong Go version.

**Solution:**

```bash
# Check which version is active
$ goenv version

# Check which binary is being used
$ goenv which golangci-lint

# If wrong version, verify GOPATH structure
$ ls -la $GOENV_GOPATH_PREFIX/*/bin/golangci-lint
```

### GOPATH Not Isolated

**Problem:** Tools from different versions are mixing.

**Solution:**

```bash
# Verify GOENV_GOPATH_PREFIX is set correctly
$ echo $GOENV_GOPATH_PREFIX
# Should be: /home/user/go (or custom path)

# Check if GOPATH is being managed by goenv
$ echo $GOPATH
# Should be: /home/user/go/1.22.5 (or similar)

# Verify goenv init is in your shell config
$ grep 'goenv init' ~/.bashrc ~/.zshrc
```

### Old Tools Still Available

**Problem:** Tools from old versions still showing up.

**Solution:**

```bash
# Clean up old shims
$ goenv rehash

# Manually remove old tools if needed
$ rm -rf $GOENV_GOPATH_PREFIX/1.20.0

# Verify shims
$ goenv whence golangci-lint
```

## Best Practices

### 1. Always Rehash After Installing Tools

```bash
# Good workflow
$ go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
$ goenv rehash

# Even better - create a shell function
goinstall() {
  go install "$@" && goenv rehash
}
```

### 2. Document Tool Versions

Create a `tools.go` file in your project:

```go
//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "github.com/cosmtrek/air"
)
```

Then install all tools:

```bash
$ cat tools.go | grep _ | awk '{print $2}' | xargs -L1 go install
$ goenv rehash
```

### 3. Use Makefiles for Tool Installation

```makefile
.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	go install golang.org/x/tools/cmd/goimports@latest
	goenv rehash

.PHONY: lint
lint: tools
	golangci-lint run
```

### 4. Pin Tool Versions in CI

```bash
# Don't use @latest in CI
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

# Do use @latest for local development
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 5. Clean Up Unused Versions

```bash
# List all GOPATH directories
$ ls -la $GOENV_GOPATH_PREFIX/

# Remove old versions you no longer use
$ rm -rf $GOENV_GOPATH_PREFIX/1.19.0
$ rm -rf $GOENV_GOPATH_PREFIX/1.20.0

# Rehash to clean up stale shims
$ goenv rehash
```

### 6. Check Tool Locations

```bash
# Always verify which binary will be used
$ goenv which golangci-lint

# List all versions with the tool
$ goenv whence golangci-lint

# See full paths
$ goenv whence --path golangci-lint
```

## Integration with Other Tools

### IDE Integration (VS Code)

Configure VS Code to use goenv's managed tools:

```json
{
  "go.toolsGopath": "${env:HOME}/go/${env:GOENV_VERSION}",
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}"
}
```

### Make Integration

```makefile
# Ensure tools are installed in the right GOPATH
GOVERSION := $(shell goenv version-name)
GOPATH := $(HOME)/go/$(GOVERSION)

tools:
	GOPATH=$(GOPATH) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	goenv rehash
```

## Performance Considerations

### Rehash Performance

`goenv rehash` scans both version and GOPATH directories. For many versions/tools:

1. **Rehash is fast** - typically < 100ms even with many versions
2. **Rehash on demand** - only run when you install new tools

### GOPATH Size

Each version's GOPATH can grow large with many tools:

```bash
# Check GOPATH sizes
$ du -sh $GOENV_GOPATH_PREFIX/*

# Clean up build cache if needed
$ go clean -cache -modcache
```

## Migration from Manual GOPATH

If you previously managed GOPATH manually:

```bash
# 1. Backup existing GOPATH
$ cp -r $GOPATH $GOPATH.backup

# 2. Enable goenv GOPATH management
$ unset GOENV_DISABLE_GOPATH

# 3. Reinstall tools for each Go version
$ for version in $(goenv list --bare); do
    goenv use $version
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    # ... other tools
  done

# 4. Rehash
$ goenv rehash

# 5. Verify
$ goenv which golangci-lint
```

## Further Reading

- [Environment Variables Reference](../reference/ENVIRONMENT_VARIABLES.md)
- [Command Reference](../reference/COMMANDS.md)
- [Advanced Configuration](ADVANCED_CONFIGURATION.md)
