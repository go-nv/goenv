# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument.

## Global Flags & Output Options

### Machine-Readable Output

For automation, CI/CD, or when stdout is not a TTY, goenv automatically suppresses emojis and colors. You can also explicitly control output formatting:

**Environment Variables:**
```bash
# Suppress all colors (https://no-color.org/ standard)
NO_COLOR=1 goenv doctor

# Force plain output (no emojis, no colors)
goenv --plain version
```

**Global Flags:**
```bash
--plain         # Plain output (no emojis, no colors)
--no-color      # Disable colored output only
--json          # JSON output (supported by: list, versions, doctor, cache status, inventory, vscode)
```

**Automatic Detection:**
- When output is piped (`goenv list | grep 1.25`), emojis are automatically suppressed
- When `NO_COLOR` environment variable is set, colors and emojis are disabled
- JSON output is always machine-readable (use `--json` where supported)

**Examples:**
```bash
# CI/CD usage
NO_COLOR=1 goenv doctor --json > report.json

# Piped output (automatic)
goenv list | grep -E "1\.(24|25)"

# Force plain output
goenv --plain install 1.25.2

# JSON for automation
goenv list --json | jq -r '.versions[] | select(.active) | .version'
```

## 🚀 Modern Unified Commands (Recommended)

**New in v3.0**: Simplified, intuitive commands for common operations:

- **[`goenv use`](#goenv-use)** - Set Go version (replaces `local`/`global`)
- **[`goenv current`](#goenv-current)** - Show active version (replaces `version`)
- **[`goenv list`](#goenv-list)** - List versions (replaces `versions`/`installed`)

These unified commands provide a cleaner, more consistent interface. The legacy commands (`local`, `global`, `versions`) still work for backward compatibility.

## 📋 All Subcommands

- [Command Reference](#command-reference)
  - [🚀 Modern Unified Commands (Recommended)](#-modern-unified-commands-recommended)
  - [📋 All Subcommands](#-all-subcommands)
  - [`goenv use`](#goenv-use)
  - [`goenv current`](#goenv-current)
  - [`goenv list`](#goenv-list)
  - [`goenv alias`](#goenv-alias)
  - [`goenv cache`](#goenv-cache)
    - [`goenv cache status`](#goenv-cache-status)
    - [`goenv cache clean`](#goenv-cache-clean)
    - [`goenv cache migrate`](#goenv-cache-migrate)
    - [`goenv cache info`](#goenv-cache-info)
  - [`goenv ci-setup`](#goenv-ci-setup)
  - [`goenv commands`](#goenv-commands)
  - [`goenv completions`](#goenv-completions)
  - [`goenv tools`](#goenv-tools)
  - [`goenv tools default`](#goenv-tools-default)
  - [`goenv doctor`](#goenv-doctor)
  - [`goenv exec`](#goenv-exec)
  - [`goenv global`](#goenv-global)
  - [`goenv help`](#goenv-help)
  - [`goenv init`](#goenv-init)
  - [`goenv install`](#goenv-install)
    - [Options](#options)
    - [Auto-Rehash](#auto-rehash)
  - [`goenv installed`](#goenv-installed)
  - [`goenv inventory`](#goenv-inventory)
  - [`goenv local`](#goenv-local)
    - [`goenv local` (advanced)](#goenv-local-advanced)
  - [`goenv tools sync`](#goenv-tools-sync)
    - [Options](#options-1)
  - [`goenv prefix`](#goenv-prefix)
  - [`goenv refresh`](#goenv-refresh)
  - [`goenv rehash`](#goenv-rehash)
  - [`goenv root`](#goenv-root)
  - [`goenv sbom`](#goenv-sbom)
  - [`goenv shell`](#goenv-shell)
  - [`goenv shims`](#goenv-shims)
  - [`goenv unalias`](#goenv-unalias)
  - [`goenv uninstall`](#goenv-uninstall)
  - [`goenv update`](#goenv-update)
    - [Options](#options-2)
    - [Installation Methods](#installation-methods)
  - [`goenv tools update`](#goenv-tools-update)
    - [Options](#options-3)
  - [`goenv vscode`](#goenv-vscode)
    - [`goenv vscode init`](#goenv-vscode-init)
  - [`goenv version`](#goenv-version)
  - [`goenv --version`](#goenv---version)
  - [`goenv version-file`](#goenv-version-file)
  - [`goenv version-file-read`](#goenv-version-file-read)
  - [`goenv version-file-write`](#goenv-version-file-write)
  - [`goenv version-name`](#goenv-version-name)
  - [`goenv version-origin`](#goenv-version-origin)
  - [`goenv versions`](#goenv-versions)
    - [Options](#options-4)
  - [`goenv whence`](#goenv-whence)
  - [Version Discovery & Precedence](#version-discovery--precedence)
    - [Version Sources](#version-sources)
    - [Smart Precedence Rules](#smart-precedence-rules)
    - [Examples](#examples)
    - [Best Practices](#best-practices)
  - [`goenv which`](#goenv-which)

## `goenv alias`

Create and manage version aliases. Aliases provide convenient shorthand names for Go versions, making it easier to reference commonly used versions.

**Usage:**

```shell
# List all aliases
> goenv alias

# Show specific alias
> goenv alias stable
1.23.0

# Create or update an alias
> goenv alias stable 1.23.0
> goenv alias dev latest
> goenv alias lts 1.22.5
```

**Using aliases:**

Once created, aliases can be used anywhere a version number is expected:

```shell
# Set global version using alias
> goenv use stable --global

# Set local version using alias
> goenv use dev

# Aliases are resolved to their target versions
> goenv current
1.23.0 (set by /home/go-nv/.goenv/version)
```

**Alias features:**

- Aliases are stored in `~/.goenv/aliases` and persist across sessions
- Alias names cannot conflict with reserved keywords (`system`, `latest`)
- Aliases must contain only alphanumeric characters, hyphens, and underscores
- Aliases are automatically resolved when setting versions with `use`
- You can create aliases that point to special versions like `latest` or `system`

**Common use cases:**

```shell
# Track LTS versions
> goenv alias lts 1.22.5
> goenv use lts --global

# Maintain development and stable versions
> goenv alias stable 1.23.0
> goenv alias dev 1.24rc1

# Quick version names
> goenv alias prod 1.23.2
> goenv alias staging 1.23.0
```

## `goenv cache`

Manage build and module caches for installed Go versions. Goenv uses architecture-aware caching to prevent conflicts when working across multiple platforms or architectures.

**Subcommands:**

- **[`goenv cache status`](#goenv-cache-status)** - Show cache sizes and locations
- **[`goenv cache clean`](#goenv-cache-clean)** - Clean build or module caches
- **[`goenv cache migrate`](#goenv-cache-migrate)** - Migrate old format caches
- **[`goenv cache info`](#goenv-cache-info)** - Show CGO toolchain information

### `goenv cache status`

Display detailed information about build and module caches across all installed Go versions.

**Usage:**

```shell
# Show cache status
goenv cache status

# Machine-readable JSON output
goenv cache status --json
```

**Example output:**

```shell
$ goenv cache status
🏗️  Build Caches:
  Go 1.23.2   (darwin-arm64):   1.24 GB, 3,421 files
  Go 1.23.2   (linux-amd64):    0.89 GB, 2,103 files
  Go 1.24.4   (darwin-arm64):   0.56 GB, 1,234 files

📦 Module Caches:
  Go 1.23.2:  0.34 GB, 456 modules/files
  Go 1.24.4:  0.28 GB, 389 modules/files

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Build Cache:  2.69 GB (6,758 files)
Total Module Cache: 0.62 GB (845 items)
Total:              3.31 GB

📍 Cache Locations:
  GOENV_ROOT: /Users/username/.goenv
  Versions:   /Users/username/.goenv/versions

💡 Tips:
  • Clean build caches:  goenv cache clean build
  • Clean module caches: goenv cache clean mod
  • Clean all caches:    goenv cache clean all
```

**What it shows:**

- Build cache size per Go version and architecture
- Module cache size per Go version
- Total disk usage
- Cache locations
- Old format caches that should be migrated

### `goenv cache clean`

Clean build or module caches to reclaim disk space or fix cache-related issues.

**Usage:**

```shell
# Clean all build caches
goenv cache clean build

# Clean all module caches
goenv cache clean mod

# Clean everything
goenv cache clean all

# Preview what would be deleted (dry-run)
goenv cache clean build --dry-run
goenv cache clean all --older-than 30d --dry-run

# Clean specific version only
goenv cache clean build --version 1.23.2

# Clean old format caches
goenv cache clean build --old-format

# Prune caches by age (delete older than threshold)
goenv cache clean build --older-than 30d   # 30 days
goenv cache clean build --older-than 1w    # 1 week
goenv cache clean all --older-than 24h     # 24 hours

# Prune caches by size (keep newest, delete oldest)
goenv cache clean build --max-bytes 1GB    # Keep only 1GB
goenv cache clean all --max-bytes 500MB    # Keep only 500MB

# Skip confirmation
goenv cache clean all --force
```

**Options:**

- `--dry-run, -n` - Show what would be cleaned without actually cleaning
- `--version <version>` - Clean caches for specific Go version only
- `--old-format` - Clean old format caches only (non-architecture-aware)
- `--older-than <duration>` - Delete caches older than duration (e.g., 30d, 1w, 24h)
- `--max-bytes <size>` - Keep only this much cache, delete oldest (e.g., 1GB, 500MB)
- `--force, -f` - Skip confirmation prompt

**Examples:**

```shell
# Preview cleanup before doing it
$ goenv cache clean build --dry-run
🧹 Caches to clean:

  Go 1.23.2   [darwin-arm64]:  1.24 GB (3,421 files)
  Go 1.23.2   [linux-amd64]:   0.89 GB (2,103 files)
  Go 1.24.4   [darwin-arm64]:  0.56 GB (1,234 files)

Total to clean: 2.69 GB (6,758 files)

🔍 Dry-run mode: No caches were actually deleted.
💡 Run without --dry-run to perform the cleanup.

# Clean old caches to free space
$ goenv cache clean build --older-than 30d
🔍 Pruning Summary:
   • Keeping 2 cache(s) (newer than 30d)
   • Deleting 1 cache(s) (older than 30d)

🧹 Caches to clean:

  Go 1.22.0   [darwin-arm64]:  0.45 GB (1,123 files)

Total to clean: 0.45 GB (1,123 files)

Proceed with cleaning? [y/N]: y
Cleaning...
✓ Cleaned Go 1.22.0 [darwin-arm64]
✓ Done! Cleaned 0.45 GB (1,123 files)

# Keep only 1GB of build caches
$ goenv cache clean build --max-bytes 1GB --force
🔍 Pruning Summary:
   • Keeping 1 cache(s) (newest caches within 1GB limit)
   • Deleting 2 cache(s) (oldest caches exceeding limit)

Cleaning...
✓ Cleaned Go 1.22.0 [darwin-arm64]
✓ Cleaned Go 1.23.2 [linux-amd64]
✓ Done! Cleaned 1.34 GB (3,226 files)
```

**When to use:**

- Free up disk space from build caches
- Fix cache corruption or "inconsistency in file cache" errors
- After cross-compiling for multiple architectures
- Clean up old versions you no longer use
- Remove old format caches after migration

**Note:** For diagnostic information about caches, use [`goenv cache status`](#goenv-cache-status) or [`goenv doctor`](#goenv-doctor)

### `goenv cache migrate`

Migrate old format build caches to architecture-aware format. This is useful when upgrading from older goenv versions that didn't isolate caches by architecture.

**Usage:**

```shell
# Migrate all old format caches
goenv cache migrate

# Skip confirmation prompt
goenv cache migrate --force
```

**What it does:**

1. Detects old format caches (`go-build` directories)
2. Moves them to architecture-specific directories (`go-build-{GOOS}-{GOARCH}`)
3. Prevents cache conflicts between architectures
4. Safe to run multiple times (idempotent)

**Example:**

```shell
$ goenv cache migrate
🔄 Migrating old format caches to architecture-aware format...

Found old format caches:
  Go 1.23.2: go-build → go-build-darwin-arm64

Migrate these caches? [y/N]: y
Migrating...
✓ Migrated Go 1.23.2
✓ Done! Migrated 1 cache(s)
```

**When to run migrate:**

- **After upgrading from bash goenv** - Convert old caches to new format
- **When seeing "exec format error"** - Often caused by architecture mismatch in old caches
- **Before cross-compiling** - Ensures each architecture has isolated cache
- **When doctor reports old cache format** - `goenv doctor` will detect and recommend migration

**Safe to run anytime:**
- Idempotent (safe to run multiple times)
- Non-destructive (keeps your cache data)
- Interactive confirmation (unless `--force`)

**After migration:**
- Old `go-build` directories are renamed to `go-build-{GOOS}-{GOARCH}`
- Existing data is preserved
- Future builds use architecture-aware caching automatically

See [Smart Caching](../advanced/SMART_CACHING.md#cache-migration) for details on cache format and migration process.

### `goenv cache info`

Show CGO toolchain configuration for build caches. This helps understand which C compiler and flags were used when creating caches.

**Usage:**

```shell
# Show info for all versions
goenv cache info

# Show info for specific version
goenv cache info 1.23.2

# Machine-readable JSON output
goenv cache info --json
```

**Example output:**

```shell
$ goenv cache info
🔧 CGO Toolchain Information:

Go 1.23.2 (darwin-arm64):
  CGO_ENABLED: 1
  CC:          clang
  CXX:         clang++
  CFLAGS:      -O2 -g
  Platform:    darwin/arm64

Go 1.23.2 (linux-amd64):
  CGO_ENABLED: 1
  CC:          gcc
  CXX:         g++
  CFLAGS:      -O2 -g
  Platform:    linux/amd64
```

**When to use:**

- Debug why different cache directories exist
- Verify CGO configuration for cross-compilation
- Understand cache isolation strategy
- Troubleshoot CGO-related build issues

**Architecture-Aware Caching:**

Goenv automatically creates separate build caches for each OS/architecture combination:

- `go-build-darwin-arm64` - macOS Apple Silicon
- `go-build-darwin-amd64` - macOS Intel
- `go-build-linux-amd64` - Linux x86_64
- `go-build-linux-arm64` - Linux ARM64
- `go-build-windows-amd64` - Windows x86_64

This prevents conflicts when:

- Cross-compiling for different platforms
- Using the same goenv installation from multiple machines (via NFS/network drives)
- Switching between native and emulated architectures (e.g., Rosetta 2)

**See also:**

- [What NOT to Sync](../advanced/WHAT_NOT_TO_SYNC.md) - Cache sync safety guide
- [Cross-Building](../advanced/CROSS_BUILDING.md) - Platform-specific build strategies

## `goenv ci-setup`

Configures goenv for optimal CI/CD usage with two-phase caching optimization.

**Two modes:**

1. **Environment Setup (default)**: Outputs environment variables and recommendations
2. **Two-Phase Installation**: Separates Go installation (cacheable) from usage (fast)

**Basic environment setup:**

```shell
# Output environment variables
> goenv ci-setup

# With verbose recommendations
> goenv ci-setup --verbose

# Specific shell format
> goenv ci-setup --shell github    # GitHub Actions
> goenv ci-setup --shell gitlab    # GitLab CI
> goenv ci-setup --shell bash      # Bash/Zsh
> goenv ci-setup --shell fish      # Fish shell
```

**Two-phase installation (recommended for CI):**

```shell
# Install specific versions (Phase 1 - cacheable)
> goenv ci-setup --install 1.23.2 1.22.5

# Install from .go-version file
> goenv ci-setup --install --from-file

# Install without rehashing (faster for multiple versions)
> goenv ci-setup --install --from-file --skip-rehash

# After installation, use the version (Phase 2 - fast)
> goenv global 1.23.2
> goenv use --auto
```

**Supported version files:**

- `.go-version` - Simple version file
- `.tool-versions` - ASDF-compatible format
- `go.mod` - Go module file

**Example CI pipeline:**

```yaml
# GitHub Actions
- name: Cache Go installations
  uses: actions/cache@v3
  with:
    path: ~/.goenv/versions
    key: goenv-${{ hashFiles('.go-version') }}

- name: Install Go versions
  run: goenv ci-setup --install --from-file --skip-rehash

- name: Use Go version
  run: goenv use --auto
```

See the [CI/CD Integration Guide](../CI_CD_GUIDE.md) for detailed examples and best practices.

## `goenv commands`

Lists all available goenv commands.

**Filtering commands:**

```shell
# List all commands
> goenv commands

# List only shell commands (sh-*)
> goenv commands --sh

# List only non-shell commands
> goenv commands --no-sh
```

## `goenv completions`

Provides auto-completion for itself and other commands by calling them with `--complete`.

## `goenv tools`

Manage Go tools on a per-version basis. Ensures tools are properly isolated per Go version and prevents accidental global installations.

**Subcommands:**

- `goenv tools install` - Install a tool for the current Go version
- `goenv tools list` - List installed tools for the current version
- `goenv tools update` - Update installed tools to latest versions
- `goenv tools sync` - Copy tools from one version to another
- `goenv tools default` - Manage automatic tool installation

**Quick Start:**

```shell
# Install a tool for current Go version
goenv tools install golang.org/x/tools/cmd/goimports@latest

# List tools
goenv tools list

# Update all tools
goenv tools update

# Sync tools from one version to another
goenv tools sync 1.23.2 1.24.4
```

See the detailed sections below for each subcommand.

## `goenv tools default`

Manages the list of tools automatically installed with each new Go version.

Default tools are specified in `~/.goenv/default-tools.yaml` and are automatically installed after each `goenv install` command completes successfully.

**Subcommands:**

```shell
# List configured default tools
> goenv tools default list

# Initialize default tools configuration with sensible defaults
> goenv tools default init

# Enable automatic tool installation
> goenv tools default enable

# Disable automatic tool installation
> goenv tools default disable

# Install default tools for a specific Go version
> goenv tools default install 1.25.2
```

**Common use cases:**

- Auto-install gopls, golangci-lint, staticcheck, delve
- Ensure consistent development environment across Go versions
- Reduce manual setup after installing new Go versions

**Configuration file example (`~/.goenv/default-tools.yaml`):**

```yaml
enabled: true
tools:
  - golang.org/x/tools/gopls@latest
  - github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  - honnef.co/go/tools/cmd/staticcheck@latest
  - github.com/go-delve/delve/cmd/dlv@latest
```

## `goenv doctor`

Diagnose goenv installation and configuration issues.

```shell
> goenv doctor
🔍 Checking goenv installation...

✓ goenv binary
  Location: /Users/user/.goenv/bin/goenv

✓ Shell configuration
  Found eval in ~/.zshrc

✓ PATH setup
  Shims directory is in PATH

✓ Shims directory
  Location: /Users/user/.goenv/shims
  Shim count: 12

✓ Go versions
  3 version(s) installed
  Current: 1.22.5

✓ Configuration complete
```

This command verifies:

- **Runtime environment**: Detects containers (Docker, Kubernetes), WSL, and native systems
- **Filesystem type**: Identifies NFS, SMB, FUSE, bind mounts, and local filesystems
- **goenv binary**: Location and architecture
- **Shell configuration**: Init integration in shell config files
- **PATH setup**: Shims directory placement
- **Shims directory**: Existence and shim count
- **Installed Go versions**: Lists installed versions and current selection
- **Build cache isolation**: Verifies version-specific GOCACHE
- **Cache mount type**: Warns if cache is on NFS, SMB, or Docker bind mounts
- **Cache architecture**: Detects potential "exec format error" issues
- **Rosetta detection** (macOS): Identifies x86_64 vs arm64 architecture mismatches
- **Tool migration**: Recommends syncing tools between versions
- **Common configuration problems**: GOTOOLCHAIN conflicts, PATH order issues

### Flags

- `--json` - Output results in JSON format for CI/automation
- `--fail-on <level>` - Exit with non-zero status on `error` (default) or `warning`

### Exit Codes (for CI/automation)

The `doctor` command uses distinct exit codes to help CI systems differentiate between errors and warnings:

| Exit Code | Meaning  | Description                                                     |
| --------- | -------- | --------------------------------------------------------------- |
| `0`       | Success  | No issues found, or only warnings when `--fail-on=error`        |
| `1`       | Errors   | Critical issues found that prevent goenv from working correctly |
| `2`       | Warnings | Non-critical issues found when `--fail-on=warning` is set       |

**CI/CD Examples:**

```yaml
# GitHub Actions - Fail on errors only (default)
- name: Check goenv health
  run: goenv doctor --json

# GitHub Actions - Fail on warnings (strict mode)
- name: Check goenv health (strict)
  run: goenv doctor --json --fail-on=warning

# GitLab CI - Allow warnings
- name: Check goenv health
  run: goenv doctor
  # Exit code 2 (warnings) won't fail the pipeline
  allow_failure:
    exit_codes: [2]

# Shell script - Handle exit codes
if ! goenv doctor --json; then
  EXIT_CODE=$?
  if [ $EXIT_CODE -eq 1 ]; then
    echo "Errors found"
    exit 1
  elif [ $EXIT_CODE -eq 2 ]; then
    echo "Warnings found"
    # Continue anyway
  fi
fi
```

### JSON Output

The `--json` flag outputs machine-readable JSON with check IDs for automation:

```json
{
  "schema_version": "1",
  "checks": [
    {
      "id": "goenv-binary",
      "name": "goenv binary",
      "status": "ok",
      "message": "Location: /Users/user/.goenv/bin/goenv"
    },
    {
      "id": "shell-configuration",
      "name": "Shell configuration",
      "status": "warning",
      "message": "No shell init found",
      "advice": "Add 'eval \"$(goenv init -)\"' to your ~/.zshrc"
    }
  ],
  "summary": {
    "total": 15,
    "ok": 13,
    "warnings": 2,
    "errors": 0
  }
}
```

**Check IDs** are stable and can be used in CI scripts to filter or assert specific checks.

## `goenv use`

**Modern unified command** for setting Go versions. Replaces `goenv local` and `goenv global` with a cleaner interface.

**Usage:**

```shell
# Set local version (creates .go-version in current directory)
> goenv use 1.25.2

# Set global version (updates ~/.goenv/version)
> goenv use 1.25.2 --global

# Show current version
> goenv use
1.25.2

# Unset local version (removes .go-version)
> goenv use --unset
```

**Options:**

- `--global` - Set global version instead of local
- `--unset` - Remove local .go-version file
- `--vscode` - Also update VS Code settings.json
- `--sync` - Sync go.mod toolchain directive
- `--yes, -y` - Auto-confirm installation prompts (useful for CI/automation)

**Examples:**

```shell
# Set local version for current project
cd my-project
goenv use 1.24.8

# Set global default version
goenv use 1.25.2 --global

# Set version and update VS Code
goenv use 1.24.0 --vscode

# Sync with go.mod toolchain
goenv use 1.24.0 --sync

# CI/automation: auto-install if needed (no prompts)
goenv use 1.25.2 --yes --global

# CI workflow: auto-detect version from .go-version and install
goenv use --yes
```

**CI/Automation Usage:**

When running in CI/CD or non-interactive environments, use `--yes` to skip installation prompts:

```bash
# GitHub Actions / GitLab CI
- name: Setup Go version
  run: goenv use --yes  # Reads .go-version, installs if needed

# Or with explicit version
- name: Setup specific Go version
  run: goenv use 1.25.2 --yes --global

# Docker/container builds
RUN goenv use $(cat .go-version) --yes

# Scripted installations
#!/bin/bash
for version in 1.24.0 1.25.0 1.25.2; do
  goenv use $version --yes --global  # No prompts, auto-install
done
```

**Why use `goenv use`?**

- **Consistent**: One command for both local and global
- **Intuitive**: `use` is clearer than `local`/`global`
- **Modern**: Follows conventions from other version managers
- **Shorter**: `goenv use X` vs `goenv local X`

**Backward compatibility**: The legacy `goenv local` and `goenv global` commands still work but are hidden from help output.

## `goenv current`

**Modern unified command** for showing the active Go version and its source. Replaces `goenv version`.

**Usage:**

```shell
# Show current version with source
> goenv current
1.25.2 (set by /Users/user/project/.go-version)

# Show just the version number
> goenv current --bare
1.25.2

# Show the file that set the version
> goenv current --origin
/Users/user/project/.go-version
```

**Options:**

- `--bare` - Output only the version number
- `--origin` - Output only the source file path

**Examples:**

```shell
# Quick version check
> goenv current
1.24.8 (set by /Users/user/.goenv/version)

# Use in scripts
VERSION=$(goenv current --bare)
echo "Building with Go $VERSION"

# Check where version is set
> goenv current --origin
/Users/user/my-project/.go-version
```

**Why use `goenv current`?**

- **Clear**: "current" is more intuitive than "version"
- **Consistent**: Matches `goenv use` terminology
- **Modern**: Common pattern in version managers

**Backward compatibility**: The legacy `goenv version` command still works but is hidden from help output.

## `goenv list`

**Modern unified command** for listing Go versions. Replaces `goenv versions` and `goenv installed`.

**Usage:**

```shell
# List installed versions (default)
> goenv list
  1.23.5
  1.24.8
* 1.25.2 (set by /Users/user/.goenv/version)

# List available remote versions
> goenv list --remote
  1.20.0
  1.20.1
  ...
  1.25.2
  1.26rc1

# List only stable versions
> goenv list --remote --stable
  1.23.5
  1.24.8
  1.25.2

# Output bare version numbers
> goenv list --bare
1.23.5
1.24.8
1.25.2
```

**Options:**

- `--remote` - List available versions from remote (not installed)
- `--stable` - Filter to stable releases only (no beta/rc)
- `--bare` - Output only version numbers (no markers or descriptions)
- `--skip-aliases` - Don't show version aliases
- `--json` - Output in JSON format (for installed versions only)

**Output Ordering:**

When using `--remote`, versions are displayed in **oldest-first order** (matching traditional shell behavior). The "go" prefix is automatically stripped from version strings. Pre-release versions (beta, rc) are included by default unless `--stable` is specified.

**Examples:**

```shell
# See what's installed
> goenv list
  1.24.8
* 1.25.2 (set by /Users/user/.go-version)

# Check available versions before installing
> goenv list --remote --stable | tail -5
1.24.6
1.24.7
1.24.8
1.25.1
1.25.2

# Use in scripts
for version in $(goenv list --bare); do
  echo "Testing with Go $version"
  goenv use $version
  go test ./...
done

# JSON output for installed versions
> goenv list --json
{
  "schema_version": "1",
  "versions": [
    {
      "version": "1.24.8",
      "active": false
    },
    {
      "version": "1.25.2",
      "active": true,
      "source": "global"
    }
  ]
}

# JSON output for remote versions
> goenv list --remote --stable --json
{
  "schema_version": "1",
  "remote": true,
  "stable_only": true,
  "versions": [
    "1.23.5",
    "1.24.8",
    "1.25.2"
  ]
}
```

**Why use `goenv list`?**

- **Intuitive**: "list" is clearer than "versions"
- **Unified**: One command for both installed and available
- **Modern**: Shorter and more familiar command name
- **Smart default**: Shows installed by default (most common use case)

**Backward compatibility**: The legacy `goenv versions` and `goenv installed` commands still work but are hidden from help output.

- Common configuration problems

Use this command to troubleshoot issues with goenv.

## `goenv exec`

Run an executable with the selected Go version.

Assuming there's an already installed golang by e.g `goenv install 1.11.1` and
selected by e.g `goenv use 1.11.1 --global`,

```shell
> goenv exec go run main.go
```

## `goenv global`

Sets the global version of Go to be used in all shells by writing
the version name to the `~/.goenv/version` file. This version can be
overridden by an application-specific `.go-version` file, or by
setting the `GOENV_VERSION` environment variable.

```shell
> goenv global 1.5.4

# Showcase
> goenv versions
  system
  * 1.5.4 (set by /Users/go-nv/.goenv/version)

> goenv version
1.5.4 (set by /Users/go-nv/.goenv/version)

> go version
go version go1.5.4 darwin/amd64
```

The special version name `system` tells goenv to use the system Go
(detected by searching your `$PATH`).

When run without a version number, `goenv global` reports the
currently configured global version.

## `goenv help`

Parses and displays help contents from a command's source file.

A command is considered documented if it starts with a comment block
that has a `Summary:` or `Usage:` section. Usage instructions can
span multiple lines as long as subsequent lines are indented.
The remainder of the comment block is displayed as extended
documentation.

```shell
> goenv help help
```

```shell
> goenv help install
```

## `goenv init`

Configure the shell environment for goenv. Must have if you want to integrate `goenv` with your shell.

The following displays how to integrate `goenv` with your user's shell:

```shell
> goenv init
```

Usually it boils down to adding to your `.bashrc` or `.zshrc` the following:

```
eval "$(goenv init -)"
```

## `goenv install`

Install a Go version (using `go-build`). It's required that the version is a known installable definition by `go-build`. Alternatively, supply `latest` as an argument to install the latest version available to goenv.

```shell
> goenv install 1.11.1
```

### Options

- `-f, --force` - Install even if the version appears to be installed already
- `-s, --skip-existing` - Skip if the version appears to be installed already
- `-l, --list` - List all available versions
- `-k, --keep` - Keep source tree after installation
- `-v, --verbose` - Verbose mode: print detailed installation info
- `-q, --quiet` - Quiet mode: disable progress bar
- `-4, --ipv4` - Resolve names to IPv4 addresses only
- `-6, --ipv6` - Resolve names to IPv6 addresses only
- `-g, --debug` - Enable debug output
- `--no-rehash` - Skip automatic rehash after installation (advanced)

### Using a Custom Mirror

For corporate proxies, regional mirrors, or air-gapped environments, you can specify a custom download mirror:

```shell
# Use a custom mirror for one installation
> GO_BUILD_MIRROR_URL=https://mirrors.example.com/golang/ goenv install 1.24.3

# Set permanently for all installations
> export GO_BUILD_MIRROR_URL=https://mirrors.example.com/golang/
> goenv install 1.24.3
```

Common use cases:

- **Corporate environments**: Use internal mirrors behind firewalls
- **China regions**: Use faster regional mirrors
- **Air-gapped networks**: Point to internal package repository
- **Testing**: Use custom build servers

### Auto-Rehash

By default, goenv automatically rehashes shims after installation so that installed binaries are immediately available. For batch installations or CI/CD pipelines, you can disable this:

```shell
# Disable for single install
> goenv install 1.21.0 --no-rehash

# Batch install with single rehash at end
> goenv install 1.21.0 --no-rehash
> goenv install 1.22.0 --no-rehash
> goenv rehash

# Or use environment variable
> export GOENV_NO_AUTO_REHASH=1
> goenv install 1.21.0
> goenv install 1.22.0
> goenv rehash
```

## `goenv installed`

Display an installed Go version, searching for shortcuts if necessary. Useful for scripting and automation to verify version installations.

**Usage:**

```shell
# Show the latest installed version
> goenv installed latest
1.23.2

# Check if a specific version is installed
> goenv installed 1.22.5
1.22.5

# With no arguments, shows current version
> goenv installed
1.22.5

# Check for system Go
> goenv installed system
/usr/local/go/bin/go
```

**Scripting Examples:**

```bash
# Check if version exists before using it
if goenv installed 1.25.2 &>/dev/null; then
  echo "Go 1.25.2 is installed"
  goenv use 1.25.2
else
  echo "Installing Go 1.25.2..."
  goenv install 1.25.2
fi

# Use latest installed version
LATEST=$(goenv installed latest)
echo "Using latest installed: $LATEST"
goenv use "$LATEST" --global

# Verify installation in CI
#!/bin/bash
REQUIRED_VERSION="1.25.2"
if ! goenv installed "$REQUIRED_VERSION" &>/dev/null; then
  echo "Error: Go $REQUIRED_VERSION not installed"
  exit 1
fi

# Switch between versions in test matrix
for version in $(goenv list --bare); do
  echo "Testing with Go $version"
  goenv use "$version"
  go test ./...
done

# Find which version satisfies a constraint
# (Combine with parsing logic)
LATEST_125=$(goenv list --bare | grep "^1\.25\." | tail -1)
if goenv installed "$LATEST_125" &>/dev/null; then
  goenv use "$LATEST_125"
fi
```

**Exit Codes:**

- `0` - Version found and printed
- `1` - Version not found or error

## `goenv inventory`

Lists Go versions and tools installed by goenv for audit and compliance purposes.

**Note:** This is NOT an SBOM generator - it's a simple inventory tool. For project SBOMs, use [`goenv sbom project`](#goenv-sbom).

**Usage:**

```shell
# List installed Go versions
> goenv inventory go

# Output as JSON
> goenv inventory go --json

# Include SHA256 checksums
> goenv inventory go --checksums
```

**Example text output:**

```
═══════════════════════════════════════════════════════════════
               GOENV GO VERSION INVENTORY
═══════════════════════════════════════════════════════════════

1. Go 1.24.4
   Path:      /Users/user/.goenv/versions/1.24.4
   Binary:    /Users/user/.goenv/versions/1.24.4/bin/go
   Platform:  darwin/arm64
   Installed: 2025-10-23 20:49:09

───────────────────────────────────────────────────────────────
Total: 1 Go version(s) installed
═══════════════════════════════════════════════════════════════
```

**Example JSON output:**

```json
[
  {
    "version": "1.24.4",
    "path": "/Users/user/.goenv/versions/1.24.4",
    "binary_path": "/Users/user/.goenv/versions/1.24.4/bin/go",
    "installed_at": "2025-10-23T20:49:09Z",
    "sha256": "abc123def456...",
    "os": "darwin",
    "arch": "arm64"
  }
]
```

**Use cases:**

- Compliance audits
- System inventory reports
- Tracking installed versions across environments

## `goenv local`

Sets a local application-specific Go version by writing the version
name to a `.go-version` file in the current directory. This version
overrides the global version, and can be overridden itself by setting
the `GOENV_VERSION` environment variable or with the `goenv shell`
command.

```shell
> goenv local 1.6.1
```

When run without a version number, `goenv local` reports the currently
configured local version. You can also unset the local version:

```shell
> goenv local --unset
```

**NEW: tfswitch-like behavior** - `goenv` (with no arguments) now automatically detects `.go-version` files:

```shell
# When you run goenv with no arguments in a directory with .go-version
> goenv
Found .go-version: 1.23.2
✓ Go 1.23.2 is installed

# If the version isn't installed, offers to install it
> goenv
Found .go-version: 1.23.2
⚠️  Go 1.23.2 is not installed
Install now? (Y/n) y
Installing Go 1.23.2...
✓ Installed Go 1.23.2

# Use --help to bypass detection and show help
> goenv --help
```

**NEW: Sync flag** - Ensure the version from `.go-version` is installed:

```shell
> goenv local --sync
Found .go-version: 1.23.2
✓ Go 1.23.2 is installed

# Or if not installed, automatically installs
> goenv local --sync
Found .go-version: 1.23.2
⚠️  Go 1.23.2 is not installed
Installing Go 1.23.2...
✓ Installed Go 1.23.2
```

This makes it easy to ensure your project's Go version is ready to use, similar to how `tfswitch` works for Terraform versions.

**Automatically set up VS Code integration when setting a local version:**

```shell
> goenv local 1.22.0 --vscode

Initializing VS Code workspace...
✓ Created/updated .vscode/settings.json
✓ Created/updated .vscode/extensions.json
✨ VS Code workspace configured for goenv!
```

The `--vscode` flag creates `.vscode/settings.json` and `.vscode/extensions.json` files configured to use goenv. This makes it easy to set up new projects with proper IDE integration in one command.

Previous versions of goenv stored local version specifications in a
file named `.goenv-version`. For backwards compatibility, goenv will
read a local version specified in an `.goenv-version` file, but a
`.go-version` file in the same directory will take precedence.

**Note:** goenv automatically discovers versions from both `.go-version` and `go.mod` files with smart precedence rules. See the [Version Discovery](#version-discovery--precedence) section below for details.

### `goenv local` (advanced)

You can specify local Go version.

```shell
> goenv local 1.5.4

# Showcase
> goenv versions
  system
  * 1.5.4 (set by /Users/syndbg/path/to/project/.go-version)

> goenv version
1.5.4 (set by /Users/syndbg/path/to/project/.go-version)

> go version

go version go1.5.4 darwin/amd64
```

**Version shorthand syntax:**

You can use a shorthand syntax to set the local version by specifying the version directly as the first argument:

```shell
# These are equivalent:
> goenv 1.21.0
> goenv local 1.21.0

# Works with various version formats:
> goenv 1.22.5        # Full version
> goenv latest        # Latest installed version
> goenv system        # System Go
```

This shorthand automatically routes to the `local` command, making it faster to switch versions in your current directory.

## `goenv tools sync`

Sync/replicate installed Go tools from one Go version to another.

This command discovers tools in the source version and reinstalls them (from source) in the target version. The source version remains unchanged - think of this as "syncing" or "replicating" your tool setup rather than "moving" tools.

**Smart defaults:**

- No args: Sync from version with most tools → current version
- One arg: Sync from that version → current version
- Two args: Sync from source → target (explicit control)

**Usage:**

```shell
# Auto-sync: finds best source, syncs to current version
> goenv tools sync

# Sync from specific version to current version
> goenv tools sync 1.24.1

# Explicit source and target
> goenv tools sync 1.24.1 1.25.2

# Preview auto-sync
> goenv tools sync --dry-run

# Sync only specific tools
> goenv tools sync 1.24.1 --select gopls,delve

# Exclude certain tools
> goenv tools sync 1.24.1 --exclude staticcheck
```

### Options

- `--dry-run` / `-n` - Show what would be synced without actually syncing
- `--select <tools>` - Comma-separated list of tools to sync (e.g., gopls,delve)
- `--exclude <tools>` - Comma-separated list of tools to exclude from sync

This command is useful when upgrading Go versions and wanting to maintain your tool environment across versions.

## `goenv prefix`

Displays the directory where a Go version is installed. If no
version is given, `goenv prefix' displays the location of the
currently selected version.

```shell
> goenv prefix
/home/go-nv/.goenv/versions/1.11.1
```

## `goenv refresh`

Clears all cached version data and forces a fresh fetch from the official Go API on the next version-related command.

```shell
> goenv refresh
✓ Cache cleared! Removed 2 cache file(s).
Next version fetch will retrieve fresh data from go.dev

# With verbose output
> goenv refresh --verbose
✓ Removed versions-cache.json
✓ Removed releases-cache.json
✓ Cache cleared! Removed 2 cache file(s).
Next version fetch will retrieve fresh data from go.dev
```

This removes:

- `versions-cache.json` - Version list cache
- `releases-cache.json` - Full release metadata cache

Use this command when:

- A new Go version was just released and you want it immediately
- You suspect cached data is stale or corrupted
- Testing version fetching behavior
- Troubleshooting version-related issues

The next time you run `goenv install --list` or similar commands, goenv will fetch fresh data from go.dev.

## `goenv rehash`

Installs shims for all Go binaries known to goenv. This includes:

- Go version binaries: `~/.goenv/versions/*/bin/*`
- GOPATH binaries: `$GOENV_GOPATH_PREFIX/<version>/bin/*` (unless `GOENV_DISABLE_GOPATH=1`)

Run this command after you install a new version of Go, or install a package that provides binaries.

```shell
> goenv rehash
```

**GOPATH binary support:**

By default, goenv also creates shims for binaries in your GOPATH. This allows tools installed with `go install` to work seamlessly with version switching:

```shell
# Install a tool with the current Go version
> go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Rehash to create shims
> goenv rehash

# The tool is now available via shim
> golangci-lint version
```

To disable GOPATH binary scanning:

```shell
export GOENV_DISABLE_GOPATH=1
```

To customize the GOPATH location:

```shell
export GOENV_GOPATH_PREFIX=/custom/path  # Default: $HOME/go
```

## `goenv root`

Display the root directory where versions and shims are kept

```shell
> goenv root
/home/go-nv/.goenv
```

## `goenv sbom`

Generate Software Bill of Materials (SBOM) for Go projects using industry-standard tools.

**Note:** This is a thin wrapper that runs mature SBOM tools (cyclonedx-gomod, syft) with goenv-managed toolchains for reproducible CI builds.

### Prerequisites

Install an SBOM tool first:

```shell
# CycloneDX for Go modules
> goenv tools install cyclonedx-gomod@v1.6.0

# Syft for multi-language/container support
> goenv tools install syft@v1.0.0
```

### Usage: `goenv sbom project`

Generate an SBOM for a Go project:

```shell
# Generate CycloneDX SBOM (default)
> goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json

# Generate SPDX SBOM with syft
> goenv sbom project --tool=syft --format=spdx-json --output=sbom.spdx.json

# Scan a container image
> goenv sbom project --tool=syft --image=ghcr.io/myapp:v1.0.0

# Modules only (cyclonedx-gomod)
> goenv sbom project --tool=cyclonedx-gomod --modules-only

# Offline mode
> goenv sbom project --tool=cyclonedx-gomod --offline
```

**Options:**

| Option | Type | Description |
|--------|------|-------------|
| `--tool` | string | SBOM tool to use: `cyclonedx-gomod` (Go modules), `syft` (multi-language/containers) |
| `--format` | string | Output format: `cyclonedx-json`, `spdx-json`, `cyclonedx-xml`, `spdx-json`, `table`, `text` |
| `-o, --output` | string | Output file path (default: `sbom.json`, or stdout if `-`) |
| `--dir` | string | Project directory to scan (default: `.`) |
| `--image` | string | Container image to scan (syft only), e.g., `ghcr.io/myapp:v1.0.0` |
| `--modules-only` | bool | Only scan Go modules, exclude stdlib/main (cyclonedx-gomod only) |
| `--offline` | bool | Offline mode - avoid network access for module metadata |
| `--tool-args` | string | Additional arguments to pass directly to the underlying tool |

**Benefits:**

- Reproducible SBOMs with pinned Go and tool versions
- Correct cache isolation per Go version
- CI-friendly output (provenance to stderr, SBOM to stdout/file)
- Exit code preservation for pipeline integration

### Secured CI/CD Usage

For reproducible, auditable SBOMs in CI/CD pipelines, use fixed tool versions and offline mode:

**1. CycloneDX with Fixed Versions (Recommended for Go projects)**

```yaml
# GitHub Actions - Reproducible SBOM generation
- name: Setup Go version
  run: goenv use 1.25.2 --yes  # Pin Go version

- name: Install SBOM tool (pinned version)
  run: goenv tools install cyclonedx-gomod@v1.6.0  # Pin tool version

- name: Generate SBOM (offline, reproducible)
  run: |
    goenv sbom project \
      --tool=cyclonedx-gomod \
      --format=cyclonedx-json \
      --output=sbom.cdx.json \
      --offline  # No network calls during generation

- name: Verify SBOM
  run: |
    # Validate CycloneDX format
    jq '.bomFormat == "CycloneDX"' sbom.cdx.json
    jq '.components | length' sbom.cdx.json

- name: Upload SBOM artifact
  uses: actions/upload-artifact@v4
  with:
    name: sbom-cyclonedx
    path: sbom.cdx.json
```

**2. Syft with Fixed Versions (Multi-language projects)**

```yaml
# GitLab CI - SBOM with Syft (SPDX format)
sbom:
  stage: security
  script:
    - goenv use 1.25.2 --yes
    - goenv tools install syft@v1.0.0  # Pin syft version

    # Generate SPDX SBOM
    - goenv sbom project \
        --tool=syft \
        --format=spdx-json \
        --output=sbom.spdx.json \
        --offline

    # Also generate human-readable table
    - goenv sbom project \
        --tool=syft \
        --format=table \
        --output=sbom.txt
  artifacts:
    paths:
      - sbom.spdx.json
      - sbom.txt
    expire_in: 90 days
```

**3. Container Image Scanning**

```bash
# Scan a container image with Syft
goenv sbom project \
  --tool=syft \
  --format=spdx-json \
  --output=image-sbom.json \
  --image=ghcr.io/myorg/myapp:v1.2.3

# Scan with specific architecture
goenv sbom project \
  --tool=syft \
  --format=cyclonedx-json \
  --image=ghcr.io/myorg/myapp:v1.2.3 \
  --tool-args="--platform=linux/arm64"
```

**4. Modules-Only SBOM (Faster, Dependencies Only)**

```bash
# Generate SBOM for dependencies only (excludes stdlib, main package)
goenv sbom project \
  --tool=cyclonedx-gomod \
  --format=cyclonedx-json \
  --modules-only \
  --output=dependencies.cdx.json
```

**Security Best Practices:**

- ✅ **Pin tool versions** - Use `@v1.6.0` syntax, not `@latest`
- ✅ **Use `--offline` mode** - Prevents network calls during generation (faster, more secure)
- ✅ **Pin Go version** - Use `.go-version` file or `goenv use <version>`
- ✅ **Validate output** - Check SBOM format and component count after generation
- ✅ **Store SBOMs as artifacts** - Keep SBOMs with build artifacts for auditing
- ⚠️ **Avoid `@latest`** - Breaks reproducibility and audit trails

**Example CI workflow:**

```yaml
- name: Install SBOM tool
  run: goenv tools install cyclonedx-gomod@v1.6.0

- name: Generate SBOM
  run: goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json

- name: Upload SBOM
  uses: actions/upload-artifact@v4
  with:
    name: sbom
    path: sbom.json
```

## `goenv shell`

Sets a shell-specific Go version by setting the `GOENV_VERSION`
environment variable in your shell. This version overrides
application-specific versions and the global version.

```shell
> goenv shell 1.5.4
```

When run without a version number, `goenv shell` reports the current
value of `GOENV_VERSION`. You can also unset the shell version:

```shell
> goenv shell --unset
```

Note that you'll need goenv's shell integration enabled (refer to [Installation](./INSTALL.md]) in order to use this command. If you
prefer not to use shell integration, you may simply set the
`GOENV_VERSION` variable yourself:

```shell
> export GOENV_VERSION=1.5.4
```

## `goenv shims`

List existing goenv shims

```shell
> goenv shims
/home/go-nv/.goenv/shims/go
/home/go-nv/.goenv/shims/godoc
/home/go-nv/.goenv/shims/gofmt
```

## `goenv unalias`

Remove a version alias.

**Usage:**

```shell
# Remove an alias
> goenv unalias stable

# List remaining aliases
> goenv alias
dev -> 1.24rc1
lts -> 1.22.5
```

This command removes the specified alias from the `~/.goenv/aliases` file. The target version remains installed and can still be used directly by its version number.

**Example workflow:**

```shell
# Create an alias
> goenv alias temp 1.23.0

# Use it
> goenv local temp

# Remove it when no longer needed
> goenv unalias temp

# Version still works directly
> goenv local 1.23.0
```

## `goenv uninstall`

Uninstalls the specified version if it exists, otherwise - error.

```shell
> goenv uninstall 1.6.3
```

## `goenv update`

Update goenv to the latest version.

**Usage:**

```shell
# Update goenv
> goenv update
🔄 Checking for goenv updates...

✓ Updated to version 2.2.0

# Check for updates without installing
> goenv update --check
Current version: 2.1.0
Latest version: 2.2.0
Update available!

# Force update even if already up-to-date
> goenv update --force
```

### Options

- `--check` - Check for updates without installing
- `--force` - Force update even if already up-to-date

### Installation Methods

**Git-based installations (recommended):**

- Automatically detected when `.git` directory exists in `$GOENV_ROOT`
- Runs `git pull` in GOENV_ROOT directory
- Shows changes and new version
- Requires `git` command in PATH

**Binary installations:**

- Automatically detected when no `.git` directory exists
- Downloads latest release from GitHub using API
- Replaces current binary with platform-specific download
- Requires write permission to binary location
- Uses HTTP ETag caching for efficient update checks

### Update Detection & Caching

The update command automatically detects your installation type and uses the appropriate method.

**GitHub API with ETag Caching:**

For binary installations, goenv uses GitHub's releases API with conditional requests to minimize bandwidth and API rate limits:

- **ETag Cache Location:** `$GOENV_ROOT/cache/update-etag`
- **How it works:**
  1. First request: Downloads release info and saves ETag
  2. Subsequent requests: Sends `If-None-Match` header with cached ETag
  3. GitHub responds with `304 Not Modified` if no new release
  4. Only downloads release assets when a new version is available

**GitHub API Rate Limits:**

- Unauthenticated requests: 60 per hour per IP
- Authenticated requests: 5,000 per hour (recommended for CI/CD)

**Using GITHUB_TOKEN for higher rate limits:**

```shell
# Export token for authenticated API requests
export GITHUB_TOKEN=ghp_your_personal_access_token

# Now update commands use authenticated requests
goenv update --check
```

**Token requirements:**
- Fine-grained PAT: Read-only access to public repositories
- Classic PAT: No scopes needed (read public data)

**Best practices:**
- Store token in `.bashrc`, `.zshrc`, or CI environment variables
- Use fine-grained tokens with minimal permissions
- Never commit tokens to version control

### Error Recovery

**Git not found (git-based installations):**

If `git` is not in PATH, the command provides platform-specific installation guidance:

- **macOS:** Install Xcode Command Line Tools or Homebrew
- **Windows:** Install Git for Windows or use winget
- **Linux:** Install via package manager (apt, yum, pacman)
- **Alternative:** Download binary from GitHub releases

**Permission denied (binary installations):**

If binary update fails due to permissions:

- **macOS/Linux:** Run with `sudo goenv update` or install to user-writeable path
- **Windows:** Run PowerShell as Administrator or install to `%LOCALAPPDATA%\goenv`
- **Alternative:** Download and install manually from GitHub releases

## `goenv tools update`

Update installed Go tools to their latest versions.

**Usage:**

```shell
# Update all tools for current Go version
> goenv tools update

# Check for updates without installing
> goenv tools update --check

# Update only a specific tool
> goenv tools update --tool gopls

# Show what would be updated (dry run)
> goenv tools update --dry-run

# Update to specific version (default: latest)
> goenv tools update --version v1.2.3
```

### Options

- `--check` - Check for updates without installing
- `--tool <name>` - Update only the specified tool
- `--dry-run` - Show what would be updated without actually updating
- `--version <version>` - Target version (default: latest)

This command updates tools installed with `go install` in your current Go version's GOPATH.

## `goenv vscode`

Manage Visual Studio Code integration with goenv.

```shell
> goenv vscode
Commands to configure and manage Visual Studio Code integration with goenv

Available Commands:
  init        Initialize VS Code workspace for goenv
```

### `goenv vscode init`

Initialize VS Code workspace with goenv configuration.

This command creates or updates `.vscode/settings.json` and `.vscode/extensions.json` in the current directory to ensure VS Code uses the correct Go version from goenv.

**Usage:**

```shell
> goenv vscode init
✓ Created/updated /path/to/project/.vscode/settings.json
✓ Created/updated /path/to/project/.vscode/extensions.json

✨ VS Code workspace configured for goenv!
```

**Flags:**

- `--force` - Overwrite existing settings instead of merging
- `--template <name>` - Use specific template (basic, advanced, monorepo)
- `--workspace-paths` - Use `${workspaceFolder}`-relative paths for portability
- `--versioned-tools` - Use per-version tools directory

**Templates:**

| Template   | Description                                               |
| ---------- | --------------------------------------------------------- |
| `basic`    | Go configuration with goenv env vars (default)            |
| `advanced` | Includes gopls settings, format on save, organize imports |
| `monorepo` | Configured for large repositories with multiple modules   |

**Portability Knobs:**

For teams working across different machines or sharing VS Code settings in version control:

**`--workspace-paths`** - Makes paths relative to workspace folder
```shell
goenv vscode init --workspace-paths
```
- **Without flag**: Uses absolute paths like `/Users/you/.goenv/versions/1.25.2`
- **With flag**: Uses `${workspaceFolder}/.goenv/versions/1.25.2`
- **Use when**: Sharing .vscode settings in git, team has different home directories
- **Benefit**: Works across different machines without path modifications

**`--versioned-tools`** - Isolates tools per Go version
```shell
goenv vscode init --versioned-tools
```
- **Without flag**: Uses shared tools directory `~/.goenv/tools`
- **With flag**: Uses version-specific `~/.goenv/versions/1.25.2/tools`
- **Use when**: Testing gopls versions, strict version isolation needed
- **Benefit**: Each Go version has its own tool installations

**Combined for maximum portability:**
```shell
goenv vscode init --workspace-paths --versioned-tools
```

**Examples:**

```shell
# Basic setup (default)
> goenv vscode init

# Advanced setup with gopls configuration
> goenv vscode init --template advanced

# Force overwrite existing settings
> goenv vscode init --force

# Monorepo setup
> goenv vscode init --template monorepo
```

**What it does:**

1. Creates `.vscode/settings.json` with:

   - `go.goroot: ${env:GOROOT}` - Uses goenv's GOROOT
   - `go.gopath: ${env:GOPATH}` - Uses goenv's GOPATH
   - `go.toolsGopath` - Shared location for Go tools
   - Optional gopls and editor settings (advanced template)

2. Creates `.vscode/extensions.json` with:

   - Recommendation for the official Go extension (`golang.go`)

3. Merges with existing settings (unless `--force` is used)

**Integration with goenv use:**

You can also automatically set up VS Code when setting a Go version:

```shell
> goenv use 1.22.0 --vscode

Initializing VS Code workspace...
✓ Created/updated .vscode/settings.json
✓ Created/updated .vscode/extensions.json
✨ VS Code workspace configured for goenv!
```

**Doctor integration:**

The `goenv doctor` command checks your VS Code integration:

```shell
> goenv doctor
...
✅ VS Code integration
   VS Code configured to use goenv environment variables
```

See the [VS Code Integration Guide](../user-guide/VSCODE_INTEGRATION.md) for detailed setup instructions and troubleshooting.

## `goenv version`

Displays the currently active Go version, along with information on
how it was set.

```shell
> goenv version
1.11.1 (set by /home/syndbg/work/go-nv/goenv/.go-version)
```

## `goenv --version`

Show version of `goenv` in format of `goenv <version>`.

## `goenv version-file`

Detect the file that sets the current goenv version

```shell
> goenv version-file
/home/syndbg/work/go-nv/goenv/.go-version
```

## `goenv version-file-read`

Reads specified version file if it exists

```shell
> goenv version-file-read ./go-version
1.11.1
```

## `goenv version-file-write`

Writes specified version(s) to the specified file if the version(s) exist

```shell
> goenv version-file-write ./go-version 1.11.1
```

## `goenv version-name`

Shows the current Go version

```shell
> goenv version-name
1.11.1
```

## `goenv version-origin`

Explain how the current Go version is set.

```shell
> goenv version-origin
/home/go-nv/.goenv/version)
```

## `goenv versions`

> **Note:** This is a legacy command. Use [`goenv list`](#goenv-list) instead for a more consistent interface.

Lists all Go versions known to goenv, and shows an asterisk next to
the currently active version.

```shell
> goenv versions
  1.4.0
  1.4.1
  1.4.2
  1.4.3
  1.5.0
  1.5.1
  1.5.2
  1.5.3
  1.5.4
  1.6.0
* 1.6.1 (set by /home/go-nv/.goenv/version)
  1.6.2

# Display bare version numbers only (without current marker)
> goenv versions --bare
1.4.0
1.4.1
1.4.2
1.5.0
1.5.1
1.6.0
1.6.1
1.6.2

# Skip aliases in output
> goenv versions --skip-aliases
  1.4.0
  1.5.0
* 1.6.1 (set by /home/go-nv/.goenv/version)

# JSON output for automation
> goenv versions --json
{
  "schema_version": "1",
  "versions": [
    {
      "version": "1.4.0",
      "active": false
    },
    {
      "version": "1.6.1",
      "active": true,
      "source": "global"
    }
  ]
}
```

### Options

- `--bare` - Display bare version numbers only (no current marker, one per line)
- `--skip-aliases` - Skip aliases in the output
- `--json` - Output in JSON format (for automation/CI)

## `goenv whence`

Lists all Go versions with the given command installed. Searches both version bin directories and GOPATH bin directories (unless `GOENV_DISABLE_GOPATH=1`).

```shell
# List versions with the 'go' command
> goenv whence go
1.3.0
1.3.1
1.3.2
1.3.3
1.4.0
1.4.1
1.4.2
1.4.3
1.5.0
1.5.1
1.5.2
1.5.3
1.5.4
1.6.0
1.6.1
1.6.2

# List versions with a GOPATH-installed tool
> goenv whence golangci-lint
1.21.5
1.22.5
```

Use the `--path` flag to show full paths instead of version names:

```shell
> goenv whence --path golangci-lint
/home/go-nv/go/1.21.5/bin/golangci-lint
/home/go-nv/go/1.22.5/bin/golangci-lint
```

## `goenv which`

Displays the full path to the executable that goenv will invoke when
you run the given command. Searches both version bin directories and GOPATH bin directories (unless `GOENV_DISABLE_GOPATH=1`).

```shell
# Find a Go core binary
> goenv which gofmt
/home/go-nv/.goenv/versions/1.6.1/bin/gofmt

# Find a GOPATH-installed tool
> goenv which golangci-lint
/home/go-nv/go/1.22.5/bin/golangci-lint
```

---

## Version Discovery & Precedence

goenv automatically discovers the required Go version from multiple sources. When you run `goenv` or other commands, it searches for version information in this order:

### Version Sources

1. **GOENV_VERSION environment variable** (highest priority)

   - Set explicitly in your current shell
   - Example: `GOENV_VERSION=1.24.1 goenv version`

2. **.go-version file** (project-specific)

   - Simple text file with version number
   - Created with `goenv use <version>`
   - Searched from current directory up to root

3. **go.mod file** (Go module projects)

   - Uses `toolchain` directive (preferred)
   - Falls back to `go` directive if no toolchain
   - Standard Go toolchain mechanism

4. **~/.goenv/version** (global fallback)
   - Your default Go version
   - Set with `goenv use <version> --global`

### Smart Precedence Rules

When **both** `.go-version` and `go.mod` exist in the same directory:

| Scenario                                              | Result          | Reason                                      |
| ----------------------------------------------------- | --------------- | ------------------------------------------- |
| `.go-version = 1.25.0`<br>`go.mod toolchain = 1.24.1` | Uses **1.25.0** | User's explicit choice to use newer version |
| `.go-version = 1.24.1`<br>`go.mod toolchain = 1.24.1` | Uses **1.24.1** | Versions match (prefers user's file)        |
| `.go-version = 1.23.0`<br>`go.mod toolchain = 1.24.1` | Uses **1.24.1** | go.mod toolchain is project requirement     |

**Key principle:** go.mod's `toolchain` directive specifies the **minimum required version**. If your `.go-version` is older, goenv will use the go.mod version to ensure builds work correctly.

#### Interactive Mode

When goenv detects that `.go-version` is older than go.mod's toolchain, it will prompt:

```
⚠️  Your .go-version (1.23.0) is older than go.mod's toolchain requirement (1.24.1)
   Using 1.24.1 as required by go.mod

Update .go-version to 1.24.1 to avoid this warning? (Y/n)
```

This helps keep your version files in sync.

### Examples

**Example 1: Only .go-version exists**

```shell
> cat .go-version
1.24.1

> goenv version
1.24.1 (set by /path/to/project/.go-version)
```

**Example 2: Only go.mod exists**

```shell
> cat go.mod
module myproject

go 1.22
toolchain go1.24.1

> goenv version
1.24.1 (set by /path/to/project/go.mod)
```

**Example 3: Both exist, .go-version is newer**

```shell
> cat .go-version
1.25.0

> cat go.mod
module myproject
go 1.22
toolchain go1.24.1

> goenv version
1.25.0 (set by /path/to/project/.go-version)
# Uses .go-version because 1.25.0 >= 1.24.1 (requirement satisfied)
```

**Example 4: Both exist, .go-version is older**

```shell
> cat .go-version
1.23.0

> cat go.mod
module myproject
go 1.22
toolchain go1.24.1

> goenv
Found go.mod: 1.24.1
⚠️  Your .go-version (1.23.0) is older than go.mod's toolchain requirement (1.24.1)
   Using 1.24.1 as required by go.mod

Update .go-version to 1.24.1 to avoid this warning? (Y/n)
```

### Best Practices

#### For Go Module Projects

1. **Rely on go.mod** - It's the standard way to specify Go version requirements
2. **Optional .go-version** - Only create if you need to pin a specific version
3. **Keep in sync** - If you create .go-version, keep it >= go.mod toolchain

```shell
# Let go.mod manage version
go mod edit -toolchain=go1.24.1

# OR pin explicitly with .go-version
goenv use 1.24.1  # Creates .go-version
```

#### For Scripts/Non-Module Projects

Use `.go-version` as your primary version specification:

```shell
goenv use 1.24.1  # Creates .go-version
```

#### For CI/CD

Set `GOENV_AUTO_INSTALL=1` to automatically install the discovered version:

```yaml
# GitHub Actions example
env:
  GOENV_AUTO_INSTALL: 1

steps:
  - run: goenv # Auto-discovers and installs version from .go-version or go.mod
```

### Troubleshooting

**Q: Why is goenv using a different version than my .go-version?**

A: Check if you have a go.mod with a newer toolchain requirement. Run `goenv doctor` to see what version is discovered and why.

**Q: How do I force goenv to use my .go-version?**

A: Make sure your .go-version is >= the go.mod toolchain requirement. Update it with:

```shell
goenv use $(grep toolchain go.mod | awk '{print $2}' | sed 's/go//')
```

**Q: Can I disable go.mod version checking?**

A: No, this is intentional. Go's toolchain directive is a hard requirement for builds to work correctly. If you want to use a specific version, set it in .go-version to be >= the toolchain requirement.

**Q: I'm getting "exec format error" when running Go tools. How do I fix this?**

A: This error occurs when cached binaries were built with a different Go version or architecture. Starting with goenv v3, build caches are automatically isolated per version to prevent this.

If you're still seeing this error:

1. Clean your shared system cache (one-time migration):

   ```shell
   go clean -cache
   go clean -modcache
   ```

2. Verify cache isolation is working:

   ```shell
   goenv exec go env GOCACHE
   # Should show: ~/.goenv/versions/{version}/go-build (version-specific)
   ```

3. Run diagnostics:

   ```shell
   goenv cache status
   goenv doctor
   ```

4. If the issue persists, clean the build cache:
   ```shell
   goenv cache clean build
   ```

See the [Smart Caching Guide](../advanced/SMART_CACHING.md#build-cache-isolation) for more details about cache isolation.
