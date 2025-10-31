# CI/CD Integration Guide

This guide explains how to optimize goenv for CI/CD pipelines using the two-phase installation strategy.

## Overview

The `goenv ci-setup` command provides two modes:

1. **Environment Setup** (default): Outputs environment variables for CI configuration
2. **Two-Phase Installation** (`--install`): Optimizes caching by separating Go installation from usage

### CI/CD Best Practices

For robust CI pipelines, goenv provides:

1. **Two-phase installation** with `goenv ci-setup --install` for optimal caching
2. **Health validation** with `goenv doctor --json` for catching configuration issues
3. **Structured exit codes** (0/1/2) for errors vs warnings
4. **Machine-readable JSON output** for automation and parsing

All quick start examples below include `goenv doctor --json` validation to catch configuration issues early. See [Health Checks and Validation](#health-checks-and-validation) for advanced usage.

## Why Two-Phase Installation?

Traditional CI approach:

```bash
# Install and use in one step - harder to cache effectively
goenv install 1.23.2
goenv use 1.23.2 --global
```

Two-phase approach (optimized):

```bash
# Phase 1: Install Go versions (cached separately)
goenv ci-setup --install 1.23.2

# Phase 2: Use the cached version (fast)
goenv use 1.23.2 --global
```

**Benefits:**

- Better cache utilization
- Faster CI runs when versions are cached
- Clear separation of concerns
- Support for batch installation with `--skip-rehash`

## Quick Start

### GitHub Actions

```yaml
name: Go CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up goenv
        run: |
          git clone https://github.com/go-nv/goenv.git ~/.goenv
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
          echo "$HOME/.goenv/shims" >> $GITHUB_PATH

      - name: Cache Go installations
        uses: actions/cache@v3
        with:
          path: ~/.goenv/versions
          key: ${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}
          restore-keys: |
            ${{ runner.os }}-goenv-

      - name: Install Go versions
        run: goenv ci-setup --install --from-file --skip-rehash

      - name: Use Go version
        run: goenv use --auto

      - name: Validate goenv setup
        run: goenv doctor --json --fail-on=error

      - name: Run tests
        run: go test ./...
```

### GitLab CI

```yaml
image: ubuntu:latest

variables:
  GOENV_ROOT: ${CI_PROJECT_DIR}/.goenv
  GOTOOLCHAIN: local

cache:
  key: ${CI_COMMIT_REF_SLUG}
  paths:
    - .goenv/versions

before_script:
  - apt-get update && apt-get install -y git curl
  - git clone https://github.com/go-nv/goenv.git ${GOENV_ROOT}
  - export PATH="${GOENV_ROOT}/bin:${GOENV_ROOT}/shims:$PATH"
  - goenv ci-setup --install --from-file --skip-rehash
  - goenv use --auto
  - goenv doctor --json --fail-on=error

test:
  script:
    - go test ./...
```

### CircleCI

```yaml
version: 2.1

jobs:
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

      - restore_cache:
          keys:
            - goenv-{{ checksum ".go-version" }}
            - goenv-

      - run:
          name: Install goenv
          command: |
            git clone https://github.com/go-nv/goenv.git ~/.goenv
            echo 'export PATH="$HOME/.goenv/bin:$HOME/.goenv/shims:$PATH"' >> $BASH_ENV

      - run:
          name: Install Go versions
          command: goenv ci-setup --install --from-file --skip-rehash

      - save_cache:
          key: goenv-{{ checksum ".go-version" }}
          paths:
            - ~/.goenv/versions

      - run:
          name: Use Go version
          command: goenv use --auto

      - run:
          name: Validate goenv setup
          command: goenv doctor --json --fail-on=error

      - run:
          name: Run tests
          command: go test ./...
```

### Travis CI

```yaml
os: linux
dist: focal

cache:
  directories:
    - $HOME/.goenv/versions

before_install:
  - git clone https://github.com/go-nv/goenv.git ~/.goenv
  - export PATH="$HOME/.goenv/bin:$HOME/.goenv/shims:$PATH"
  - goenv ci-setup --install --from-file --skip-rehash
  - goenv use --auto
  - goenv doctor --json --fail-on=error

script:
  - go test ./...
```

## Command Reference

### Basic Environment Setup

```bash
# Output environment variables (default mode)
goenv ci-setup

# With verbose recommendations
goenv ci-setup --verbose

# Specific shell format
goenv ci-setup --shell bash
goenv ci-setup --shell github    # GitHub Actions format
goenv ci-setup --shell gitlab    # GitLab CI format
```

### Two-Phase Installation

```bash
# Install specific versions
goenv ci-setup --install 1.23.2 1.22.5

# Install from .go-version file
goenv ci-setup --install --from-file

# Install without rehashing (faster for multiple versions)
goenv ci-setup --install --from-file --skip-rehash

# After installation, use the version
goenv use 1.23.2 --global
# or
goenv use --auto
```

## Supported Version Files

The `--from-file` flag reads versions from:

1. **`.go-version`** - Simple text file with version

   ```
   1.23.2
   ```

2. **`.tool-versions`** - ASDF-compatible format

   ```
   golang 1.23.2
   nodejs 20.0.0
   ```

3. **`go.mod`** - Go module file

   ```go
   module myproject

   go 1.23
   ```

   Note: go.mod versions may need to be expanded (e.g., `1.23` → `1.23.2`)

## Performance Optimization

### Skip Rehashing for Multiple Installations

When installing multiple Go versions, skip rehashing during installation and do it once at the end:

```bash
# Using ci-setup (automatically optimized)
goenv ci-setup --install 1.23.2 1.22.5 1.21.0 --skip-rehash

# Manual approach
export GOENV_NO_AUTO_REHASH=1
goenv install 1.23.2
goenv install 1.22.5
goenv install 1.21.0
goenv rehash
```

### Cache Strategy

**Recommended cache key patterns:**

- GitHub Actions: `${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}`
- GitLab CI: `goenv-${CI_COMMIT_REF_SLUG}`
- CircleCI: `goenv-{{ checksum ".go-version" }}`

**What to cache:**

**Primary (essential):**
- `~/.goenv/versions` - Installed Go versions (saves download time)

**Optional (nice to have):**
- `~/.goenv/cache` - Version metadata cache with ETags (saves ~40ms per version check)
- `~/.cache/go-build` - Build cache (speeds up builds)
- `~/go/pkg/mod` - Module cache (speeds up dependency downloads)

**Cache size considerations:**

| Directory | Typical Size | Speed Gain | When to Cache |
|-----------|--------------|------------|---------------|
| `~/.goenv/versions` | 100-500 MB per version | ⚡⚡⚡ Large (minutes) | Always |
| `~/.goenv/cache` | ~2 MB | ⚡ Small (~40ms) | Optional - fast anyway |
| `~/.cache/go-build` | 500 MB - 2 GB | ⚡⚡ Medium (seconds) | Recommended for builds |
| `~/go/pkg/mod` | 100 MB - 1 GB | ⚡⚡ Medium (seconds) | Recommended for projects with many deps |

**Note on `~/.goenv/cache`:**
- Contains version list cache with ETags for efficient update checks
- Very small (~2 MB) but update checks are already fast (<500ms with cache miss, <100ms with ETag hit)
- **Recommendation**: Skip caching this unless you're optimizing every millisecond
- Use `GOENV_OFFLINE=1` instead for maximum speed (bypasses network entirely)

**Example cache configuration with all paths:**

```yaml
# GitHub Actions - comprehensive caching
- name: Cache goenv
  uses: actions/cache@v3
  with:
    path: |
      ~/.goenv/versions
      ~/.goenv/cache
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-goenv-all-${{ hashFiles('**/go.sum', '.go-version') }}
    restore-keys: |
      ${{ runner.os }}-goenv-all-
      ${{ runner.os }}-goenv-
```

**Minimal cache configuration (recommended for most users):**

```yaml
# GitHub Actions - just cache installed versions
- name: Cache Go installations
  uses: actions/cache@v3
  with:
    path: ~/.goenv/versions
    key: ${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}
    restore-keys: |
      ${{ runner.os }}-goenv-
```

## Cache Cleaning in CI

---

### ⚠️ IMPORTANT: Non-Interactive Confirmation in CI

**Cache cleaning commands require confirmation in CI/CD pipelines.**

You have **two options** to handle this:

#### Option 1: `GOENV_ASSUME_YES` Environment Variable (Recommended)

Set `GOENV_ASSUME_YES=1` to auto-confirm all prompts globally:

```yaml
# GitHub Actions (recommended)
env:
  GOENV_ASSUME_YES: 1
run: |
  goenv cache clean build
  goenv cache migrate
  # All prompts auto-confirmed
```

```bash
# GitLab CI / Jenkins
export GOENV_ASSUME_YES=1
goenv cache clean all
```

**Why this is better:**
- ✅ More explicit about intent (auto-confirming vs forcing)
- ✅ Works globally for all goenv commands
- ✅ Self-documenting in CI config files
- ✅ Follows industry standards (like `DEBIAN_FRONTEND=noninteractive`)

#### Option 2: `--force` Flag (Alternative)

Add `--force` to individual commands:

```bash
# Per-command approach
goenv cache clean build --force
goenv cache migrate --force
```

#### What happens without either?

You'll see a helpful error with suggestions:

```
⚠️  Running in non-interactive mode (no TTY detected)

This command requires confirmation. Options:
  1. Add --force flag: goenv cache clean all --force
  2. Use dry-run first: goenv cache clean all --dry-run
  3. Set env var: GOENV_ASSUME_YES=1 goenv cache clean

For CI/CD, we recommend: GOENV_ASSUME_YES=1
```

**The rule:** Use `GOENV_ASSUME_YES=1` globally (recommended) or add `--force` to each cache command.

---

> **📝 Best Practice: Use `GOENV_ASSUME_YES` in CI**
>
> For CI/CD pipelines, set `GOENV_ASSUME_YES=1` in your environment variables. This auto-confirms all prompts without needing `--force` on every command.
>
> **Examples below show both approaches** - choose what works best for your pipeline.

When managing cache sizes or cleaning up between builds, non-interactive confirmation prevents CI pipeline hangs.

### Clean Build Caches

```bash
# Approach 1: Using GOENV_ASSUME_YES (recommended)
export GOENV_ASSUME_YES=1
goenv cache clean build
goenv cache clean build --older-than 30d
goenv cache clean all

# Approach 2: Using --force flag
goenv cache clean build --force
goenv cache clean build --older-than 30d --force
goenv cache clean all --force

# Preview what would be deleted (no confirmation needed)
goenv cache clean all --older-than 30d --dry-run
```

### Size-Based Cleanup

```bash
# Approach 1: Using GOENV_ASSUME_YES
GOENV_ASSUME_YES=1 goenv cache clean build --max-bytes 1GB
GOENV_ASSUME_YES=1 goenv cache clean all --max-bytes 500MB

# Approach 2: Using --force flag
goenv cache clean build --max-bytes 1GB --force
goenv cache clean all --max-bytes 500MB --force
```

### Best Practices

- **Use `GOENV_ASSUME_YES=1`** (recommended): Set globally for all commands
- **Alternative: use `--force`**: Add to individual cache commands
- **Use `--older-than`**: Clean caches by age rather than deleting all
- **Test with `--dry-run`**: Preview cleanup before applying (no confirmation needed)
- **Monitor cache sizes**: Use `goenv cache status --fast` for quick checks

**Example: Helpful error without GOENV_ASSUME_YES or --force:**

```
$ goenv cache clean build
⚠️  Running in non-interactive mode (no TTY detected)

This command requires confirmation. Options:
  1. Add --force flag: goenv cache clean build --force
  2. Use dry-run first: goenv cache clean build --dry-run
  3. Set env var: GOENV_ASSUME_YES=1 goenv cache clean

For CI/CD, we recommend: GOENV_ASSUME_YES=1
```

### Example: Scheduled Cache Cleanup

```yaml
# GitHub Actions - weekly cache maintenance
name: Cache Maintenance

on:
  schedule:
    - cron: "0 0 * * 0" # Weekly on Sunday

jobs:
  cleanup:
    runs-on: ubuntu-latest
    steps:
      - name: Clean old caches
        run: goenv cache clean all --older-than 30d --force
```

## Environment Variables

Key environment variables for CI optimization:

```bash
# Prevent automatic Go downloads
export GOTOOLCHAIN=local

# Disable GOPATH (use modules only)
export GOENV_DISABLE_GOPATH=1

# Skip automatic rehashing during install
export GOENV_NO_AUTO_REHASH=1

# Enable debug output (for troubleshooting)
export GOENV_DEBUG=1
```

### Offline Mode for Maximum Speed & Reproducibility

> **🚀 CI/CD Best Practice: Use `GOENV_OFFLINE=1`**
>
> For optimal CI/CD performance and guaranteed reproducibility, enable offline mode to use embedded versions (no network calls):
>
> ```bash
> export GOENV_OFFLINE=1
> ```
>
> **Benefits:**
> - ✅ **5-10x faster** version listing (~8ms vs ~500ms with network)
> - ✅ **100% reproducible** builds (no dependency on external API availability)
> - ✅ **Zero network bandwidth** for version metadata
> - ✅ **Works in air-gapped CI** runners and private networks
> - ✅ **No external dependencies** or DNS resolution required
>
> **What's available offline:**
> - 331 Go versions embedded at build time (updated with each goenv release)
> - All `goenv install --list`, `goenv list --remote` commands work offline
> - Version installation still requires downloading Go binaries (unless pre-cached)
>
> **Example GitHub Actions workflow with offline mode:**
> ```yaml
> - name: Set up goenv
>   run: |
>     git clone https://github.com/go-nv/goenv.git ~/.goenv
>     echo "$HOME/.goenv/bin" >> $GITHUB_PATH
>     echo "$HOME/.goenv/shims" >> $GITHUB_PATH
>
> - name: Install Go (offline mode)
>   env:
>     GOENV_OFFLINE: 1  # Use embedded versions for speed
>   run: |
>     goenv ci-setup --install --from-file --skip-rehash
>     goenv use --auto
> ```

### Network Reliability in CI

All goenv network operations have sensible defaults for CI reliability:

| **Operation**       | **Timeout** | **Fallback Behavior**                    |
| ------------------- | ----------- | ---------------------------------------- |
| Version fetching    | 30 seconds  | Falls back to cached data if available   |
| Doctor health check | 3 seconds   | Returns warning (not error) if unreachable |

**Debug network issues in CI:**
```bash
# Show detailed network/cache behavior
GOENV_DEBUG=1 goenv install --list

# Verify connectivity
goenv doctor --json

# Force cache refresh after network issues
goenv refresh
```

See [Network Reliability Defaults](advanced/SMART_CACHING.md#network-reliability-defaults) for complete details.

## Windows CI Environments

### PowerShell PATH and Quoting Gotchas

Windows CI environments (GitHub Actions, Azure Pipelines) use PowerShell, which has specific quoting rules that differ from bash/sh.

> **💡 Good News: `goenv ci-setup` Handles This Automatically**
>
> When you use `goenv ci-setup --shell powershell`, all the quoting complexity described below is handled for you! The command outputs properly escaped PowerShell code that:
>
> - Doubles single quotes in paths (`'` → `''`)
> - Backtick-escapes double quotes in PATH assignments (`"` → `` `" ``)
> - Handles spaces, parentheses, and special characters correctly
>
> **This escaping is thoroughly tested** across paths with spaces, parentheses, single quotes, and other special characters (see `cmd/ci-setup_test.go:306-352` for test coverage).
>
> If you're manually constructing PowerShell PATH commands, the rules below explain what you need to know.

#### Safe PATH Modification

```powershell
# ✅ CORRECT: Append to PATH safely (handles spaces correctly)
$env:PATH = "$env:USERPROFILE\.goenv\bin;$env:USERPROFILE\.goenv\shims;$env:PATH"

# ✅ CORRECT: Using goenv ci-setup for PowerShell (recommended!)
$output = goenv ci-setup --shell powershell
Invoke-Expression $output

# ❌ WRONG: Don't use bare variables with spaces
$env:PATH = $HOME\.goenv\bin;$env:PATH  # Breaks if $HOME has spaces
```

#### PowerShell String Quoting Rules

**Why This Matters:** PowerShell treats strings differently than bash/sh, especially when paths contain spaces or special characters. Windows user directories often have spaces (e.g., `C:\Users\John Smith\.goenv`), making proper quoting critical for CI reliability.

PowerShell has unique quoting requirements for arguments with special characters:

| **Character**    | **Bash/sh**        | **PowerShell Single-Quoted String**        | **PowerShell Double-Quoted String** | **goenv ci-setup Handling**           |
| ---------------- | ------------------ | ------------------------------------------ | ----------------------------------- | ------------------------------------- |
| Space            | `'arg with space'` | `'arg with space'`                         | `"arg with space"`                  | Preserved in both quote styles        |
| Single quote `'` | `"can't"`          | `'can''t'` (double the quote)              | `"can't"` (no escape needed)        | Doubled in `$env:GOENV_ROOT = '…'`    |
| Double quote `"` | `'say "hi"'`       | `'say "hi"'` (no escape needed)            | `"say `"hi`""` (backtick escape)    | Backtick-escaped in `$env:PATH = "…"` |
| Backtick `` ` `` | `` 'tick`' ``      | `` 'tick`' `` (no escape needed)           | ` "tick`" `` (double the backtick)  | Rare in practice                      |
| Parentheses      | `'dir (x)'`        | `'dir (x)'`                                | `"dir (x)"`                         | No escape needed (tested in CI)       |
| Env variable     | `"$HOME/path"`     | `'$env:HOME\path'` (literal, not expanded) | `"$env:HOME\path"` (expanded)       | Variables only in double quotes       |

**Key Insight:** The quoting style matters!

- **Single-quoted strings** (`'…'`) in PowerShell are literal (no variable expansion). Single quotes inside must be doubled: `'don''t'`
- **Double-quoted strings** (`"…"`) expand variables (`$env:VAR`). Double quotes inside must be backtick-escaped: `` "path`"with`"quotes" ``

**How `goenv ci-setup --shell powershell` handles this:**

- Uses single quotes for `$env:GOENV_ROOT = '...'` (no variables to expand) → escapes single quotes by doubling them
- Uses double quotes for `$env:PATH = "...\bin;...\shims;$env:PATH"` (expands `$env:PATH`) → escapes double quotes with backticks

#### Practical Examples: What Works vs What Breaks

**Scenario 1: Path with spaces**

```powershell
# ✅ WORKS: Properly quoted
$env:GOENV_ROOT = 'C:\Users\John Smith\.goenv'
$env:PATH = "C:\Users\John Smith\.goenv\bin;$env:PATH"

# ❌ BREAKS: Unquoted path with spaces
$env:PATH = C:\Users\John Smith\.goenv\bin;$env:PATH  # PowerShell error: unexpected token 'Smith'
```

**Scenario 2: Path with single quote (e.g., `C:\Users\john's folder\.goenv`)**

```powershell
# ✅ WORKS: Single quote doubled in single-quoted string
$env:GOENV_ROOT = 'C:\Users\john''s folder\.goenv'

# ❌ BREAKS: Single quote not escaped
$env:GOENV_ROOT = 'C:\Users\john's folder\.goenv'  # PowerShell error: unexpected token '''
```

**Scenario 3: Using goenv ci-setup (recommended)**

```powershell
# ✅ WORKS: goenv handles all escaping for you
$output = goenv ci-setup --shell powershell
Invoke-Expression $output

# This automatically handles:
# - Spaces in $env:USERPROFILE
# - Single quotes in paths (doubled)
# - Double quotes in paths (backtick-escaped)
# - Parentheses and other special chars
```

#### Implementation Details and Testing

The PowerShell quoting logic is implemented and thoroughly tested:

**Implementation:** `cmd/ci-setup.go:213-234` (`outputPowerShell` function)

- Line 218: Single quotes doubled for `$env:GOENV_ROOT` assignment
- Lines 224-226: Double quotes backtick-escaped for `$env:PATH` assignment

**Test Coverage:** `cmd/ci-setup_test.go:306-352` (`TestCISetupPowerShellSpecialCharacters`)

- Verifies escaping for paths with spaces, parentheses, and single quotes
- Includes actual PowerShell execution tests on Windows (lines 117-209)
- Tests common special characters that appear in real Windows environments

**Verified scenarios:**

- `C:\Users\John Smith\.goenv` (spaces)
- `C:\Users\john's folder\.goenv` (single quote)
- `C:\goenv (test)` (parentheses)
- Paths with multiple special characters combined

This robust testing ensures that the PowerShell output from `goenv ci-setup` works correctly across diverse Windows path scenarios encountered in CI environments.

#### GitHub Actions Windows Example

**Option 1: Using `goenv ci-setup --shell powershell` (Recommended)**

This approach uses goenv's built-in PowerShell configuration generator:

```yaml
name: Windows CI

on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install goenv
        shell: powershell
        run: |
          git clone https://github.com/go-nv/goenv.git $env:USERPROFILE\.goenv

          # Use goenv ci-setup to generate properly escaped PowerShell commands
          $output = & "$env:USERPROFILE\.goenv\bin\goenv.exe" ci-setup --shell powershell
          Invoke-Expression $output

          # Persist PATH for subsequent steps
          echo "$env:USERPROFILE\.goenv\bin" | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append
          echo "$env:USERPROFILE\.goenv\shims" | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append

      - name: Cache Go installations
        uses: actions/cache@v3
        with:
          path: ~\.goenv\versions
          key: ${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}

      - name: Install Go versions
        shell: powershell
        run: goenv ci-setup --install --from-file --skip-rehash

      - name: Use Go version
        shell: powershell
        run: goenv use --auto

      - name: Verify Go installation
        shell: powershell
        run: |
          go version
          go env GOROOT

      - name: Run tests
        shell: powershell
        run: go test ./...
```

**Option 2: Manual PATH setup**

If you prefer manual control over PATH configuration:

```yaml
name: Windows CI (Manual PATH)

on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install goenv (Manual)
        shell: powershell
        run: |
          git clone https://github.com/go-nv/goenv.git $env:USERPROFILE\.goenv

          # Manual PATH setup (properly quoted for paths with spaces)
          $env:PATH = "$env:USERPROFILE\.goenv\bin;$env:USERPROFILE\.goenv\shims;$env:PATH"

          # Persist for subsequent steps
          echo "$env:USERPROFILE\.goenv\bin" | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append
          echo "$env:USERPROFILE\.goenv\shims" | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append

      - name: Cache Go installations
        uses: actions/cache@v3
        with:
          path: ~\.goenv\versions
          key: ${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}

      - name: Install Go versions
        run: goenv ci-setup --install --from-file --skip-rehash

      - name: Use Go version
        run: goenv use --auto

      - name: Verify Go installation
        run: |
          go version
          go env GOROOT

      - name: Run tests
        run: go test ./...
```

**Key differences:**
- Option 1 uses `goenv ci-setup --shell powershell` which handles all quoting automatically (see `cmd/ci-setup.go:213-234`)
- Option 2 manually constructs PATH (requires understanding PowerShell quoting rules)
- Both handle spaces and special characters correctly
- Option 1 is recommended for maintainability

#### Azure Pipelines Windows Example

```yaml
trigger:
  - main

pool:
  vmImage: "windows-latest"

steps:
  - checkout: self

  - powershell: |
      git clone https://github.com/go-nv/goenv.git $env:USERPROFILE\.goenv
      $env:PATH = "$env:USERPROFILE\.goenv\bin;$env:USERPROFILE\.goenv\shims;$env:PATH"
      Write-Host "##vso[task.setvariable variable=PATH]$env:PATH"
    displayName: "Install goenv"

  - task: Cache@2
    inputs:
      key: 'goenv | "$(Agent.OS)" | .go-version'
      path: $(USERPROFILE)\.goenv\versions
    displayName: "Cache Go installations"

  - script: goenv ci-setup --install --from-file --skip-rehash
    displayName: "Install Go versions"

  - script: |
      goenv use --auto
      go version
      go test ./...
    displayName: "Test"
```

#### Common Windows CI Issues

| **Problem**                            | **Cause**                            | **Solution**                                                 |
| -------------------------------------- | ------------------------------------ | ------------------------------------------------------------ |
| 🔴 `goenv: command not found`          | PATH not set correctly in PowerShell | Use `$env:PATH = "...;$env:PATH"` (not `export`)             |
| 🔴 Error with paths containing spaces  | Unquoted paths with spaces           | Always quote: `"$env:USERPROFILE\.goenv"`                    |
| 🔴 `goenv ci-setup` output not applied | Not using `Invoke-Expression`        | Use: `Invoke-Expression (goenv ci-setup --shell powershell)` |
| 🔴 Cache key mismatch                  | Forward slashes in Windows paths     | Use backslashes: `~\.goenv\versions`                         |
| 🟡 Slow builds on Windows              | No cache or poor cache key           | Cache `~\.goenv\versions` with version file hash             |

#### Best Practices for Windows CI

1. **Always use PowerShell syntax** for PATH and environment variables
2. **Quote all paths** that might contain spaces (especially `$env:USERPROFILE`)
3. **Use backslashes** in Windows paths for consistency
4. **Set GITHUB_PATH** or equivalent for persistent PATH across steps
5. **Test with `goenv doctor`** to verify environment setup
6. **Use `--force` flag** for cache cleaning (required in non-interactive CI)

## Troubleshooting

### Cache Cleaning Fails in CI

**Error:**

```
Error: non-interactive mode requires --force

This command requires confirmation when run interactively.
Refusing to clean caches in non-interactive environment without --force flag.
In CI/automation, use: goenv cache clean --force
```

**Cause:** Cache cleaning commands require interactive confirmation by default. CI/CD environments are non-interactive (non-TTY), so the command fails to prevent accidental hangs waiting for user input.

**Solution:** Always use `--force` flag in CI:

```bash
# ❌ Wrong - will fail in CI
goenv cache clean build

# ✅ Correct - required for CI
goenv cache clean build --force
goenv cache clean all --older-than 30d --force
goenv cache migrate --force
```

**Why this exists:** Deliberate safety feature to prevent:

- CI pipelines hanging indefinitely waiting for confirmation
- Accidental cache deletion without explicit intent
- Silent failures in automation scripts

**Remember:** `--force` is REQUIRED for any `goenv cache clean` or `goenv cache migrate` command in CI/CD pipelines.

### Cache Not Working

1. Verify cache key includes version file hash
2. Check cache path matches goenv root
3. Ensure goenv root is consistent across runs

### Version Not Found

```bash
# List available versions
goenv install --list

# Install specific version
goenv ci-setup --install 1.23.2

# Check installed versions
goenv versions
```

### Slow Installation

```bash
# Use skip-rehash for faster batch installation
goenv ci-setup --install --from-file --skip-rehash

# Verify cache is working
ls -la ~/.goenv/versions
```

### Permission Issues

```bash
# Ensure goenv root is writable
chmod -R u+w ~/.goenv

# Or use workspace-local goenv root
export GOENV_ROOT=${CI_PROJECT_DIR}/.goenv
```

## Best Practices

1. **Use specific versions**: Avoid `latest` for reproducible builds
2. **Enable caching**: Cache `$GOENV_ROOT/versions` directory
3. **Two-phase installation**: Use `ci-setup --install` for optimal caching
4. **Skip rehashing**: Use `--skip-rehash` for multiple installations
5. **Set GOTOOLCHAIN=local**: Prevent automatic downloads
6. **Version files**: Commit `.go-version` for consistency
7. **⚠️ ALWAYS use `--force` for cache commands**: `goenv cache clean` and `goenv cache migrate` will fail in CI without `--force` flag (non-interactive protection)

## Advanced Examples

### Matrix Builds

Test against multiple Go versions:

```yaml
# GitHub Actions
jobs:
  test:
    strategy:
      matrix:
        go-version: [1.21.0, 1.22.0, 1.23.2]
    steps:
      - name: Install Go
        run: goenv ci-setup --install ${{ matrix.go-version }} --skip-rehash
      - name: Use Go
        run: goenv use ${{ matrix.go-version }} --global
      - name: Test
        run: go test ./...
```

### Conditional Installation

Install only if not cached:

```yaml
- name: Check cache
  id: cache
  uses: actions/cache@v3
  with:
    path: ~/.goenv/versions
    key: goenv-${{ hashFiles('.go-version') }}

- name: Install Go
  if: steps.cache.outputs.cache-hit != 'true'
  run: goenv ci-setup --install --from-file --skip-rehash
```

### Multi-Version Project

Install multiple versions for compatibility testing:

```bash
# Install all required versions at once
goenv ci-setup --install 1.21.0 1.22.0 1.23.2 --skip-rehash

# Test each version
for version in 1.21.0 1.22.0 1.23.2; do
  goenv use $version --global
  go test ./...
done
```

## Generating SBOMs in CI

Software Bill of Materials (SBOM) generation for compliance and vulnerability scanning.

### Prerequisites

Install an SBOM tool via goenv:

```yaml
- name: Install SBOM tool
  run: goenv tools install cyclonedx-gomod@v1.6.0
```

### Basic SBOM Generation

```yaml
- name: Setup Go
  run: |
    goenv ci-setup --install --from-file
    goenv use --auto

- name: Generate SBOM
  run: goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json --output=sbom.json

- name: Upload SBOM
  uses: actions/upload-artifact@v4
  with:
    name: sbom
    path: sbom.json
```

### SPDX Format with Syft

```yaml
- name: Install syft
  run: goenv tools install syft@v1.0.0

- name: Generate SPDX SBOM
  run: goenv sbom project --tool=syft --format=spdx-json --output=sbom.spdx.json
```

### Container Image SBOM

```yaml
- name: Generate container SBOM
  run: goenv sbom project --tool=syft --image=ghcr.io/${{ github.repository }}:${{ github.sha }}
```

### Benefits

- **Reproducible**: Uses pinned Go and SBOM tool versions
- **Cache-isolated**: Respects per-version cache isolation
- **CI-friendly**: Provenance metadata to stderr, SBOM to stdout/file
- **Exit codes**: Preserves tool exit codes for pipeline integration

### Vulnerability Scanning

Combine with vulnerability scanners:

```yaml
- name: Generate SBOM
  run: goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json --output=sbom.json

- name: Scan SBOM for vulnerabilities
  uses: anchore/scan-action@v3
  with:
    sbom: sbom.json
```

## Health Checks and Validation

The `goenv doctor` command is a **first-class CI feature** designed specifically for validation in automated pipelines. It provides machine-readable JSON output, structured exit codes, and comprehensive environment diagnostics.

### Why Use Doctor in CI?

Traditional CI approaches rely on manual verification (`go version`, `which go`) which can miss configuration issues. The `doctor` command validates:

- Runtime environment (containers, WSL, native)
- Filesystem compatibility (NFS, SMB, Docker mounts)
- PATH and shim configuration
- Cache isolation and architecture mismatches
- GOTOOLCHAIN conflicts
- Cross-compilation support
- 24+ comprehensive checks

**Key Features for CI/CD:**

- **Exit codes**: 0 (success), 1 (errors), 2 (warnings) - perfect for pipeline control
- **JSON output**: Machine-readable with stable check IDs for parsing
- **Fail-on levels**: `--fail-on=error` or `--fail-on=warning` for strict validation
- **Fast**: Completes in <1 second, negligible CI time impact

### Quick Start - Add to Your Pipeline

Add one line after `goenv use`:

```yaml
# GitHub Actions
- name: Validate goenv setup
  run: goenv doctor --json --fail-on=error

# GitLab CI
- goenv doctor --json --fail-on=error

# CircleCI
- run: goenv doctor --json --fail-on=error
```

This catches 90% of configuration issues before they cause cryptic build failures.

### Basic Health Check

```yaml
# GitHub Actions
- name: Validate goenv installation
  run: goenv doctor
```

### JSON Output for Parsing

The `--json` flag provides machine-readable output perfect for CI/CD automation:

```yaml
- name: Check goenv health (JSON)
  run: goenv doctor --json
```

**JSON Output Example:**

```json
{
  "schema_version": "1",
  "checks": [
    {
      "id": "runtime-environment",
      "name": "Runtime environment",
      "status": "ok",
      "message": "Running in GitHub Actions"
    },
    {
      "id": "path-configuration",
      "name": "PATH configuration",
      "status": "error",
      "message": "Shims directory not in PATH",
      "advice": "Add goenv shims to PATH: export PATH=\"$GOENV_ROOT/shims:$PATH\""
    }
  ],
  "summary": {
    "total": 24,
    "ok": 22,
    "warnings": 1,
    "errors": 1
  }
}
```

### Exit Codes

The `doctor` command uses distinct exit codes to help CI systems differentiate between errors and warnings:

| Exit Code | Meaning  | Description                                                     |
| --------- | -------- | --------------------------------------------------------------- |
| `0`       | Success  | No issues found, or only warnings when `--fail-on=error`        |
| `1`       | Errors   | Critical issues found that prevent goenv from working correctly |
| `2`       | Warnings | Non-critical issues found when `--fail-on=warning` is set       |

### Fail-On Levels

Control when the doctor command should fail using `--fail-on`:

```yaml
# Fail only on critical errors (default)
- name: Check goenv health
  run: goenv doctor --json --fail-on=error

# Strict mode - fail on any warning
- name: Check goenv health (strict)
  run: goenv doctor --json --fail-on=warning
```

### GitHub Actions Examples

**Basic validation with error handling:**

```yaml
name: Validate goenv

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up goenv
        run: |
          git clone https://github.com/go-nv/goenv.git ~/.goenv
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
          echo "$HOME/.goenv/shims" >> $GITHUB_PATH

      - name: Install Go
        run: goenv ci-setup --install --from-file

      - name: Health check
        run: goenv doctor --json --fail-on=error
```

**Complete workflow with strict validation and JSON parsing:**

```yaml
name: Go CI with Doctor Validation

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up goenv
        run: |
          git clone https://github.com/go-nv/goenv.git ~/.goenv
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
          echo "$HOME/.goenv/shims" >> $GITHUB_PATH

      - name: Cache Go installations
        uses: actions/cache@v3
        with:
          path: ~/.goenv/versions
          key: ${{ runner.os }}-goenv-${{ hashFiles('.go-version') }}

      - name: Install Go
        run: goenv ci-setup --install --from-file --skip-rehash

      - name: Set Go version
        run: goenv use --auto

      - name: Validate with doctor (strict)
        run: goenv doctor --json --fail-on=warning

      - name: Run tests
        run: go test -v ./...
```

**Parse JSON output and create GitHub annotations:**

```yaml
- name: Health check with annotations
  id: doctor
  run: |
    goenv doctor --json > doctor-report.json || true

- name: Parse doctor report
  if: always()
  run: |
    # Create GitHub annotations from doctor output
    jq -r '.checks[] | select(.status=="error") | "::error ::\(.name): \(.message)"' doctor-report.json
    jq -r '.checks[] | select(.status=="warning") | "::warning ::\(.name): \(.message)"' doctor-report.json

    # Display summary
    echo "### Doctor Summary" >> $GITHUB_STEP_SUMMARY
    jq -r '"- Total checks: \(.summary.total)"' doctor-report.json >> $GITHUB_STEP_SUMMARY
    jq -r '"- OK: \(.summary.ok)"' doctor-report.json >> $GITHUB_STEP_SUMMARY
    jq -r '"- Warnings: \(.summary.warnings)"' doctor-report.json >> $GITHUB_STEP_SUMMARY
    jq -r '"- Errors: \(.summary.errors)"' doctor-report.json >> $GITHUB_STEP_SUMMARY

    # Fail if errors found
    if jq -e '.summary.errors > 0' doctor-report.json; then
      echo "::error ::goenv doctor found critical errors"
      exit 1
    fi

- name: Upload doctor report
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: doctor-report
    path: doctor-report.json
```

**Handle different exit codes:**

```yaml
- name: Health check (warnings allowed)
  run: |
    if ! goenv doctor --json --fail-on=error; then
      EXIT_CODE=$?
      if [ $EXIT_CODE -eq 1 ]; then
        echo "::error ::Critical errors found in goenv configuration"
        exit 1
      elif [ $EXIT_CODE -eq 2 ]; then
        echo "::warning ::Warnings found in goenv configuration"
        # Continue execution - warnings don't fail the build
        exit 0
      fi
    fi
```

**Matrix builds with per-OS validation:**

```yaml
name: Multi-OS Testing

on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ["1.23.2", "1.22.5"]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - name: Set up goenv
        shell: bash
        run: |
          if [ "$RUNNER_OS" == "Windows" ]; then
            git clone https://github.com/go-nv/goenv.git $HOME/.goenv
            echo "$HOME/.goenv/bin" >> $GITHUB_PATH
            echo "$HOME/.goenv/shims" >> $GITHUB_PATH
          else
            git clone https://github.com/go-nv/goenv.git ~/.goenv
            echo "$HOME/.goenv/bin" >> $GITHUB_PATH
            echo "$HOME/.goenv/shims" >> $GITHUB_PATH
          fi

      - name: Install Go ${{ matrix.go }}
        run: goenv ci-setup --install ${{ matrix.go }}

      - name: Set Go version
        run: goenv use ${{ matrix.go }} --global

      - name: Validate (${{ runner.os }})
        run: goenv doctor --json --fail-on=error

      - name: Run tests
        run: go test ./...
```

### GitLab CI Examples

**Allow warnings (exit code 2) to pass:**

```yaml
check-goenv:
  script:
    - goenv doctor --json --fail-on=warning
  # Allow exit code 2 (warnings) to pass
  allow_failure:
    exit_codes: [2]
```

**Strict validation (fail on warnings):**

```yaml
validate-strict:
  script:
    - goenv doctor --json --fail-on=warning
  # No allow_failure - both errors (1) and warnings (2) fail the job
```

**Parse JSON and create custom reports:**

```yaml
check-health:
  script:
    - goenv doctor --json > doctor-report.json || true
    - |
      # Generate summary
      jq -r '"Total checks: \(.summary.total), OK: \(.summary.ok), Warnings: \(.summary.warnings), Errors: \(.summary.errors)"' doctor-report.json

    # Fail if errors found
    - |
      if [ $(jq '.summary.errors' doctor-report.json) -gt 0 ]; then
        echo "❌ Critical errors found"
        exit 1
      fi
  artifacts:
    reports:
      json: doctor-report.json
    paths:
      - doctor-report.json
    expire_in: 1 week
```

### Circle CI Example

```yaml
version: 2.1

jobs:
  validate:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install goenv
          command: |
            git clone https://github.com/go-nv/goenv.git ~/.goenv
            echo 'export PATH="$HOME/.goenv/bin:$PATH"' >> $BASH_ENV
            echo 'export PATH="$HOME/.goenv/shims:$PATH"' >> $BASH_ENV

      - run:
          name: Health check
          command: |
            goenv doctor --json --fail-on=error || {
              EXIT_CODE=$?
              if [ $EXIT_CODE -eq 1 ]; then
                echo "Critical errors found"
                exit 1
              elif [ $EXIT_CODE -eq 2 ]; then
                echo "Warnings found (allowed)"
                exit 0
              fi
            }
```

### Best Practices

1. **Run early**: Check health after goenv setup, before installing Go versions
2. **Use JSON in CI**: Parse structured output for better error handling
3. **Set appropriate fail-on level**:
   - `--fail-on=error` for most CI pipelines (default)
   - `--fail-on=warning` for strict validation in critical environments
4. **Cache reports**: Save JSON output as artifacts for debugging
5. **Create annotations**: Convert errors/warnings to CI-specific annotations
6. **Handle exit codes**: Differentiate between errors (1) and warnings (2)

## See Also

- [Environment Variables Reference](reference/ENVIRONMENT_VARIABLES.md)
- [Commands Reference](reference/COMMANDS.md)
- [Smart Caching Strategy](advanced/SMART_CACHING.md)
- [SBOM Command Documentation](reference/COMMANDS.md#goenv-sbom)
- [Doctor Command Documentation](reference/COMMANDS.md#goenv-doctor)
