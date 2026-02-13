# New Features in Go Implementation

This document summarizes all the new features and improvements in the Go implementation of goenv that were previously undocumented or non-functional in the bash version.

## Quick Overview

| Feature                      | Status in Bash         | Status in Go          | Priority | Documentation                                                 |
| ---------------------------- | ---------------------- | --------------------- | -------- | ------------------------------------------------------------- |
| SBOM Generation & Inventory  | Not Available          | ✅ Fully Functional   | High     | [Commands Reference](reference/COMMANDS.md#goenv-sbom)        |
| Smart Caching & Offline Mode | Not Available          | ✅ Fully Functional   | High     | [Smart Caching Guide](advanced/SMART_CACHING.md)              |
| Configuration Hooks          | Not Available          | ✅ YAML config (NEW)  | High     | [Hooks Guide](../reference/HOOKS.md)                                       |
| GOPATH Integration           | Not Available          | ✅ Fully Functional   | High     | [GOPATH Integration](advanced/GOPATH_INTEGRATION.md)          |
| Auto-Rehash Control          | Not Available          | ✅ Automatic (NEW)    | High     | [Commands Reference](reference/COMMANDS.md#install)           |
| Enhanced UX Commands         | Not Available          | ✅ 6 new commands     | High     | [Quick Reference](QUICK_REFERENCE.md)                         |
| Version Shorthand            | Not Available          | ✅ Fully Functional   | Medium   | [Commands Reference](reference/COMMANDS.md#version-shorthand) |
| File Arg Detection           | Not Available          | ✅ Fully Functional   | Low      | [Hooks Guide](HOOKS.md#environment-variables)                 |
| Shell Completion             | Partial                | ✅ Complete           | Medium   | [Commands Reference](reference/COMMANDS.md)                   |
| VS Code Integration          | Not Available          | ✅ Fully Functional   | Medium   | [VS Code Guide](user-guide/VSCODE_INTEGRATION.md)             |

## 1. Smart Caching & Offline Mode 🚀

### What It Does

Intelligent caching system that stores Go version information locally with automatic cache management, plus a complete offline mode using embedded versions.

### Why It Matters

- **Speed:** Version lists load instantly from cache (10-50x faster)
- **Reliability:** Works without network access via offline mode
- **Cost Efficiency:** Reduces API calls to go.dev
- **Air-gapped Support:** Complete functionality in restricted environments
- **CI/CD Performance:** Faster builds with cached version data

### How It Works

**Cache Management:**

```bash
# Cache structure
$GOENV_ROOT/cache/
  ├── versions.json          # Cached version list
  ├── versions.json.etag     # ETag for cache validation
  └── versions.json.timestamp # Last fetch timestamp
```

**Cache Strategy:**

1. **First Request:** Fetches from go.dev API, stores in cache
2. **Subsequent Requests:** Reads from cache (instant)
3. **Cache Validation:** Uses HTTP ETag to check if cache is stale
4. **Auto-Refresh:** Updates cache when stale or expired
5. **Fallback:** Uses embedded versions if network unavailable

### Quick Start

**Using Cache (Default):**

```bash
# First time - fetches and caches
goenv install --list
# Takes 1-2 seconds (network call)

# Subsequent calls - reads from cache
goenv install --list
# Takes <100ms (instant!)

# Force cache refresh
goenv refresh
goenv install --list
```

**Using Offline Mode:**

```bash
# Enable offline mode
export GOENV_OFFLINE=1

# All version operations use embedded data
goenv install --list     # Uses embedded versions
goenv install 1.22.5     # No network calls
goenv versions          # Works completely offline

# Disable when network available
unset GOENV_OFFLINE
```

### Cache Commands

**Refresh Cache:**

```bash
# Clear cache and force fresh fetch
goenv refresh

# With verbose output
goenv refresh --verbose
# Output:
# ✓ Cache cleared! Removed 2 cache file(s).
# Next version fetch will retrieve fresh data from go.dev
```

**Check Cache Status:**

```bash
# View cache files
ls -lh $GOENV_ROOT/cache/

# Check cache age
stat $GOENV_ROOT/cache/versions.json
```

### Configuration

**Environment Variables:**

```bash
# Enable offline mode
export GOENV_OFFLINE=1

# Cache location (default: $GOENV_ROOT/cache)
# Cache is automatically managed
```

**Cache Behavior:**

- **TTL:** Cache is validated on each use via ETag
- **Size:** Typically 50-200KB for version data
- **Location:** `$GOENV_ROOT/cache/`
- **Auto-cleanup:** Stale cache files automatically replaced

### Performance Comparison

| Operation         | Bash Version | Go (First Call) | Go (Cached)    | Go (Offline)   |
| ----------------- | ------------ | --------------- | -------------- | -------------- |
| `install --list`  | 2-5 seconds  | 1-2 seconds     | <100ms         | <50ms          |
| `install 1.22.5`  | 3-30 seconds | 5-30 seconds    | 5-30 seconds\* | 5-30 seconds\* |
| Version detection | 100-500ms    | 10-50ms         | 10-50ms        | 10-50ms        |

\*Download time depends on network speed and Go version size

### Use Cases

**1. CI/CD Pipelines:**

```yaml
# .github/workflows/test.yml
steps:
  - name: Cache goenv versions
    uses: actions/cache@v3
    with:
      path: ~/.goenv/cache
      key: goenv-cache-${{ runner.os }}

  - name: Install Go (uses cache)
    run: |
      goenv install 1.22.5
      # Uses cached version list = faster builds
```

**2. Air-gapped Environments:**

```bash
# Development environment without internet
export GOENV_OFFLINE=1

# All operations work using embedded versions
goenv install --list    # 334 versions available
goenv install 1.22.5    # Downloads from mirror or uses local
```

**3. Offline Development:**

```bash
# Working on airplane/train
export GOENV_OFFLINE=1

# Continue working normally
goenv local 1.22.5
goenv versions
goenv which go
# Everything works!
```

**4. Bandwidth-Constrained Environments:**

```bash
# Fetch once, use many times
goenv install --list     # Fetches and caches
goenv refresh            # Only refresh when needed

# Subsequent calls use cache (no bandwidth)
goenv install --list     # Instant, no network
```

### Embedded Versions

The Go implementation includes **334+ embedded Go versions** for offline operation:

```bash
# Enable offline mode
export GOENV_OFFLINE=1

# List embedded versions
goenv install --list
# Shows all 334+ versions from Go 1.4beta1 to latest

# Install using embedded data (still downloads binary)
goenv install 1.22.5
```

**Embedded version data includes:**

- Version numbers
- Release dates
- Stability flags (stable/beta/rc)
- Platform support information

### Cache vs Offline Mode

| Feature        | Cache Mode (Default)   | Offline Mode            |
| -------------- | ---------------------- | ----------------------- |
| Network calls  | When cache stale       | Never                   |
| Version list   | From cache (validated) | From embedded data      |
| Updates        | Automatic (ETag-based) | No updates              |
| Speed          | Very fast (<100ms)     | Extremely fast (<50ms)  |
| Data freshness | Always current         | As fresh as goenv build |
| Use case       | Normal operation       | No network available    |

### Troubleshooting

**Cache Issues:**

```bash
# Cache seems stale
goenv refresh

# Cache corruption
rm -rf $GOENV_ROOT/cache
goenv install --list  # Rebuilds cache

# Verify cache is working
ls -la $GOENV_ROOT/cache/
cat $GOENV_ROOT/cache/versions.json | jq '.versions | length'
```

**Offline Mode Not Working:**

```bash
# Verify offline mode is enabled
echo $GOENV_OFFLINE  # Should show "1"

# Check embedded versions are available
goenv install --list | wc -l  # Should show 334+

# Test installation (still needs binary download)
goenv install 1.22.5
```

### Documentation

- [Complete Smart Caching Guide](advanced/SMART_CACHING.md)
- [Embedded Versions Details](advanced/EMBEDDED_VERSIONS.md)
- [Command Reference - refresh](reference/COMMANDS.md#goenv-refresh)

---

## 2. YAML-Based Configuration Hooks ⚡

### What It Does

YAML-based configuration system with predefined safe actions at specific goenv lifecycle points. Uses declarative configuration with limited, predefined actions—**not arbitrary shell scripts**—for security and cross-platform compatibility.

### Why It Matters

- **Security:** Limited to 6 predefined actions - no arbitrary code execution
- **Declarative:** Define what should happen with configuration, not scripts
- **Cross-Platform:** Works identically on Windows, macOS, and Linux
- **Template Support:** Dynamic variable interpolation in all actions
- **Non-blocking:** Hook failures don't break goenv commands
- **Safe by Design:** Configuration-based system prevents security risks

### Available Actions

- **`log_to_file`** - Write messages to log files
- **`http_webhook`** - Send HTTP POST requests with structured data
- **`notify_desktop`** - Display desktop notifications
- **`check_disk_space`** - Validate available disk space before operations
- **`set_env`** - Set environment variables for templates
- **`run_command`** - Execute shell commands safely

### Hook Points

- **`pre_install`** - Before installing a Go version
- **`post_install`** - After installing a Go version
- **`pre_uninstall`** - Before uninstalling a Go version
- **`post_uninstall`** - After uninstalling a Go version
- **`pre_exec`** - Before executing commands
- **`post_exec`** - After executing commands
- **`pre_rehash`** - Before rehashing shims
- **`post_rehash`** - After rehashing shims

### Quick Start

**1. Initialize Configuration:**

```bash
# Generate template configuration
goenv hooks init
```

**2. Enable Hooks:**

Edit `~/.goenv/hooks.yaml`:

```yaml
version: 1
enabled: true
acknowledged_risks: true

hooks:
  post_install:
    - action: log_to_file
      path: ~/.goenv/install.log
      message: "Installed Go {version} at {timestamp}"

    - action: notify_desktop
      title: "goenv"
      message: "Go {version} installed successfully"
```

**3. Test Configuration:**

```bash
# Validate configuration
goenv hooks validate

# Test hooks without executing
goenv hooks test post_install

# Install a version to see hooks in action
goenv install 1.22.5
```

### Template Variables

All actions support dynamic variable interpolation:

- `{version}` - Go version (e.g., "1.22.5")
- `{hook}` - Hook point name (e.g., "post_install")
- `{timestamp}` - ISO 8601 timestamp
- Custom variables from `set_env` actions

### Use Cases

- **Audit Logging:** Track all Go version installations
- **Team Notifications:** Send Slack/Teams messages on version changes
- **Space Management:** Check disk space before large installations
- **CI Integration:** Send webhooks to monitoring systems
- **Desktop Alerts:** Notify users of long-running operations

### Documentation

- **[Complete Hooks Guide](../reference/HOOKS.md)** - Comprehensive user documentation
- **[Internal Hooks Reference](hooks/HOOKPOINTS.md)** - Developer documentation for internal implementation

---

## 3. GOPATH Binary Integration 📦

### What It Does

Automatically manages binaries installed with `go install`, creating shims for tools in your GOPATH and isolating them per Go version.

### Why It Matters

- **Isolation:** Each Go version has its own GOPATH and tools
- **No Conflicts:** Avoid mixing Go modules from different versions
- **Seamless Switching:** Tools switch automatically with Go version
- **Clean Management:** Easy to see which tools are installed for each version

### How It Works

```bash
# Default structure
$HOME/go/
  ├── 1.21.5/bin/    # Tools for Go 1.21.5
  ├── 1.22.5/bin/    # Tools for Go 1.22.5
  └── 1.23.2/bin/    # Tools for Go 1.23.2

# Automatically scanned during rehash
```

### Quick Start

```bash
# Install a tool
goenv local 1.22.5
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create shims
goenv rehash

# Tool is now available
golangci-lint version

# Switch version - tool from 1.22.5 is no longer available
goenv local 1.21.5
golangci-lint version  # Not found

# Install for this version too
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
goenv rehash
golangci-lint version  # Works now
```

### Configuration

```bash
# Disable GOPATH integration
export GOENV_DISABLE_GOPATH=1

# Customize GOPATH location
export GOENV_GOPATH_PREFIX=/custom/path
```

### Use Cases

- **Development Tools:** golangci-lint, goimports, air, etc.
- **Project Tools:** Project-specific tooling per version
- **Testing:** Test tools with different Go versions
- **CI/CD:** Consistent tool versions in pipelines

### Documentation

- [Complete GOPATH Integration Guide](advanced/GOPATH_INTEGRATION.md)
- [Environment Variables Reference](reference/ENVIRONMENT_VARIABLES.md#goenv_disable_gopath)

---

## 4. Auto-Rehash Control ⚙️

### What It Does

Provides fine-grained control over automatic rehashing with the `--no-rehash` flag and `GOENV_NO_AUTO_REHASH` environment variable, allowing you to optimize performance in CI/CD and batch operations.

### Why It Matters

- **Performance:** Skip unnecessary rehashing in CI/CD pipelines
- **Batch Operations:** Install multiple versions efficiently
- **Docker Optimization:** Reduce container build times
- **Flexibility:** Automatic by default, controllable when needed
- **Best UX:** Works seamlessly for 95% of users, customizable for power users

### How It Works

**Automatic Rehashing (Default):**

```bash
# Rehash happens automatically after:
goenv install 1.22.5        # Auto-rehash creates shims
goenv exec go install ...   # Auto-rehash after tool installation

# Takes ~60ms - negligible for interactive use
```

**Disable Auto-Rehash for Single Command:**

```bash
# Skip rehash for one install
goenv install --no-rehash 1.22.5

# Install multiple versions efficiently
goenv install --no-rehash 1.21.0
goenv install --no-rehash 1.22.0
goenv install --no-rehash 1.23.0
goenv rehash  # Single rehash at the end
```

**Disable Auto-Rehash Globally:**

```bash
# Set environment variable
export GOENV_NO_AUTO_REHASH=1

# All operations skip auto-rehash
goenv install 1.22.5        # No auto-rehash
goenv exec go install ...   # No auto-rehash

# Manually rehash when needed
goenv rehash
```

### Quick Start

**CI/CD Pipeline Optimization:**

```yaml
# .github/workflows/test.yml
steps:
  - name: Install multiple Go versions
    env:
      GOENV_NO_AUTO_REHASH: 1
    run: |
      goenv install 1.21.0
      goenv install 1.22.0
      goenv install 1.23.0
      goenv rehash  # Single rehash

  - name: Run tests
    run: |
      goenv local 1.22.0
      go test ./...
```

**Docker Multi-Stage Build:**

```dockerfile
# Optimize installation phase
ENV GOENV_NO_AUTO_REHASH=1
RUN goenv install 1.22.5 && \
    goenv install 1.23.0 && \
    goenv rehash

# Or use flag for single installs
RUN goenv install --no-rehash 1.22.5 && goenv rehash
```

**Batch Script:**

```bash
#!/bin/bash
# Install multiple versions efficiently
export GOENV_NO_AUTO_REHASH=1

for version in 1.21.0 1.22.0 1.23.0; do
  goenv install "$version"
done

# Single rehash at the end
goenv rehash
echo "Installed and rehashed all versions"
```

### Configuration

**Flag:**

- `--no-rehash` - Skip automatic rehash for single command
- Applies to: `goenv install`

**Environment Variable:**

- `GOENV_NO_AUTO_REHASH=1` - Disable auto-rehash globally
- Applies to: `goenv install`, `goenv exec go install`, `goenv tools sync-tools`

### When to Use

**Use `--no-rehash` or `GOENV_NO_AUTO_REHASH=1` when:**

- Installing multiple Go versions in sequence
- Running in CI/CD pipelines
- Building Docker containers
- Running automated scripts
- Performance is critical

**Keep default (auto-rehash) when:**

- Interactive development (95% of use cases)
- Single version installs
- Learning/exploring goenv
- Convenience matters more than 60ms

### Performance Impact

| Scenario       | With Auto-Rehash | Without Auto-Rehash | Savings |
| -------------- | ---------------- | ------------------- | ------- |
| Single install | ~30s + 60ms      | ~30s                | 60ms    |
| 3 installs     | ~90s + 180ms     | ~90s + 60ms         | 120ms   |
| 10 installs    | ~300s + 600ms    | ~300s + 60ms        | 540ms   |

_Install time varies based on network speed and Go version size. Rehash is typically 50-70ms._

### Use Cases

- **CI/CD Pipelines:** Skip rehash during setup, rehash once at end
- **Docker Builds:** Optimize multi-version installations
- **Automation Scripts:** Batch operations with single rehash
- **Testing:** Install multiple versions for test matrix
- **Development:** Default auto-rehash "just works"

### Documentation

- [Commands Reference - Install](reference/COMMANDS.md#install)
- [Environment Variables Reference](reference/ENVIRONMENT_VARIABLES.md#goenv_no_auto_rehash)

---

## 5. PATH Command Discovery 🔌

### What It Does

Automatically discovers `goenv-*` commands from your system PATH, making them available as goenv subcommands.

### Why It Matters

- **Easy Extension:** Add custom commands by placing them in PATH
- **Discoverability:** Custom commands show in `goenv commands` output
- **Clean Integration:** Custom commands work exactly like built-in commands
- **No Configuration:** Just install and use

### How It Works

```bash
# Create a custom command script
cat > /usr/local/bin/goenv-hello << 'EOF'
#!/usr/bin/env bash
echo "Hello from custom command! Go version: $(goenv version-name)"
EOF

chmod +x /usr/local/bin/goenv-hello

# Automatically discovered and available as:
$ goenv hello
goenv commands | grep hello
```

### Use Cases

- **Custom Commands:** Add project-specific workflows
- **Team Tools:** Share team-wide goenv extensions
- **Cloud Integration:** Deploy, monitor, manage cloud resources
- **Version Management:** Custom version aliasing and management

### Documentation

- [Commands Reference](reference/COMMANDS.md#goenv-commands)

---

## 6. Version Shorthand Syntax ⚡

### What It Does

Allows using `goenv <version>` as a shortcut for `goenv local <version>`.

### Why It Matters

- **Convenience:** Faster version switching
- **Less Typing:** 2 words instead of 3
- **Intuitive:** Natural command syntax

### Quick Start

```bash
# Old way (still works)
goenv local 1.22.5

# New shorthand
goenv 1.22.5

# Works with all version formats
goenv 1.21.0
goenv latest
goenv system
```

### Supported Formats

- Full versions: `1.22.5`, `1.21.0`
- Minor versions: `1.22`, `1.21`
- Major versions: `1`, `2`
- Keywords: `latest`, `system`

### Documentation

- [Commands Reference](reference/COMMANDS.md#version-shorthand-syntax)

---

## 7. File Argument Detection 📄

### What It Does

Automatically detects when Go commands are executed with file arguments and sets `GOENV_FILE_ARG` environment variable.

### Why It Matters

- **Hook Context:** Hooks can see which files are being processed
- **Smart Actions:** Perform file-specific operations in hooks
- **Logging:** Track which files are compiled/run
- **Automation:** Trigger actions based on file types

### How It Works

```bash
# Command: go run main.go
# Hook receives: GOENV_FILE_ARG=main.go

# Command: go build ./cmd/app
# Hook receives: GOENV_FILE_ARG=./cmd/app

# Command: go test
# Hook receives: GOENV_FILE_ARG= (empty)
```

### Quick Start

The `GOENV_FILE_ARG` environment variable is available to the executed command, not to hooks. This is useful for Go applications and scripts that need to know which file was passed as an argument.

```bash
# Test with file argument
export GOENV_DEBUG=1
go run main.go  # Sets GOENV_FILE_ARG=main.go

# In your Go program, you can access it:
fileArg := os.Getenv("GOENV_FILE_ARG")
if fileArg != "" {
    fmt.Printf("Processing file: %s\n", fileArg)
}
```

**Note:** This feature sets the environment variable for the executed command, but the YAML hooks system does not currently expose `GOENV_FILE_ARG` as a template variable. Hooks receive `{command}` and `{version}` for exec hooks.

### Use Cases

- **File Logging:** Track which files are being compiled
- **Validation:** Check file existence/permissions before running
- **Formatting:** Auto-format files before execution
- **Metrics:** Collect statistics on file usage

### Documentation

- [Hooks Guide - Environment Variables](HOOKS.md#environment-variables)

---

## 8. Complete Shell Completion 🔄

### What It Does

Comprehensive tab completion support for all shells (bash, zsh, fish, PowerShell) with command-specific completion logic.

### Why It Matters

- **Productivity:** Faster command entry with intelligent suggestions
- **Discovery:** Find available commands and options easily
- **Accuracy:** Reduce typos and command errors

### How It Works

The Go implementation provides native completion generators for each shell format with command-specific logic for version names, file paths, and flags.

### Quick Start

See shell-specific setup in [INSTALL.md](INSTALL.md#shell-configuration).

---

## 9. VS Code Integration 🎯

### What It Does

Native Visual Studio Code integration with 5 commands to manage Go settings in `.vscode/settings.json` files. Features security validation, workspace-relative paths, and comprehensive health checks.

### Why It Matters

- **Team Collaboration:** Share consistent Go settings across team members
- **Portability:** Workspace-relative paths work across different machines
- **Security:** Schema validation prevents accidental modification of non-Go settings
- **Reliability:** Doctor command provides 8 health checks for troubleshooting
- **Productivity:** Automatic sync when switching Go versions

### Commands

1. **`goenv vscode init`** - Initialize VS Code settings with current Go environment
2. **`goenv vscode sync`** - Sync settings with current Go version
3. **`goenv vscode status`** - Show current VS Code configuration
4. **`goenv vscode doctor`** - Run 8 comprehensive health checks
5. **`goenv vscode revert`** - Restore settings from backup

### Security Features

- **Key Allowlist:** Only `go.*` and `gopls.*` keys can be modified
- **Schema Validation:** Validates all keys before writing
- **Deprecated Warnings:** Alerts about obsolete settings
- **Backup/Restore:** Automatic backups before any changes

### Quick Start

```bash
# Initialize with workspace-relative paths
goenv vscode init --goroot --gopath --workspace-paths

# Sync when switching Go versions
goenv local 1.22.5
goenv vscode sync

# Check health
goenv vscode doctor

# View current status
goenv vscode status
```

### Example: Portable Team Settings

```json
{
  "go.goroot": "${workspaceFolder}/.goenv/versions/1.22.5",
  "go.gopath": "${workspaceFolder}/.goenv/gopath/1.22.5",
  "go.toolsGopath": "${workspaceFolder}/.goenv/gopath/1.22.5/tools"
}
```

These paths work for all team members regardless of their home directory or OS (Windows/macOS/Linux).

### Doctor Command Checks

The `goenv vscode doctor` command performs 8 comprehensive health checks:

1. ✅ Workspace has `.vscode` directory
2. ✅ Settings file exists and is valid JSON
3. ✅ Go settings match current goenv version
4. ✅ Environment variables expand correctly
5. ✅ Paths exist on filesystem
6. ✅ No duplicate settings in workspace/folder levels
7. ✅ `gopls` is available
8. ✅ Current Go version is compatible

### Cross-Platform Compatibility

- **Windows:** Uses `${env:USERPROFILE}` for home directory
- **Unix/macOS:** Uses `${env:HOME}` for home directory
- **Path Normalization:** Converts backslashes to forward slashes in VS Code settings

### Documentation

- **[VS Code Integration Guide](user-guide/VSCODE_INTEGRATION.md)** - Complete usage guide
- **[VS Code Quick Reference](reference/VSCODE_QUICK_REFERENCE.md)** - Command reference
- [Commands Reference](reference/COMMANDS.md) - All goenv commands

---

## Implementation Details

### Performance Characteristics

| Feature                | Performance Impact                | Notes                         |
| ---------------------- | --------------------------------- | ----------------------------- |
| Smart Cache            | ~50-100ms (first), <50ms (cached) | 10-50x faster than bash       |
| Offline Mode           | <50ms                             | No network overhead           |
| Hooks                  | ~1-5ms per hook                   | Minimal unless hooks are slow |
| GOPATH Scan            | ~10-50ms                          | Only during rehash            |
| PATH Command Discovery | ~5-20ms                           | Only during command listing   |
| Version Shorthand      | < 1ms                             | No overhead                   |
| File Arg Detection     | < 1ms                             | Happens in shim               |
| Completion             | ~10-50ms                          | Only when requested           |

### Backward Compatibility

All features maintain full backward compatibility:

- Old commands still work
- New features are opt-in or transparent
- No breaking changes for existing users

### Environment Variables Added

| Variable               | Purpose                                     | Default      |
| ---------------------- | ------------------------------------------- | ------------ |
| `GOENV_OFFLINE`        | Enable offline mode (use embedded versions) | `0`          |
| `GOENV_FILE_ARG`       | File argument in Go commands                | _(auto-set)_ |
| `GOENV_DISABLE_GOPATH` | Disable GOPATH integration                  | `0`          |
| `GOENV_GOPATH_PREFIX`  | Custom GOPATH location                      | `$HOME/go`   |

### Commands Enhanced

| Command                | Enhancement                               |
| ---------------------- | ----------------------------------------- |
| `goenv install --list` | Smart caching, 10-50x faster              |
| `goenv refresh`        | New command to clear cache                |
| `goenv commands`       | Now includes `goenv-*` commands from PATH |
| `goenv rehash`         | Now scans GOPATH binaries                 |
| `goenv which`          | Now checks GOPATH                         |
| `goenv whence`         | Now checks GOPATH                         |
| `goenv hooks`          | Now actually executes hooks               |
| `goenv local`          | Now supports shorthand                    |
| `goenv global`         | Now has completion                        |
| `goenv install`        | Now has completion + caching              |
| `goenv uninstall`      | Now has completion                        |

---

## Getting Started

### For New Users

All features work out of the box with sensible defaults:

```bash
# Install goenv
git clone https://github.com/go-nv/goenv.git ~/.goenv

# Add to shell
echo 'eval "$(goenv init -)"' >> ~/.bashrc

# Install and use Go
goenv install 1.22.5
goenv 1.22.5  # Use shorthand!

# Install tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
goenv rehash  # Creates shims for GOPATH tools

# Everything just works!
```

### For Existing Users

See [Migration Guide](../user-guide/MIGRATION_GUIDE.md) for detailed upgrade instructions.

Key points:

1. Features are backward compatible
2. Hooks now execute (review if you had them)
3. GOPATH integration is optional but recommended
4. No configuration changes required

---

## Documentation Index

- **[Migration Guide](../user-guide/MIGRATION_GUIDE.md)** - Upgrading from bash version
- **[Hooks Guide](../reference/HOOKS.md)** - Complete hook system documentation
- **[GOPATH Integration](../advanced/GOPATH_INTEGRATION.md)** - GOPATH management guide
- **[Commands Reference](reference/COMMANDS.md)** - All available commands including custom ones
- **[Commands Reference](reference/COMMANDS.md)** - All commands with new features
- **[Environment Variables](reference/ENVIRONMENT_VARIABLES.md)** - Configuration reference

---

## Support and Feedback

- **Issues:** [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Discussions:** [GitHub Discussions](https://github.com/go-nv/goenv/discussions)
- **Documentation:** [docs/](.)

---

## 10. New User Experience Commands 🎯

### What They Do

Six new commands designed to improve user experience, particularly for first-time users and daily operations:

1. **`goenv info <version>`** - Detailed version information with lifecycle tracking
2. **`goenv compare <v1> <v2>`** - Side-by-side version comparison
3. **`goenv status`** - Quick installation health check
4. **`goenv setup`** - Automatic first-time configuration wizard
5. **`goenv get-started`** - Interactive beginner's guide
6. **`goenv explore`** - Command discovery by intent/category

### Why They Matter

- **Onboarding:** Dramatically simplified first-time setup
- **Decision Making:** Better information for version selection
- **Diagnostics:** Quick health checks complement comprehensive doctor command
- **Discovery:** Easy command navigation for new users
- **Lifecycle Management:** Track version EOL status and get upgrade recommendations

### Quick Start

**First-time setup:**
```bash
# Automatic configuration (detects shell, adds initialization, etc.)
goenv setup

# Interactive beginner guide
goenv get-started

# Quick health check
goenv status
```

**Version information:**
```bash
# Detailed version info (release date, EOL status, size, etc.)
goenv info 1.25.2

# Compare two versions side-by-side
goenv compare 1.21.5 1.23.2

# Check if current version is EOL
goenv info $(goenv current --bare)
```

**Command discovery:**
```bash
# Browse commands by category
goenv explore
# Shows: Getting Started, Version Management, Tools, Diagnostics, etc.
```

### Enhanced Doctor Command

The existing `goenv doctor` command has been significantly enhanced with interactive fix mode:

```bash
# Comprehensive diagnostics (existing)
goenv doctor

# NEW: Interactive repair mode
goenv doctor --fix
```

**Doctor improvements:**
- 18 structured issue types with automated fixes
- 3 new diagnostic checks (shell environment, profile sourcing, duplicate installations)
- Interactive fix mode that can automatically repair common issues
- Detects "undo sourcing" scenarios (major pain point)
- Better UX with colors, emojis, and clearer messages

### Documentation

- [Commands Reference - info](reference/COMMANDS.md#goenv-info)
- [Commands Reference - compare](reference/COMMANDS.md#goenv-compare)
- [Commands Reference - status](reference/COMMANDS.md#goenv-status)
- [Commands Reference - setup](reference/COMMANDS.md#goenv-setup)
- [Commands Reference - get-started](reference/COMMANDS.md#goenv-get-started)
- [Commands Reference - explore](reference/COMMANDS.md#goenv-explore)
- [Commands Reference - doctor](reference/COMMANDS.md#goenv-doctor)

---

## Summary

The Go implementation adds **10 major feature sets** that enhance goenv's functionality:

1. 🚀 **Smart Caching & Offline Mode** - 10-50x faster with intelligent caching and complete offline support
2. ⚡ **Hook System** - Extend goenv with custom scripts at 7 execution points
3. 📦 **GOPATH Integration** - Automatic tool management per version with isolated GOPATHs
4. ⚙️ **Auto-Rehash Control** - Configurable automatic rehashing for optimized CI/CD and batch operations
5. 🔌 **PATH Command Discovery** - Easy custom command addition via PATH-based auto-discovery
6. ⚡ **Version Shorthand** - Faster version switching with intuitive syntax
7. 📄 **File Arg Detection** - Context-aware hooks with automatic file detection
8. 🔄 **Complete Completion** - Better shell integration with comprehensive tab completion
9. 🎯 **VS Code Integration** - Native VS Code settings management with security validation and portability
10. 🎯 **Enhanced UX Commands** - 6 new commands for better onboarding, diagnostics, and version management

### Key Improvements

- **🚀 Performance:** 10-50x faster version operations with smart caching
- **📡 Reliability:** Complete offline mode with 334+ embedded versions
- **🔧 Extensibility:** Hooks for customization
- **⚙️ Automation:** GOPATH integration and auto-rehashing for seamless workflows
- **💡 UX:** Version shorthand and completion for better developer experience

All features are **production-ready**, **fully tested**, and **documented**.

Enjoy the enhanced goenv experience! 🚀
