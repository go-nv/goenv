# Migration Guide: Bash to Go Implementation

This guide helps users migrate from the bash-based goenv to the new Go implementation, highlighting new features and improvements.

## Table of Contents

- [Overview](#overview)
- [What's Changed](#whats-changed)
- [New Features](#new-features)
- [Breaking Changes](#breaking-changes)
- [Migration Steps](#migration-steps)
- [Feature Comparison](#feature-comparison)

## Overview

The Go implementation of goenv maintains full backward compatibility with the bash version while adding new features and improvements. Most users can upgrade seamlessly without any configuration changes.

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

### 5. File Argument Detection in Shims

**What's New:**

- Shims automatically detect file arguments in `go*` commands
- Sets `GOENV_FILE_ARG` environment variable
- Available to hooks for context-aware operations

**Migration Impact:** NONE

- Transparent feature for most users
- Useful if you write hooks that need file context

**Example:**

```bash
# When running: go run main.go
# Hook receives: GOENV_FILE_ARG=main.go

# Example hook using this:
cat > ~/.goenv/hooks/exec.bash << 'EOF'
#!/usr/bin/env bash
if [[ -n $GOENV_FILE_ARG ]]; then
  echo "Processing file: $GOENV_FILE_ARG"
fi
EOF
```

### 6. Improved Completion Support

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

### 1. Hook Execution (Previously Non-functional)

**Issue:** If you had hooks configured in bash version, they were only listed but never executed.

**Now:** Hooks actually execute, which could cause unexpected behavior if:

- Hooks were written but never tested (they didn't run before)
- Hooks have side effects you weren't aware of
- Hooks are slow and now impact command performance

**✅ Cross-Platform Support:** Hooks now support multiple script types:

- **Unix/macOS/Linux**: `.bash`, `.sh` scripts, or shebang-based scripts
- **Windows PowerShell**: `.ps1` scripts
- **Windows CMD**: `.cmd`, `.bat` scripts
- goenv automatically detects and uses the appropriate interpreter

**Fix:**

```bash
# Unix/macOS - Temporarily disable hooks to test
unset GOENV_HOOK_PATH

# Windows PowerShell
Remove-Item Env:GOENV_HOOK_PATH

# Review and test hooks individually
# Unix/macOS
bash ~/.goenv/hooks/exec/exec.bash

# Windows PowerShell
& "$env:GOENV_ROOT\hooks\exec\exec.ps1"

# Re-enable when ready (Unix/macOS)
export GOENV_HOOK_PATH="$HOME/.goenv/hooks"

# Re-enable when ready (Windows PowerShell)
$env:GOENV_HOOK_PATH="$env:USERPROFILE\.goenv\hooks"
```

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

# Set up hooks (if desired)
export GOENV_HOOK_PATH="$HOME/.goenv/hooks"
mkdir -p "$GOENV_HOOK_PATH"

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

### Step 7: Verify Everything Works

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
```

## Feature Comparison

| Feature            | Bash Version   | Go Version          | Notes                  |
| ------------------ | -------------- | ------------------- | ---------------------- |
| Version switching  | ✅             | ✅                  | Faster in Go           |
| Installation       | ✅             | ✅                  | Better caching         |
| Hooks              | ⚠️ Listed only | ✅ Functional       | Hooks now execute      |
| GOPATH integration | ❌             | ✅ Optional         | New feature            |
| Version shorthand  | ❌             | ✅                  | `goenv 1.22.5`         |
| File arg detection | ❌             | ✅                  | `GOENV_FILE_ARG`       |
| Completion         | ✅             | ✅                  | More complete          |
| Performance        | Good           | Excellent           | 10-50x faster          |
| Platform support   | Unix-like      | Unix-like + Windows | Better Windows support |

## Troubleshooting

### Hooks Not Working

**Symptom:** Hooks configured but not executing.

**Solution:**

```bash
# Check GOENV_HOOK_PATH is set
echo $GOENV_HOOK_PATH

# Check hooks are executable
ls -la ~/.goenv/hooks/

# Make executable if needed
chmod +x ~/.goenv/hooks/*.bash

# Test with debug mode
GOENV_DEBUG=1 goenv exec go version
```

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

# Temporarily disable hooks to test
unset GOENV_HOOK_PATH
goenv exec go version

# Optimize slow hooks (use background processes)
# In hook: (slow_operation) &
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

## Getting Help

- **Documentation:** [docs/](.)
- **Issues:** [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Discussions:** [GitHub Discussions](https://github.com/go-nv/goenv/discussions)

## Further Reading

- [Hooks Guide](HOOKS.md)
- [GOPATH Integration Guide](advanced/GOPATH_INTEGRATION.md)
- [Commands Reference](reference/COMMANDS.md)
- [Environment Variables Reference](reference/ENVIRONMENT_VARIABLES.md)
- [Command Reference](reference/COMMANDS.md)
