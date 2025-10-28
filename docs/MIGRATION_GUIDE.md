# Migration Guide: Bash to Go Implementation

This guide helps users migrate from the bash-based goenv to the new Go implementation, highlighting new features and improvements.

## Table of Contents

- [Migration Guide: Bash to Go Implementation](#migration-guide-bash-to-go-implementation)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [What's Changed](#whats-changed)
    - [Performance Improvements](#performance-improvements)
    - [Enhanced Features](#enhanced-features)
  - [New Features](#new-features)
    - [1. Smart Caching \& Offline Mode](#1-smart-caching--offline-mode)
    - [2. GOPATH Integration](#2-gopath-integration)
    - [3. Version Shorthand Syntax](#3-version-shorthand-syntax)
    - [4. Diagnostic Tool (goenv doctor)](#4-diagnostic-tool-goenv-doctor)
    - [5. Architecture-Aware Caching](#5-architecture-aware-caching)
    - [6. Enhanced Safety Features](#6-enhanced-safety-features)
    - [7. File Argument Detection in Shims](#7-file-argument-detection-in-shims)
    - [8. Improved Completion Support](#8-improved-completion-support)
  - [Breaking Changes](#breaking-changes)
    - [None for Most Users](#none-for-most-users)
    - [1. Hooks System Changes (YAML-Based Configuration)](#1-hooks-system-changes-yaml-based-configuration)
    - [2. GOPATH Structure Change (If Enabled)](#2-gopath-structure-change-if-enabled)
  - [Migration Steps](#migration-steps)
    - [Step 1: Backup Configuration](#step-1-backup-configuration)
    - [Step 2: Install Go Implementation](#step-2-install-go-implementation)
    - [Step 3: Update Shell Configuration](#step-3-update-shell-configuration)
    - [Step 4: Test Basic Operations](#step-4-test-basic-operations)
    - [Step 5: Configure New Features (Optional)](#step-5-configure-new-features-optional)
    - [Step 6: Reinstall GOPATH Tools (If Using GOPATH Integration)](#step-6-reinstall-gopath-tools-if-using-gopath-integration)
    - [Step 7: Run Diagnostics](#step-7-run-diagnostics)
    - [Step 8: Verify Everything Works](#step-8-verify-everything-works)
  - [Feature Comparison](#feature-comparison)
  - [Troubleshooting](#troubleshooting)
    - ["Exec Format Error" or Wrong Architecture](#exec-format-error-or-wrong-architecture)
    - [Hooks Not Working](#hooks-not-working)
    - [GOPATH Tools Not Found](#gopath-tools-not-found)
    - [Slow Performance](#slow-performance)
    - [Version Not Switching](#version-not-switching)
  - [Summary](#summary)
    - [Quick Migration Checklist](#quick-migration-checklist)
    - [When You Need More](#when-you-need-more)
    - [What You Get](#what-you-get)
    - [Migration is Seamless](#migration-is-seamless)
  - [Getting Help](#getting-help)
  - [Further Reading](#further-reading)

## Overview

The Go implementation of goenv maintains **full backward compatibility** with the bash version while adding new features and improvements.

**✅ Seamless Migration for 95% of Users:**

The Go version was deliberately designed as a **drop-in replacement** for bash goenv:

- ✅ **Installed Go versions work unchanged** - All versions in `~/.goenv/versions/` continue to work
- ✅ **Configuration files unchanged** - `.go-version` and `~/.goenv/version` files work as-is
- ✅ **Shell setup unchanged** - Existing `eval "$(goenv init -)"` still works
- ✅ **Commands work the same** - All existing commands have same behavior
- ✅ **Environment variables compatible** - Existing variables still work

**Most users can simply:**

1. Update goenv (git pull)
2. Run `goenv rehash` (one-time)
3. Done! ✅

The migration guide below covers optional new features and edge cases, but **the basics just work**.

## What's Changed

### Performance Improvements

- **Faster command execution** - Go binary is significantly faster than bash scripts
- **Improved caching** - Smart caching system reduces network calls
- **Concurrent operations** - Better performance for operations that can run in parallel

### Enhanced Features

1. **Smart caching & offline mode** - 10-50x faster version lists with intelligent caching
2. **GOPATH integration** - Automatic management of GOPATH binaries
3. **Version shorthand** - Use `goenv 1.22.5` instead of `goenv local 1.22.5`

## New Features

### 1. Smart Caching & Offline Mode

**What's New:**

- Version information cached locally with automatic ETag-based validation
- `goenv install --list` is 10-50x faster after first run
- Complete offline mode with 334+ embedded Go versions
- New `goenv refresh` command to clear cache

**Migration Impact:** NONE

- Automatic and transparent
- No configuration changes needed
- Cache managed automatically

**Benefits:**

```bash
# First run - fetches and caches
goenv install --list  # 1-2 seconds

# Subsequent runs - reads from cache
goenv install --list  # <100ms (instant!)

# Work completely offline
export GOENV_OFFLINE=1
goenv install --list  # Uses embedded versions

# Clear cache when needed
goenv refresh
```

**See:** [Smart Caching Guide](advanced/SMART_CACHING.md)

### 2. GOPATH Integration

````

### 2. GOPATH Binary Integration

**What's New:**
- goenv automatically creates shims for GOPATH-installed binaries
- Each Go version has its own GOPATH: `$HOME/go/{version}/bin`
- Tools installed with `go install` are isolated per version

**Migration Impact:** MEDIUM
- Existing GOPATH structure may need adjustment
- Tools installed with `go install` will need reinstallation per version
- Can be disabled with `GOENV_DISABLE_GOPATH=1`

**Migration Steps:**
```bash
# Option 1: Enable GOPATH integration (recommended)
unset GOENV_DISABLE_GOPATH  # or set to 0

# Reinstall tools for each version
goenv local 1.22.5
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
goenv rehash

# Option 2: Disable if you manage GOPATH manually
export GOENV_DISABLE_GOPATH=1
````

**See:** [GOPATH Integration Guide](advanced/GOPATH_INTEGRATION.md)

### 3. Version Shorthand Syntax

**What's New:**

- Use `goenv <version>` as shorthand for `goenv local <version>`
- Works with version numbers, "latest", and "system"

**Migration Impact:** NONE

- New feature, doesn't affect existing workflows
- Old syntax still works: `goenv local 1.22.5`

**Example:**

```bash
# Old way (still works)
goenv local 1.22.5

# New shorthand (convenience)
goenv 1.22.5

# Both are equivalent
```

### 4. Diagnostic Tool (goenv doctor)

**What's New:**

- New `goenv doctor` command for comprehensive system diagnostics
- Detects environment issues (WSL, containers, NFS mounts)
- Verifies cache isolation and architecture compatibility
- Network connectivity check using HTTPS HEAD request (works in CI/containers where ICMP/ping is blocked)
- Identifies configuration problems automatically

**Migration Impact:** NONE

- Diagnostic tool only, doesn't change behavior
- Highly recommended to run after migration
- **Fixed:** Network check now uses HTTPS instead of ICMP ping, making it reliable in containerized environments and CI pipelines

**Example:**

```bash
# Run comprehensive diagnostics
goenv doctor

# Checks:
# ✓ Runtime environment (native, WSL, Docker)
# ✓ Filesystem type (local, NFS, SMB)
# ✓ Network connectivity (HTTPS to go.dev)
# ✓ goenv binary and paths
# ✓ Shell configuration
# ✓ Build cache isolation
# ✓ Architecture compatibility
# ✓ Rosetta detection (macOS)
# ✓ Common configuration problems
```

**See:** [Commands Reference - doctor](reference/COMMANDS.md#goenv-doctor)

### 5. Architecture-Aware Caching

**What's New:**

- Build caches automatically isolated per architecture
- Prevents "exec format error" when switching architectures
- Separate caches for cross-compilation targets
- Enhanced cache name parsing for ARM variants (armv5/v6/v7), CGO hashes, and GOEXPERIMENT flags
- Cache migration correctly detects runtime platform (not affected by GOOS/GOARCH environment variables)
- New `goenv cache` command for cache management

**Migration Impact:** LOW

- Automatic for new installs
- Old non-architecture-aware caches still work
- Can migrate old caches with `goenv cache migrate`
- **Fixed:** Cache migration now uses runtime platform detection, preventing incorrect architecture assignment during cross-compilation

**Benefits:**

```bash
# Prevents this common error when switching architectures:
# exec format error: cached binary is wrong architecture

# Caches are now organized as:
~/.goenv/versions/1.23.2/
  ├── go-build-darwin-arm64/   # macOS Apple Silicon
  ├── go-build-darwin-amd64/   # macOS Intel
  └── go-build-linux-amd64/    # Linux cross-compile

# Manage caches
goenv cache status              # View cache sizes
goenv cache clean build         # Clean build caches
goenv cache migrate             # Migrate old format caches
```

**Important:** `goenv cache migrate` is **NOT** for migrating from bash to Go goenv. It only migrates cache formats within the Go implementation (from old single-cache to new architecture-aware caches).

**See:** [Commands Reference - cache](reference/COMMANDS.md#goenv-cache)

### 6. Enhanced Safety Features

**What's New:**

- **WSL Detection**: Warns when running Windows binaries in WSL or vice versa
- **Rosetta Detection** (macOS): Warns about x86_64/arm64 architecture mixing
- **Container Detection**: Identifies NFS/SMB mounts and Docker bind mounts
- **Session Memoization**: Prevents repeated architecture verification prompts

**Migration Impact:** NONE

- Warning-only features that help identify issues
- Don't break existing workflows
- Provide actionable suggestions

**Example warnings you might see:**

```bash
# On Apple Silicon with mixed architectures
⚠️  Mixing architectures: goenv is native arm64 but tool is x86_64
   Consider: Rebuilding the tool with native arm64 Go version

# In WSL with Windows binary
⚠️  Running Windows binary in WSL. This may work via Windows interop but could have issues.
   Consider rebuilding for Linux: GOOS=linux GOARCH=amd64 go install <package>@latest

# On NFS mount
⚠️  Cache directory is on a remote/networked filesystem (nfs).
   Consider: Using a local cache directory with GOENV_GOCACHE_DIR
```

### 7. File Argument Detection in Shims

**What's New:**

- Shims automatically detect file arguments in `go*` commands
- Sets `GOENV_FILE_ARG` environment variable
- Available to executed commands (not currently exposed to YAML hooks)

**Migration Impact:** NONE

- Transparent feature for most users
- The environment variable is set for the executed command itself

**Example:**

```bash
# When running: go run main.go
# The executed command receives: GOENV_FILE_ARG=main.go

# In your Go program:
package main
import (
    "fmt"
    "os"
)

func main() {
    if fileArg := os.Getenv("GOENV_FILE_ARG"); fileArg != "" {
        fmt.Printf("Processing file: %s\n", fileArg)
    }
}
```

**Note:** The YAML hooks system does not currently expose `GOENV_FILE_ARG` as a template variable. Hooks receive `{command}` and `{version}` for exec hooks.

### 8. Improved Completion Support

**What's New:**

- All commands now support `--complete` flag
- Better shell integration
- Completion for `local`, `global`, `install`, `uninstall`

**Migration Impact:** NONE

- Completions work better automatically
- No configuration changes needed

## Breaking Changes

### None for Most Users

The Go implementation maintains full backward compatibility. However, be aware of:

### 1. New Feature: YAML-Based Hooks System

**What's New:**

The Go version 3.0 introduces a **brand new YAML-based declarative hooks system** - this feature did NOT exist in bash goenv.

**This is NOT a migration** - it's an entirely new opt-in feature with predefined safe actions to prevent arbitrary code execution.

**Available Predefined Actions:**

1. `log_to_file` - Write messages to log files
2. `http_webhook` - Send HTTP POST requests
3. `notify_desktop` - Display desktop notifications
4. `check_disk_space` - Validate available disk space
5. `set_env` - Set environment variables
6. `run_command` - Execute commands (with safety controls)

**Impact:** NONE for existing users

- This is an opt-in feature - nothing changes if you don't use it
- No configuration needed unless you want to enable hooks
- Bash goenv never had hooks, so there's nothing to migrate

**Getting Started with Hooks:**

```bash
# Initialize configuration file
goenv hooks init

# Edit ~/.goenv/hooks.yaml and set:
# enabled: true
# acknowledged_risks: true

# Example configuration:
version: 1
enabled: true
acknowledged_risks: true

hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Installed Go {version} at {timestamp}"

# Validate configuration
goenv hooks validate

# Test without executing
goenv hooks test post_install
```

**See:** [Hooks Guide](HOOKS.md) for complete documentation

### 2. GOPATH Structure Change (If Enabled)

**Issue:** GOPATH structure changes from single global to per-version.

**Before:**

```
$HOME/go/
├── bin/
├── pkg/
└── src/
```

**After:**

```
$HOME/go/
├── 1.21.5/
│   ├── bin/
│   ├── pkg/
│   └── src/
└── 1.22.5/
    ├── bin/
    ├── pkg/
    └── src/
```

**Fix:**

```bash
# Option 1: Keep old structure (disable GOPATH integration)
export GOENV_DISABLE_GOPATH=1

# Option 2: Migrate to new structure (recommended)
# Reinstall tools for each version as shown above
```

## Migration Steps

### Step 1: Backup Configuration

```bash
# Backup your goenv directory
cp -r ~/.goenv ~/.goenv.backup

# Backup shell configuration
cp ~/.bashrc ~/.bashrc.backup
cp ~/.zshrc ~/.zshrc.backup 2>/dev/null || true
```

### Step 2: Install Go Implementation

```bash
# Pull latest changes
cd ~/.goenv
git fetch
git checkout go-go  # or master once merged
git pull

# Rebuild if needed
make
```

### Step 3: Update Shell Configuration

No changes needed! Your existing configuration works:

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Step 4: Test Basic Operations

```bash
# Test version detection
goenv version

# Test version switching
goenv local 1.22.5
go version

# Test installation (if you have versions installed)
goenv versions
```

### Step 5: Configure New Features (Optional)

```bash
# Enable GOPATH integration (recommended)
# Add to ~/.bashrc or ~/.zshrc
unset GOENV_DISABLE_GOPATH

# Set up YAML-based hooks (if desired)
goenv hooks init
# Then edit ~/.goenv/hooks.yaml and set:
# enabled: true
# acknowledged_risks: true

# Customize GOPATH location (optional)
# export GOENV_GOPATH_PREFIX="/custom/path"
```

### Step 6: Reinstall GOPATH Tools (If Using GOPATH Integration)

```bash
# For each Go version you use
for version in $(goenv versions --bare); do
  echo "Setting up tools for $version..."
  goenv local "$version"

  # Reinstall your commonly used tools
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  go install golang.org/x/tools/cmd/goimports@latest
  # ... other tools

  goenv rehash
done
```

### Step 7: Run Diagnostics

```bash
# Run comprehensive diagnostics (RECOMMENDED)
goenv doctor

# This checks:
# - Runtime environment (native, WSL, Docker)
# - Filesystem type (local, NFS, SMB)
# - goenv binary and paths
# - Shell configuration
# - PATH setup
# - Build cache isolation
# - Architecture compatibility
# - Rosetta detection (macOS)
# - Tool migration suggestions
# - Common configuration problems

# Review any warnings and follow the recommendations
```

### Step 8: Verify Everything Works

```bash
# Test command execution
goenv exec go version

# Test shims
which go
go version

# Test tools (if using GOPATH integration)
which golangci-lint
golangci-lint version

# Test hooks (if configured)
GOENV_DEBUG=1 goenv exec go version

# Verify cache isolation (should see version-specific cache)
go env GOCACHE
# Should show: ~/.goenv/versions/{version}/go-build-{os}-{arch}
```

## Feature Comparison

| Feature                | Bash Version              | Go Version                 | Notes                            |
| ---------------------- | ------------------------- | -------------------------- | -------------------------------- |
| Version switching      | ✅                        | ✅                         | Faster in Go                     |
| Installation           | ✅                        | ✅                         | Better caching                   |
| Diagnostics            | ❌                        | ✅ `goenv doctor`          | Comprehensive checks             |
| Architecture isolation | ❌                        | ✅ Automatic               | Prevents exec format error       |
| Cache management       | ❌                        | ✅ `goenv cache`           | Status, clean, migrate           |
| Hooks                  | ❌ Not available          | ✅ YAML declarative (NEW)  | Predefined actions, opt-in       |
| GOPATH integration     | ❌                        | ✅ Optional                | Per-version isolation            |
| Version shorthand      | ❌                        | ✅                         | `goenv 1.22.5`                   |
| File arg detection     | ❌                        | ✅                         | `GOENV_FILE_ARG`                 |
| Completion             | ✅                        | ✅                         | More complete                    |
| Safety warnings        | ❌                        | ✅                         | WSL, Rosetta, NFS detection      |
| Session memoization    | ❌                        | ✅                         | Reduces repeated checks          |
| Performance            | Good                      | Excellent                  | 10-50x faster                    |
| Platform support       | Unix-like                 | Unix-like + Windows        | Better Windows support           |

## Troubleshooting

### "Exec Format Error" or Wrong Architecture

**Symptom:** Getting "exec format error" or "bad CPU type" when running tools or cross-compiling.

**Cause:** Cached binaries built for wrong architecture (e.g., arm64 binary on x86_64 system).

**Solution:**

```bash
# Option 1: Run diagnostics to identify the issue
goenv doctor
# Look for warnings about:
# - Architecture mismatches
# - Rosetta detection
# - Cache isolation

# Option 2: Clean build caches
goenv cache clean build

# Option 3: Clean specific version's cache
goenv cache clean build --version 1.23.2

# Option 4: Migrate old format caches (if you upgraded from early Go version)
goenv cache migrate

# Option 5: Verify cache isolation is working
go env GOCACHE
# Should show architecture-specific path:
# ~/.goenv/versions/1.23.2/go-build-darwin-arm64
```

**Prevention:**

The Go implementation automatically prevents this with architecture-aware caching. If you're seeing this error after migration:

1. Run `goenv doctor` to check cache configuration
2. Clean old caches: `goenv cache clean build`
3. Verify: `go env GOCACHE` shows architecture-specific path

### Hooks Not Working

**Symptom:** YAML-based hooks configured but not executing.

**Solution:**

```bash
# Check if hooks.yaml exists
ls -la ~/.goenv/hooks.yaml

# Validate configuration
goenv hooks validate
# Should show: ✓ Configuration is valid

# Check if hooks are enabled in hooks.yaml
grep -A2 "enabled:" ~/.goenv/hooks.yaml
# Should show:
# enabled: true
# acknowledged_risks: true

# Test specific hook without executing
goenv hooks test post_install

# Test with debug mode
GOENV_DEBUG=1 goenv install 1.23.2
# Should show hook execution in debug output

# List all available hooks
goenv hooks list
```

**Common Issues:**

1. **`enabled: false`** - Set to `true` in `~/.goenv/hooks.yaml`
2. **`acknowledged_risks: false`** - Must be `true` to run hooks
3. **Invalid YAML syntax** - Run `goenv hooks validate` to check
4. **Wrong action names** - Must use predefined actions: `log_to_file`, `http_webhook`, `notify_desktop`, `check_disk_space`, `set_env`, `run_command`

### GOPATH Tools Not Found

**Symptom:** Tools installed with `go install` not available.

**Solution:**

```bash
# Check if GOPATH integration is enabled
echo $GOENV_DISABLE_GOPATH
# Should be empty or 0

# Rehash after installing tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
goenv rehash

# Check tool location
goenv which golangci-lint
```

### Slow Performance

**Symptom:** Commands slower than expected.

**Solution:**

```bash
# Check for slow hooks
GOENV_DEBUG=1 goenv exec go version
# Look for hook execution times in debug output

# Temporarily disable hooks to test
# Edit ~/.goenv/hooks.yaml and set:
# enabled: false
goenv exec go version

# Re-enable when done testing
# enabled: true

# Optimize slow hooks:
# - Avoid synchronous operations in hooks
# - Consider using http_webhook with timeouts
# - Use log_to_file instead of network calls when possible
```

### Version Not Switching

**Symptom:** `goenv local` doesn't affect `go version`.

**Solution:**

```bash
# Verify goenv init is in shell config
grep 'goenv init' ~/.bashrc ~/.zshrc

# Check shims are in PATH
echo $PATH | grep goenv/shims

# Rehash shims
goenv rehash

# Check which go is being used
which go
# Should be: /home/user/.goenv/shims/go
```

## Summary

### Quick Migration Checklist

For **95% of users**, migration is just:

- [ ] Update goenv: `cd ~/.goenv && git pull`
- [ ] Rehash shims: `goenv rehash`
- [ ] Run diagnostics: `goenv doctor`
- [ ] Done! ✅

### When You Need More

Only consider these if you want to leverage new features:

- [ ] **GOPATH Integration**: Reinstall tools per version (optional, can disable)
- [ ] **Cache Migration**: Run `goenv cache migrate` if you see old cache warnings
- [ ] **Hooks Configuration**: Set up new YAML-based hooks (opt-in, see `goenv hooks init`)

### What You Get

The Go implementation gives you:

- ✅ **10-50x faster** version operations
- ✅ **Automatic safety checks** (architecture, WSL, Rosetta, NFS)
- ✅ **Better diagnostics** with `goenv doctor`
- ✅ **Architecture-aware caching** prevents "exec format error"
- ✅ **Offline mode** with 334+ embedded versions
- ✅ **Full backward compatibility** with bash version

### Migration is Seamless

The Go version is a true **drop-in replacement**:

- All `.go-version` files work unchanged
- All installed versions work unchanged
- All shell configuration works unchanged
- All existing commands work the same way
- New features are optional and don't break existing workflows

**You get new features automatically without breaking anything!** 🎉

## Getting Help

- **Documentation:** [docs/](.)
- **Issues:** [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Discussions:** [GitHub Discussions](https://github.com/go-nv/goenv/discussions)

## Further Reading

- [New Features Overview](NEW_FEATURES.md)
- [Hooks Guide](HOOKS.md)
- [Smart Caching Guide](advanced/SMART_CACHING.md)
- [GOPATH Integration Guide](advanced/GOPATH_INTEGRATION.md)
- [Commands Reference](reference/COMMANDS.md)
- [Environment Variables Reference](reference/ENVIRONMENT_VARIABLES.md)
