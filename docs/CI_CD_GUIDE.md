# CI/CD Integration Guide

This guide explains how to optimize goenv for CI/CD pipelines using the two-phase installation strategy.

## Overview

The `goenv ci-setup` command provides two modes:

1. **Environment Setup** (default): Outputs environment variables for CI configuration
2. **Two-Phase Installation** (`--install`): Optimizes caching by separating Go installation from usage

## Why Two-Phase Installation?

Traditional CI approach:

```bash
# Install and use in one step - harder to cache effectively
goenv install 1.23.2
goenv global 1.23.2
```

Two-phase approach (optimized):

```bash
# Phase 1: Install Go versions (cached separately)
goenv ci-setup --install 1.23.2

# Phase 2: Use the cached version (fast)
goenv global 1.23.2
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

      - name: Verify Go installation
        run: |
          go version
          which go

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

test:
  script:
    - go version
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

script:
  - go version
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
goenv global 1.23.2
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

- `~/.goenv/versions` - Installed Go versions
- `~/.cache/go-build` - Build cache (optional)
- `~/go/pkg/mod` - Module cache (optional)

## Cache Cleaning in CI

> **🚨 CRITICAL: Always use `--force` in CI environments**
>
> Cache cleaning commands require interactive confirmation by default. **Without `--force`, the command will fail in non-interactive CI/CD pipelines** with an error like:
>
> ```
> Error: cache cleaning requires --force flag in non-interactive environments
> cannot prompt for confirmation: not a terminal
> ```
>
> **All cache clean examples below use `--force` - this is mandatory for CI!**

When managing cache sizes or cleaning up between builds, the `--force` flag prevents CI hangs on confirmation prompts.

### Clean Build Caches

```bash
# Clean all build caches (non-interactive) - REQUIRES --force
goenv cache clean build --force

# Clean build caches older than 30 days
goenv cache clean build --older-than 30d --force

# Clean all caches (build + module)
goenv cache clean all --force

# Preview what would be deleted (dry-run doesn't need --force)
goenv cache clean all --older-than 30d --dry-run
```

### Size-Based Cleanup

```bash
# Keep only 1GB of build caches (delete oldest first)
goenv cache clean build --max-bytes 1GB --force

# Keep 500MB total across all caches
goenv cache clean all --max-bytes 500MB --force
```

### Best Practices

- **Always use `--force` in CI**: Required in non-interactive CI environments (will fail without it)
- **Use `--older-than`**: Clean caches by age rather than deleting all
- **Test with `--dry-run`**: Preview cleanup before applying (doesn't require `--force`)
- **Monitor cache sizes**: Use `goenv cache status --fast` for quick checks

**Example error without `--force` in CI:**

```
$ goenv cache clean build
Error: cache cleaning requires --force flag in non-interactive environments
Run with --force to skip confirmation, or --dry-run to preview changes
Exit code: 1
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

## Windows CI Environments

### PowerShell PATH and Quoting Gotchas

Windows CI environments (GitHub Actions, Azure Pipelines) use PowerShell, which has specific quoting rules that differ from bash/sh.

#### Safe PATH Modification

```powershell
# ✅ CORRECT: Append to PATH safely (handles spaces correctly)
$env:PATH = "$env:USERPROFILE\.goenv\bin;$env:USERPROFILE\.goenv\shims;$env:PATH"

# ✅ CORRECT: Using goenv ci-setup for PowerShell
$output = goenv ci-setup --shell powershell
Invoke-Expression $output

# ❌ WRONG: Don't use bare variables with spaces
$env:PATH = $HOME\.goenv\bin;$env:PATH  # Breaks if $HOME has spaces
```

#### PowerShell String Quoting Rules

PowerShell has unique quoting requirements for arguments with special characters:

| **Character**    | **Bash/sh**        | **PowerShell**                     | **Example**                  |
| ---------------- | ------------------ | ---------------------------------- | ---------------------------- |
| Space            | `'arg with space'` | `'arg with space'`                 | `goenv install 'go 1.21'`    |
| Single quote `'` | `"can't"`          | `'can''t'` (double the quote)      | `'version''s name'`          |
| Double quote `"` | `'say "hi"'`       | `'say `"hi`"'` (backtick escape)   | `'my `"quoted`" arg'`        |
| Backtick `` ` `` | `` 'tick`' ``      | ` 'tick``' ` (double the backtick) | Not common in goenv usage    |
| Env variable     | `"$HOME/path"`     | `"$env:HOME\path"`                 | `"$env:GOENV_ROOT\versions"` |

#### GitHub Actions Windows Example

```yaml
name: Windows CI

on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install goenv (PowerShell)
        shell: powershell
        run: |
          git clone https://github.com/go-nv/goenv.git $env:USERPROFILE\.goenv
          $env:PATH = "$env:USERPROFILE\.goenv\bin;$env:USERPROFILE\.goenv\shims;$env:PATH"
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
        run: goenv global ${{ matrix.go-version }}
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
  goenv global $version
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

Use `goenv doctor` to validate your CI/CD environment setup and catch configuration issues early.

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

**Parse JSON output and create annotations:**

```yaml
- name: Health check with annotations
  id: doctor
  run: |
    goenv doctor --json > doctor-report.json || true

- name: Parse doctor report
  if: always()
  run: |
    # Extract errors and warnings
    jq -r '.checks[] | select(.status=="error") | "::error ::\(.name): \(.message)"' doctor-report.json
    jq -r '.checks[] | select(.status=="warning") | "::warning ::\(.name): \(.message)"' doctor-report.json

    # Fail if errors found
    if jq -e '.summary.errors > 0' doctor-report.json; then
      echo "::error ::goenv doctor found critical errors"
      exit 1
    fi
```

**Allow warnings but fail on errors:**

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
