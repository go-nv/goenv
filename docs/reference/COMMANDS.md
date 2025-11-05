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
  - [Global Flags \& Output Options](#global-flags--output-options)
    - [Machine-Readable Output](#machine-readable-output)
  - [🚀 Modern Unified Commands (Recommended)](#-modern-unified-commands-recommended)
  - [📋 All Subcommands](#-all-subcommands)
  - [`goenv get-started`](#goenv-get-started)
  - [`goenv alias`](#goenv-alias)
  - [`goenv cache`](#goenv-cache)
    - [`goenv cache status`](#goenv-cache-status)
    - [`goenv cache clean`](#goenv-cache-clean)
    - [`goenv cache migrate`](#goenv-cache-migrate)
    - [`goenv cache info`](#goenv-cache-info)
  - [`goenv ci-setup`](#goenv-ci-setup)
  - [`goenv commands`](#goenv-commands)
  - [`goenv explore`](#goenv-explore)
  - [`goenv completions`](#goenv-completions)
  - [`goenv tools`](#goenv-tools)
  - [`goenv tools default`](#goenv-tools-default)
  - [`goenv tools install`](#goenv-tools-install)
  - [`goenv tools uninstall`](#goenv-tools-uninstall)
  - [`goenv tools list`](#goenv-tools-list)
  - [`goenv tools outdated`](#goenv-tools-outdated)
  - [`goenv tools status`](#goenv-tools-status)
  - [`goenv doctor`](#goenv-doctor)
    - [Quick Usage](#quick-usage)
    - [Exit Codes (for CI/automation)](#exit-codes-for-ciautomation)
    - [Flags](#flags)
    - [Interactive Fix Mode](#interactive-fix-mode)
    - [Example Output](#example-output)
    - [Comprehensive Checks](#comprehensive-checks)
    - [Comprehensive Checks](#comprehensive-checks-1)
    - [JSON Output (Recommended for CI/CD)](#json-output-recommended-for-cicd)
    - [CI/CD Integration Examples](#cicd-integration-examples)
    - [Advanced: Parse Specific Checks](#advanced-parse-specific-checks)
  - [`goenv status`](#goenv-status)
  - [`goenv use`](#goenv-use)
  - [`goenv current`](#goenv-current)
  - [`goenv list`](#goenv-list)
  - [`goenv info`](#goenv-info)
  - [`goenv compare`](#goenv-compare)
  - [`goenv exec`](#goenv-exec)
  - [`goenv global`](#goenv-global)
  - [`goenv help`](#goenv-help)
  - [`goenv init`](#goenv-init)
  - [`goenv setup`](#goenv-setup)
  - [`goenv install`](#goenv-install)
    - [Options](#options)
    - [Using a Custom Mirror](#using-a-custom-mirror)
    - [Security: Checksum Verification](#security-checksum-verification)
    - [Auto-Rehash](#auto-rehash)
  - [`goenv installed`](#goenv-installed)
  - [`goenv inventory`](#goenv-inventory)
  - [`goenv local`](#goenv-local)
    - [`goenv local` (advanced)](#goenv-local-advanced)
  - [`goenv tools sync-tools`](#goenv-tools-sync-tools)
    - [Options](#options-1)
  - [`goenv prefix`](#goenv-prefix)
  - [`goenv refresh`](#goenv-refresh)
  - [`goenv rehash`](#goenv-rehash)
  - [`goenv root`](#goenv-root)
  - [`goenv sbom`](#goenv-sbom)
    - [⚠️ Current State (v3.0)](#️-current-state-v30)
    - [Prerequisites](#prerequisites)
    - [Usage: `goenv sbom project`](#usage-goenv-sbom-project)
    - [🔒 Secured CI/CD Example (Recommended)](#-secured-cicd-example-recommended)
    - [Alternative: Direct Tool Execution](#alternative-direct-tool-execution)
    - [CycloneDX vs SPDX Format](#cyclonedx-vs-spdx-format)
    - [Advanced: Container Image Scanning (Syft)](#advanced-container-image-scanning-syft)
    - [Troubleshooting](#troubleshooting)
    - [🔐 Security Best Practices](#-security-best-practices)
  - [`goenv shell`](#goenv-shell)
  - [`goenv shims`](#goenv-shims)
  - [`goenv unalias`](#goenv-unalias)
  - [`goenv uninstall`](#goenv-uninstall)
  - [`goenv update`](#goenv-update)
    - [Options](#options-2)
    - [Installation Type Detection](#installation-type-detection)
    - [Installation Methods](#installation-methods)
      - [Git-based installations (recommended)](#git-based-installations-recommended)
      - [Binary installations](#binary-installations)
    - [GitHub API Implementation](#github-api-implementation)
      - [ETag Caching](#etag-caching)
      - [Rate Limits \& Authentication](#rate-limits--authentication)
      - [Retry Logic \& Backoff](#retry-logic--backoff)
      - [Security Features](#security-features)
    - [Error Recovery](#error-recovery)
      - [Git not found (git-based installations)](#git-not-found-git-based-installations)
      - [Permission denied (binary installations)](#permission-denied-binary-installations)
      - [Uncommitted changes (git-based installations)](#uncommitted-changes-git-based-installations)
    - [Examples](#examples)
    - [Troubleshooting](#troubleshooting-1)
  - [`goenv tools update`](#goenv-tools-update)
  - [`goenv vscode`](#goenv-vscode)
    - [`goenv vscode setup`](#goenv-vscode-setup)
    - [`goenv vscode init`](#goenv-vscode-init)
  - [`goenv version`](#goenv-version)
  - [`goenv --version`](#goenv---version)
  - [`goenv version-file`](#goenv-version-file)
  - [`goenv version-file-read`](#goenv-version-file-read)
  - [`goenv version-file-write`](#goenv-version-file-write)
  - [`goenv version-name`](#goenv-version-name)
  - [`goenv version-origin`](#goenv-version-origin)
  - [`goenv versions`](#goenv-versions)
    - [Options](#options-3)
  - [`goenv whence`](#goenv-whence)
  - [`goenv which`](#goenv-which)
  - [Version Discovery \& Precedence](#version-discovery--precedence)
    - [Version Sources](#version-sources)
    - [Smart Precedence Rules](#smart-precedence-rules)
      - [Interactive Mode](#interactive-mode)
    - [Examples](#examples-1)
    - [Best Practices](#best-practices)
      - [For Go Module Projects](#for-go-module-projects)
      - [For Scripts/Non-Module Projects](#for-scriptsnon-module-projects)
      - [For CI/CD](#for-cicd)
    - [Troubleshooting](#troubleshooting-2)
  - [🔄 Legacy Commands (Backward Compatibility)](#-legacy-commands-backward-compatibility)
    - [Command Migration Guide](#command-migration-guide)
    - [Why Use Modern Commands?](#why-use-modern-commands)
    - [Examples](#examples-2)
    - [Backward Compatibility Promise](#backward-compatibility-promise)
    - [Deprecation Timeline](#deprecation-timeline)
    - [Recommendation for New Projects](#recommendation-for-new-projects)
    - [Legacy Command Documentation](#legacy-command-documentation)

## `goenv get-started`

Interactive beginner's guide that shows step-by-step instructions for setting up and using goenv. Perfect for first-time users or when you need a quick refresher on the basics.

**Usage:**

```shell
# Show getting started guide
> goenv get-started
👋 Welcome to goenv!

Step 1: Initialize goenv in your shell
To get started, add goenv to your shell by running:

  echo 'eval "$(goenv init -)"' >> ~/.zshrc
  source ~/.zshrc

Or for just this session:
  eval "$(goenv init -)"

Step 2: Install a Go version
Install your first Go version:

  goenv install        → Install latest stable Go
  goenv install 1.21.5 → Install specific version
  goenv install -l     → List all available versions

Step 3: Set your default version
After installing:

  goenv global <version>

Helpful Commands:
  goenv doctor    Check your goenv installation
  goenv status    Show your current configuration
  goenv --help    List all commands
  goenv install -l List installable versions
```

**Adaptive Content:**

The guide adapts based on your setup status:

1. **Shell not initialized**: Shows shell-specific initialization steps
2. **No versions installed**: Shows how to install Go versions
3. **Already set up**: Confirms setup and shows helpful commands

**Shell-Specific Instructions:**

The command detects your shell and provides tailored setup instructions:

- **bash**: Uses `~/.bashrc`
- **zsh**: Uses `~/.zshrc`
- **fish**: Uses `~/.config/fish/config.fish` with fish syntax
- **Other**: Shows generic `eval` command

**What It Covers:**

1. **Shell Initialization**

   - How to add goenv to your shell
   - Shell-specific configuration files
   - Temporary vs permanent setup

2. **Installing Go Versions**

   - How to install latest version
   - How to install specific versions
   - How to list available versions

3. **Setting Default Version**

   - Using `goenv global` to set default
   - Understanding version selection

4. **Helpful Commands**
   - Quick reference to common commands
   - Links to detailed documentation

**Examples:**

```shell
# First time setup - shell not initialized
> goenv get-started
👋 Welcome to goenv!

Step 1: Initialize goenv in your shell
To get started, add goenv to your shell by running:

  echo 'eval "$(goenv init -)"' >> ~/.zshrc
  source ~/.zshrc

# After shell initialization
> goenv get-started
👋 Welcome to goenv!

✓ goenv is initialized in your shell

Step 2: Install a Go version
Install your first Go version:

  goenv install        → Install latest stable Go
  goenv install 1.21.5 → Install specific version

# Fully set up
> goenv get-started
👋 Welcome to goenv!

✓ goenv is initialized in your shell

✓ You have Go versions installed

Helpful Commands:
  goenv doctor    Check your goenv installation
  goenv status    Show your current configuration
  goenv --help    List all commands
```

**When to Use:**

- **First installation**: Guide for initial setup
- **Quick refresher**: Reminder of basic commands
- **Teaching others**: Show to new team members
- **Troubleshooting**: Verify setup steps
- **Documentation**: Quick reference

**Comparison with Related Commands:**

- **`goenv get-started`**: Step-by-step guide, educational, shows what to do next
- **`goenv setup`**: Automated configuration, modifies files, does the work for you
- **`goenv doctor`**: Diagnostic tool, checks for problems, provides detailed analysis
- **`goenv status`**: Quick snapshot, shows current state, minimal output

**Aliases:**

The command has several aliases for convenience:

- `goenv getting-started`
- `goenv quickstart`
- `goenv first-run`

**Related Commands:**

- [`goenv setup`](#goenv-setup) - Automatic configuration wizard
- [`goenv doctor`](#goenv-doctor) - Diagnose installation issues
- [`goenv status`](#goenv-status) - Quick health check
- [`goenv explore`](#goenv-explore) - Discover commands by category
- [`goenv init`](#goenv-init) - Manual shell initialization

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
- `--force, -f` - Skip confirmation prompt (required for non-interactive environments)

**Non-Interactive Environments:**

When running in CI/CD, scripts, or non-interactive shells, you have two options:

**Option 1: Use `GOENV_ASSUME_YES` environment variable (Recommended for CI/CD)**

```bash
# CI/CD pipeline (recommended)
export GOENV_ASSUME_YES=1
goenv cache clean all

# Or inline
GOENV_ASSUME_YES=1 goenv cache clean build --older-than 30d

# GitHub Actions / GitLab CI
env:
  GOENV_ASSUME_YES: 1
run: goenv cache clean all
```

**Option 2: Use `--force` flag**

```bash
# Quick one-off commands
goenv cache clean all --force

# Automated cleanup script
goenv cache clean build --older-than 30d --force

# Docker container cleanup
RUN goenv cache clean all --force
```

**Why `GOENV_ASSUME_YES` is better for CI/CD:**

- More explicit about intent (auto-confirming vs forcing)
- Works globally for all goenv prompts
- Self-documenting in CI/CD config files
- Follows industry standards (like `DEBIAN_FRONTEND=noninteractive`)

Without either option, you'll see helpful error messages with suggestions:

```
⚠️  Running in non-interactive mode (no TTY detected)

This command requires confirmation. Options:
  1. Add --force flag: goenv cache clean all --force
  2. Use dry-run first: goenv cache clean all --dry-run
  3. Set env var: GOENV_ASSUME_YES=1 goenv cache clean

For CI/CD, we recommend: GOENV_ASSUME_YES=1
```

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

**⚠️ IMPORTANT CLARIFICATION:**

This command is **only for migrating Go's internal build cache format**, not for migrating from bash goenv to Go goenv.

- ✅ **Migrates:** Go build caches (`go-build` → `go-build-{GOOS}-{GOARCH}`)
- ❌ **Does NOT migrate:** Installed Go versions, config files, environment variables, or GOPATH

**If you're upgrading from bash goenv:** Your installed versions and configuration already work - no migration needed!

**When to run migrate:**

- **After upgrading to goenv v3.x from v2.x (Go goenv only)** - New architecture-aware caching system
- **When seeing "exec format error"** - Often caused by architecture mismatch in old caches
- **Before cross-compiling** - Ensures each architecture has isolated cache
- **When `goenv doctor` reports old cache format** - Doctor will detect and recommend migration
- **When switching between architectures** - Prevents cache corruption (e.g., M1 Mac vs Intel Mac)

**When NOT to run migrate:**

- ❌ **Upgrading from bash goenv to Go goenv** - Not applicable (versions already work)
- ❌ **Fresh goenv installation** - No old caches to migrate
- ❌ **Haven't built anything yet** - Nothing to migrate

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

See the [CI/CD Integration Guide](../advanced/CI_CD_GUIDE.md) for detailed examples and best practices.

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

## `goenv explore`

Interactive command discovery tool that helps you find the right command by browsing commands organized by category and intent. Perfect when you know what you want to do but don't know which command to use.

**Usage:**

```shell
# Interactive mode - browse all categories
> goenv explore
🧭 Explore goenv Commands

Command Categories:

  🚀 getting-started      → Setup and first-time use (3 commands)
  📦 versions             → Install, switch, and manage Go versions (10 commands)
  🔧 tools                → Manage Go tools and binaries (3 commands)
  🐚 shell                → Configure your shell environment (2 commands)
  🔍 diagnostics          → Troubleshoot and verify installation (2 commands)
  🔌 integrations         → IDE and CI/CD setup (2 commands)
  ⚡ advanced             → Power user commands (3 commands)

Usage:
  goenv explore                  → Browse all commands
  goenv explore <category>       → Show commands in category
  goenv <command> --help         → Get detailed help

Quick Examples:
  goenv explore versions         → List version management commands
  goenv explore diagnostics      → Show diagnostic commands

# Show commands in a specific category
> goenv explore versions
📖 Version Management

  ● install
    Download and install a Go version
    Example: goenv install 1.21.5

  ● uninstall
    Remove an installed Go version
    Example: goenv uninstall 1.20.0

  ● list
    Show all installed Go versions
    Example: goenv list

  ● info
    Show detailed information about a version
    Example: goenv info 1.21.5

  ● compare
    Compare two Go versions side-by-side
    Example: goenv compare 1.20.5 1.21.5

Tip: Use 'goenv <command> --help' for detailed information

# Browse getting started commands
> goenv explore getting-started
📖 Getting Started

  ● get-started
    Interactive setup guide for new users
    Example: goenv get-started

  ● setup
    Automatic shell and IDE configuration
    Example: goenv setup

  ● doctor
    Check installation and diagnose issues
    Example: goenv doctor
```

**Available Categories:**

1. **getting-started** - Setup and first-time use

   - `get-started` - Interactive beginner's guide
   - `setup` - Automatic configuration wizard
   - `doctor` - Installation diagnostics

2. **versions** - Install, switch, and manage Go versions

   - `install` - Download and install versions
   - `uninstall` - Remove installed versions
   - `list` - Show installed versions
   - `global` - Set default version
   - `local` - Set project-specific version
   - `shell` - Set session-specific version
   - `version` - Show current version
   - `info` - Detailed version information
   - `compare` - Compare two versions

3. **tools** - Manage Go tools and binaries

   - `tools install` - Install tools
   - `tools list` - List installed tools
   - `tools update` - Update tools

4. **shell** - Configure your shell environment

   - `init` - Initialize shell integration
   - `rehash` - Rebuild shims

5. **diagnostics** - Troubleshoot and verify installation

   - `doctor` - Comprehensive diagnostics
   - `status` - Quick health check

6. **integrations** - IDE and CI/CD setup

   - `vscode init` - Configure VS Code
   - `ci-setup` - Set up for CI/CD

7. **advanced** - Power user commands
   - `which` - Show path to executable
   - `whence` - List versions with executable
   - `exec` - Execute with specific version

**When to Use:**

- **Don't know the command name**: Browse by what you want to do
- **Learning goenv**: Discover available features
- **Quick reference**: See command examples
- **Category overview**: See all commands in a domain

**Examples:**

```shell
# Find commands for managing versions
> goenv explore versions

# Find diagnostic commands
> goenv explore diagnostics

# Browse all categories
> goenv explore

# See getting started commands
> goenv explore getting-started

# Find tool management commands
> goenv explore tools
```

**Related Commands:**

- [`goenv commands`](#goenv-commands) - List all command names
- [`goenv help`](#goenv-help) - Get help for specific command
- [`goenv get-started`](#goenv-get-started) - Step-by-step guide for beginners
- [`goenv --help`](#goenv---version) - General help

## `goenv completions`

Provides auto-completion for itself and other commands by calling them with `--complete`.

## `goenv tools`

Manage Go tools on a per-version basis. Ensures tools are properly isolated per Go version and prevents accidental global installations.

**Subcommands:**

- `goenv tools install` - Install a tool for the current or all Go versions
- `goenv tools uninstall` - Uninstall a tool from the current or all Go versions
- `goenv tools list` - List installed tools for the current or all versions
- `goenv tools update` - Update installed tools to latest versions
- `goenv tools outdated` - Show which tools need updating across all versions
- `goenv tools status` - View tool consistency across all Go versions
- `goenv tools sync-tools` - Copy tools from one version to another
- `goenv tools default` - Manage automatic tool installation

**Quick Start:**

```shell
# Install a tool for current Go version
goenv tools install golang.org/x/tools/cmd/goimports@latest

# Install across ALL Go versions at once
goenv tools install gopls@latest --all

# Uninstall from current version
goenv tools uninstall gopls

# Uninstall from all versions
goenv tools uninstall gopls --all

# List tools for current version
goenv tools list

# List tools across all versions
goenv tools list --all

# Check which tools need updating
goenv tools outdated

# View tool consistency across versions
goenv tools status

# Update all tools in current version
goenv tools update

# Update tools across all versions
goenv tools update --all

# Sync tools from one version to another
goenv tools sync-tools 1.23.2 1.24.4
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

## `goenv tools install`

Install Go tools for the current or all Go versions with proper version isolation.

**Usage:**

```shell
# Install for current Go version
goenv tools install golang.org/x/tools/gopls@latest

# Install across ALL Go versions at once
goenv tools install gopls@latest --all

# Install multiple tools at once
goenv tools install gopls@latest staticcheck@latest golangci-lint@latest

# Preview what would be installed without installing
goenv tools install gopls@latest --all --dry-run

# Verbose output
goenv tools install gopls@latest --verbose
```

**Options:**

- `--all` - Install across all installed Go versions
- `--dry-run` - Show what would be installed without installing
- `--verbose`, `-v` - Show detailed output

**Common Tools:**

```shell
# Go language server
goenv tools install golang.org/x/tools/gopls@latest --all

# Import formatting
goenv tools install golang.org/x/tools/cmd/goimports@latest --all

# Linting
goenv tools install github.com/golangci/golangci-lint/cmd/golangci-lint@latest --all

# Static analysis
goenv tools install honnef.co/go/tools/cmd/staticcheck@latest --all

# Debugger
goenv tools install github.com/go-delve/delve/cmd/dlv@latest --all

# Stricter gofmt
goenv tools install mvdan.cc/gofumpt@latest --all
```

**Example Output:**

```shell
$ goenv tools install gopls@latest --all

📦 Installation Plan

Tools to install: gopls
Target versions:  1.21.0, 1.22.0, 1.23.0
Total operations: 3

🔧 Installing Tools

Go 1.21.0:
  ✓ Installed gopls

Go 1.22.0:
  ✓ Installed gopls

Go 1.23.0:
  ✓ Installed gopls

✅ Successfully installed 1 tool(s) across 3 version(s)
```

## `goenv tools uninstall`

Uninstall Go tools from the current or all Go versions with proper cleanup.

**Usage:**

```shell
# Uninstall from current Go version
goenv tools uninstall gopls

# Uninstall from ALL Go versions at once
goenv tools uninstall gopls --all

# Uninstall from global GOPATH
goenv tools uninstall gopls --global

# Uninstall multiple tools at once
goenv tools uninstall gopls staticcheck golangci-lint

# Preview what would be removed without removing
goenv tools uninstall gopls --all --dry-run

# Force removal without confirmation
goenv tools uninstall gopls --force

# Verbose output showing all files
goenv tools uninstall gopls --verbose
```

**Options:**

- `--all` - Uninstall from all installed Go versions
- `--global` - Uninstall from global GOPATH/bin
- `--force` - Skip confirmation prompts
- `--dry-run` - Show what would be removed without actually removing
- `--verbose`, `-v` - Show detailed output

**Example Output:**

```shell
$ goenv tools uninstall gopls --all

🗑️  Uninstall Plan

Go 1.21.0: /Users/you/.goenv/versions/1.21.0/gopath/bin
  ✗ gopls
    (1 file(s))

Go 1.22.0: /Users/you/.goenv/versions/1.22.0/gopath/bin
  ✗ gopls
    (1 file(s))

Go 1.23.0: /Users/you/.goenv/versions/1.23.0/gopath/bin
  ✗ gopls
    (1 file(s))

Total: 3 tool(s), 3 file(s)

Remove 3 tool(s)? (y/N): y

🗑️  Uninstalling Tools

✓ Uninstalled gopls (Go 1.21.0)
✓ Uninstalled gopls (Go 1.22.0)
✓ Uninstalled gopls (Go 1.23.0)

✅ Successfully uninstalled 3 tool(s)
```

**With --verbose flag:**

```shell
$ goenv tools uninstall gopls --all --verbose

🗑️  Uninstall Plan

Go 1.21.0: /Users/you/.goenv/versions/1.21.0/gopath/bin
  ✗ gopls
    → gopls
    → gopls.exe

Total: 1 tool(s), 2 file(s)
```

**Use Cases:**

- **Clean up old tools**: Remove tools you no longer need
- **Free up space**: Uninstall large tools like golangci-lint across all versions
- **Version migration**: Remove tools before syncing from a different version
- **Global cleanup**: Use `--global` to clean system GOPATH

## `goenv tools list`

List all Go tools installed for the currently active Go version or across all versions. Tools are isolated per version in `$HOME/.goenv/versions/{version}/gopath/bin/`.

**Usage:**

```shell
# List tools for current version
goenv tools list

# List tools across ALL versions
goenv tools list --all

# JSON output for automation
goenv tools list --json
goenv tools list --all --json
```

**Options:**

- `--all` - List tools across all installed Go versions
- `--json` - Output in JSON format for CI/automation (machine-readable)

**Example Output:**

```shell
$ goenv tools list

🔧 Tools for Go 1.23.0

  • gopls
  • staticcheck
  • gofmt

Total: 3 tool(s)
```

```shell
$ goenv tools list --all

🔧 Tools Across All Versions

Go 1.21.0: (2 tool(s))
  • gopls
  • staticcheck

Go 1.22.0: (1 tool(s))
  • gopls

Go 1.23.0: (3 tool(s))
  • gopls
  • staticcheck
  • gofmt

Total: 3 tool(s) across 3 version(s)
```

**Output Format:**

Human-readable output shows one tool per line with the full import path.

JSON output provides structured data with a stable schema:

```json
{
  "schema_version": "1",
  "go_version": "1.25.2",
  "tools": [
    {
      "path": "golang.org/x/tools/cmd/goimports",
      "binary": "goimports"
    },
    {
      "path": "golang.org/x/tools/gopls",
      "binary": "gopls"
    },
    {
      "path": "github.com/golangci/golangci-lint/cmd/golangci-lint",
      "binary": "golangci-lint"
    }
  ]
}
```

**CI/CD Examples:**

```bash
# Check if a specific tool is installed
goenv tools list --json | jq -e '.tools[] | select(.binary=="gopls")'

# Count installed tools
TOOL_COUNT=$(goenv tools list --json | jq '.tools | length')
echo "Tools installed for Go $(goenv current --bare): $TOOL_COUNT"

# Verify required tools exist
REQUIRED_TOOLS=("gopls" "golangci-lint" "staticcheck")
INSTALLED=$(goenv tools list --json | jq -r '.tools[].binary')

for tool in "${REQUIRED_TOOLS[@]}"; do
  if ! echo "$INSTALLED" | grep -q "^${tool}$"; then
    echo "ERROR: Required tool '$tool' not installed"
    exit 1
  fi
done

# GitHub Actions - Verify tool installation
- name: Verify Go tools
  run: |
    goenv use 1.25.2
    goenv tools list --json | jq -e '.tools[] | select(.binary=="gopls")'
    goenv tools list --json | jq -e '.tools[] | select(.binary=="golangci-lint")'
```

## `goenv tools outdated`

Show which tools are outdated across all installed Go versions. Checks all tools in all Go versions and reports which ones have newer versions available.

**Usage:**

```shell
# Show all outdated tools
goenv tools outdated

# JSON output for automation
goenv tools outdated --json
```

**Options:**

- `--json` - Output in JSON format for CI/automation

**Example Output:**

```shell
$ goenv tools outdated

📊 Outdated Tools

Go 1.21.0: (2 outdated)
  ⬆️  gopls v0.12.0 → v0.13.2 available
  ⬆️  staticcheck v0.4.0 → v0.4.6 available

Go 1.23.0: (1 outdated)
  ⬆️  gopls v0.12.0 → v0.13.2 available

Total: 3 outdated tool(s) across 2 version(s)

💡 To update:
  goenv tools update                  # Update current version
  goenv tools update --all            # Update all versions
```

**When all tools are up to date:**

```shell
$ goenv tools outdated

✅ All tools are up to date!
```

**Use Cases:**

- Regular maintenance: Check which tools need updating before updates
- CI/CD: Detect drift in tool versions across Go versions
- Auditing: Ensure all versions have current security patches

## `goenv tools status`

Show tool installation consistency across all Go versions. Displays which tools are installed in which versions, helping maintain consistency.

**Usage:**

```shell
# Show tool installation status
goenv tools status

# JSON output for automation
goenv tools status --json
```

**Options:**

- `--json` - Output in JSON format for CI/automation

**Example Output:**

```shell
$ goenv tools status

📊 Tool Installation Status

Go versions: 3 installed

✅ Consistent Tools (3/3 versions)
  • gopls

⚠️  Partially Installed Tools
  • staticcheck → 2/3 versions (67%)
    Missing in: [1.22.0]

ℹ️  Version-Specific Tools
  • golangci-lint → only in 1.23.0
  • gofmt → only in 1.21.0

📝 Summary:
  Total tools: 4
  Consistent:  1
  Partial:     1
  Specific:    2

💡 Recommendations:
  • Use 'goenv tools install <tool> --all' to install across all versions
  • Use 'goenv tools sync-tools <from> <to>' to copy tools between versions
```

**Tool Categories:**

- **Consistent Tools**: Installed in all Go versions (100% consistency)
- **Partially Installed**: Installed in some but not all versions
- **Version-Specific**: Installed in only one version

**Use Cases:**

- **Onboarding**: Quickly see what tools are available and where
- **Maintenance**: Identify inconsistencies across Go versions
- **Migration**: Plan tool installation when adding new Go versions
- **CI/CD**: Verify development environment consistency

**Per-Version Isolation:**

Tools are isolated per Go version to prevent conflicts:

```shell
# Install gopls for Go 1.25.2
> goenv use 1.25.2
> go install golang.org/x/tools/gopls@latest
> goenv tools list
golang.org/x/tools/gopls

# Switch to Go 1.24.0
> goenv use 1.24.0
> goenv tools list
# (empty - gopls not installed for this version)

# Install gopls for Go 1.24.0
> go install golang.org/x/tools/gopls@latest
> goenv tools list
golang.org/x/tools/gopls
```

Each version maintains its own `$HOME/go/{version}/bin/` directory, ensuring tools are version-specific and don't interfere with each other.

See [GOPATH Integration](../advanced/GOPATH_INTEGRATION.md) for complete details on tool isolation.

## `goenv doctor`

**First-class CI/CD feature** for validating goenv installation and configuration, with **interactive fix mode** for automated repairs.

### Quick Usage

```shell
# Human-readable output (default)
goenv doctor

# Interactive fix mode (automatically repair issues)
goenv doctor --fix

# CI/CD with JSON output (recommended)
goenv doctor --json --fail-on=error

# Strict validation (fail on warnings)
goenv doctor --json --fail-on=warning
```

### Exit Codes (for CI/automation)

The `doctor` command uses distinct exit codes for precise pipeline control:

| Exit Code | Meaning  | Description                                                     |
| --------- | -------- | --------------------------------------------------------------- |
| `0`       | Success  | No issues found, or only warnings when `--fail-on=error`        |
| `1`       | Errors   | Critical issues found that prevent goenv from working correctly |
| `2`       | Warnings | Non-critical issues found when `--fail-on=warning` is set       |

### Flags

- `--fix` - **Interactive repair mode** - automatically fix detected issues (new in v3.0)
- `--json` - Output results in JSON format for CI/automation (machine-readable)
- `--fail-on <level>` - Exit with non-zero status on `error` (default) or `warning`

### Interactive Fix Mode

The `--fix` flag enables automated repair of common issues:

```shell
goenv doctor --fix
```

**Supported fixes:**

- Add missing shell initialization
- Fix PATH configuration
- Remove duplicate goenv installations
- Fix profile sourcing issues
- Clean stale cache
- Repair shims directory

Each fix is presented interactively with a prompt to confirm before applying. Safe to run multiple times.

### Example Output

Human-readable format:

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

### Comprehensive Checks

### Comprehensive Checks

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

### JSON Output (Recommended for CI/CD)

The `--json` flag outputs machine-readable JSON with stable check IDs:

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
    "ok": 21,
    "warnings": 2,
    "errors": 1
  }
}
```

**Check IDs** are stable and can be used in CI scripts to filter or assert specific checks. The JSON schema version ensures backward compatibility.

### CI/CD Integration Examples

**GitHub Actions - Basic:**

```yaml
- name: Validate goenv setup
  run: goenv doctor --json --fail-on=error
```

**GitHub Actions - With annotations:**

```yaml
- name: Health check
  run: |
    goenv doctor --json > report.json || true
    jq -r '.checks[] | select(.status=="error") | "::error ::\(.name): \(.message)"' report.json
    jq -r '.checks[] | select(.status=="warning") | "::warning ::\(.name): \(.message)"' report.json
    if jq -e '.summary.errors > 0' report.json; then exit 1; fi
```

**GitLab CI - Allow warnings:**

```yaml
check-goenv:
  script:
    - goenv doctor --json --fail-on=warning
  allow_failure:
    exit_codes: [2] # Allow warnings (exit code 2) to pass
```

**CircleCI:**

```yaml
- run:
    name: Validate goenv
    command: goenv doctor --json --fail-on=error
```

**Shell script - Handle all exit codes:**

```bash
goenv doctor --json --fail-on=warning || EXIT_CODE=$?

if [ "${EXIT_CODE:-0}" -eq 1 ]; then
  echo "ERROR: Critical errors found"
  exit 1
elif [ "${EXIT_CODE:-0}" -eq 2 ]; then
  echo "WARNING: Non-critical warnings found"
  # Continue anyway
  exit 0
fi

echo "SUCCESS: No issues found"
```

### Advanced: Parse Specific Checks

Use `jq` to filter specific check IDs for custom validation:

```bash
# Check if PATH is configured correctly
goenv doctor --json | jq -e '.checks[] | select(.id=="path-configuration" and .status=="ok")'

# Get all errors
goenv doctor --json | jq '.checks[] | select(.status=="error")'

# Count warnings
goenv doctor --json | jq '.summary.warnings'

# Check specific subsystem
goenv doctor --json | jq '.checks[] | select(.id | startswith("cache-"))'
```

## `goenv status`

Quick installation health check showing goenv initialization status, current version, and installed versions count. Similar to `git status` - provides a snapshot of your goenv environment at a glance.

**Usage:**

```shell
# Show current goenv status
> goenv status
📊 goenv Status

✓ goenv is initialized
  Shell: zsh
  Root: /Users/user/.goenv

Current version: 1.23.2 ✓
  Set by: ~/.goenv/version

Installed versions: 5
→ 1.23.2
  1.22.3
  1.21.8
  1.20.12
  1.19.13

Shims: 12 available

Auto-rehash: ✓ enabled
Auto-install: disabled

Run 'goenv doctor' for detailed diagnostics
```

**What it Shows:**

1. **Initialization Status**

   - Whether goenv is properly initialized in your shell
   - Current shell type (bash, zsh, fish, etc.)
   - goenv root directory

2. **Current Version**

   - Active Go version
   - Whether it's installed (✓) or missing (✗)
   - Source file that set the version

3. **Installed Versions**

   - Count of installed Go versions
   - List of up to 5 most recent versions
   - Arrow (→) indicates current version

4. **Shims Status**

   - Number of available shims
   - Reminder to run `goenv rehash` if needed

5. **Configuration**
   - Auto-rehash setting
   - Auto-install setting

**Examples:**

```shell
# Quick health check
> goenv status
📊 goenv Status

✓ goenv is initialized
  Shell: zsh

Current version: 1.23.2 ✓
  Set by: ~/.goenv/version

Installed versions: 3
→ 1.23.2
  1.22.3
  1.21.8

# Check initialization in new shell
> goenv status
📊 goenv Status

✗ goenv is not initialized in this shell
  Run: eval "$(goenv init -)"

# When no versions are installed
> goenv status
📊 goenv Status

✓ goenv is initialized
  Shell: bash

Current version: none (not set)
  Set with: goenv global <version>

Installed versions: 0
  Install with: goenv install

# Use in scripts to check if initialized
if goenv status | grep -q "initialized"; then
  echo "goenv is ready"
fi
```

**When to Use:**

- **First time setup**: Verify goenv is properly configured
- **Troubleshooting**: Quick check before diving into `goenv doctor`
- **New shell sessions**: Confirm initialization is working
- **CI/CD**: Verify goenv setup in pipelines
- **Daily use**: Quick overview of your environment

**Comparison with Related Commands:**

- **`goenv status`**: Quick overview, minimal output, fast
- **`goenv doctor`**: Comprehensive diagnostics, detailed checks, slower
- **`goenv current`**: Shows only the active version
- **`goenv list`**: Shows only installed versions

**Related Commands:**

- [`goenv doctor`](#goenv-doctor) - Comprehensive diagnostics and health checks
- [`goenv current`](#goenv-current) - Show active Go version
- [`goenv list`](#goenv-list) - List installed versions
- [`goenv init`](#goenv-init) - Initialize goenv in shell

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

**Output Format:**

When using `--remote`, the output follows the bash `goenv install --list` format for compatibility:

- **Header**: "Available versions:"
- **Indentation**: Two-space indentation for each version
- **Prefix stripping**: The "go" prefix is automatically stripped from version strings
- **Order**: Versions are displayed in **oldest-first order** (matching traditional bash behavior)
- **Pre-releases**: Beta, rc, and other pre-release versions are included by default unless `--stable` is specified

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

## `goenv info`

Show detailed information about a specific Go version, including installation status, release dates, support status, and upgrade recommendations.

**Usage:**

```shell
# Show info for a specific version
> goenv info 1.21.5
ℹ️   Go 1.21.5

  ✅ Status:       Installed
  📁 Install path: /Users/user/.goenv/versions/1.21.5
  💾 Size on disk: 487.3 MB

  📅 Released:     2023-12-05
  🟡 Support:      Near EOL (ends 2024-08-06)
                   Security updates only

  📖 Release notes: https://go.dev/doc/go1.21
  📦 Downloads:     https://go.dev/dl/

# Show info for current version
> goenv info $(goenv current --bare)

# Check if a version is EOL before installing
> goenv info 1.20.0
```

**Options:**

- `--json` - Output in JSON format for automation

**JSON Output:**

```json
{
  "version": "1.21.5",
  "installed": true,
  "install_path": "/Users/user/.goenv/versions/1.21.5",
  "size_bytes": 510918656,
  "size_human": "487.3 MB",
  "release_date": "2023-12-05",
  "eol_date": "2024-08-06",
  "status": "near_eol",
  "recommended": "1.22.0",
  "release_url": "https://go.dev/doc/go1.21",
  "download_url": "https://go.dev/dl/"
}
```

**Support Status Indicators:**

- **🟢 Current**: Fully supported with feature updates and bug fixes
- **🟡 Near EOL**: Approaching end of life, security updates only
- **🔴 EOL**: End of life, no longer receiving updates
- **❓ Unknown**: Version information not available (possibly newer)

**Examples:**

```shell
# Quick version check
> goenv info 1.23.2
ℹ️   Go 1.23.2

  ✅ Status:       Installed
  📁 Install path: /Users/user/.goenv/versions/1.23.2
  💾 Size on disk: 512.1 MB

  📅 Released:     2024-03-05
  🟢 Support:      Current (fully supported)

# Check before installing
> goenv info 1.25.0
ℹ️   Go 1.25.0

  ❌ Status:       Not installed
  💡 Install with: goenv install 1.25.0

  📅 Released:     2025-02-06
  🟢 Support:      Current (fully supported)

# Use in scripts to check EOL status
if goenv info 1.20.0 --json | jq -e '.status == "eol"' > /dev/null; then
  echo "Version is EOL, upgrade recommended"
fi
```

**Related Commands:**

- [`goenv compare`](#goenv-compare) - Compare two versions side-by-side
- [`goenv list`](#goenv-list) - List installed or available versions
- [`goenv install`](#goenv-install) - Install a Go version

## `goenv compare`

Compare two Go versions side-by-side to help decide which version to use or whether to upgrade.

**Usage:**

```shell
# Compare two versions
> goenv compare 1.21.5 1.22.3
⚖️   Comparing Go Versions

  Version:      1.21.5  vs  1.22.3
  Installed:    ✓ Installed  vs  ✓ Installed
  Released:     2023-12-05  vs  2024-05-15
  Age:          1y 2mo  vs  8mo
  Support:      🟡 Near EOL  vs  🟢 Current
  EOL Date:     2024-08-06  vs  2025-05-15
  Size:         487.3 MB  vs  512.1 MB

  📊 Size difference: +24.8 MB

🔍 Version Analysis
  📈 1 minor version newer (1.21 → 1.22)
  📅 Released 5 months apart

💡 Recommendations
  • 1.21.5 approaching EOL - plan upgrade soon
  • ✅ Upgrade to 1.22.3 recommended (current, supported)

  📖 Release notes:
     1.21.5: https://go.dev/doc/go1.21
     1.22.3: https://go.dev/doc/go1.22

# Compare current version with latest
> goenv compare $(goenv current --bare) 1.23.0

# Compare installed versions
> goenv compare 1.20.5 1.21.13
```

**Comparison Details:**

The command analyzes and displays:

- **Installation status**: Whether each version is installed
- **Release dates**: When each version was released
- **Age**: How old each version is (years/months/days)
- **Support status**: Current, Near EOL, or EOL
- **EOL dates**: When support ends for each version
- **Size**: Disk space used (if installed)
- **Version difference**: Major, minor, or patch level changes
- **Time gap**: Months between releases
- **Recommendations**: Upgrade suggestions based on support status

**Version Analysis Types:**

- **Major version change**: Significant changes expected (e.g., 1.x → 2.x)
- **Minor version**: New features and improvements (e.g., 1.20 → 1.21)
- **Patch upgrade**: Bug fixes and security updates (e.g., 1.21.1 → 1.21.5)

**Examples:**

```shell
# Check if upgrade is worth it
> goenv compare 1.21.5 1.22.0
⚖️   Comparing Go Versions

  Version:      1.21.5  vs  1.22.0
  Support:      🟡 Near EOL  vs  🟢 Current

💡 Recommendations
  • 1.21.5 approaching EOL - plan upgrade soon
  • ✅ Upgrade to 1.22.0 recommended (current, supported)

# Compare patch versions
> goenv compare 1.22.0 1.22.3
⚖️   Comparing Go Versions

🔍 Version Analysis
  🔧 Patch upgrade (+3) - bug fixes and security updates

# Compare before installing
> goenv compare $(goenv current --bare) $(goenv list --remote --stable | tail -1)
```

**Related Commands:**

- [`goenv info`](#goenv-info) - Show detailed information about a single version
- [`goenv list`](#goenv-list) - List available or installed versions
- [`goenv use`](#goenv-use) - Switch to a different version

## `goenv exec`

Run an executable with the selected Go version.

Assuming there's an already installed golang by e.g `goenv install 1.11.1` and
selected by e.g `goenv use 1.11.1 --global`,

```shell
> goenv exec go run main.go
```

## `goenv global`

> **Note:** This is a legacy command. Use [`goenv use <version> --global`](#goenv-use) instead for a more consistent interface.
>
> **For backward compatibility:** This command still works as expected, but `goenv use --global` is recommended as it provides a unified interface for both local and global version management.

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

## `goenv setup`

Automatic first-time configuration wizard that detects your shell, adds goenv initialization to your profile, and optionally configures VS Code. Safe to run multiple times - won't duplicate configuration.

**Usage:**

```shell
# Interactive setup with prompts
> goenv setup
🚀 Welcome to goenv setup!

🐚 Configuring shell integration...
  Detected shell: zsh
  Profile file: ~/.zshrc

  Add this to ~/.zshrc? [Y/n]:
    eval "$(goenv init -)"

  Created backup: .zshrc.goenv-backup.20241030-143025
  ✓ Added goenv initialization

💻 Checking for VS Code...
  Set up VS Code integration for this directory? [y/N]: y
  ✓ Configured VS Code to use environment variables

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Setup Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Changes made:
  • Backed up ~/.zshrc
  • Added goenv init to ~/.zshrc
  • Configured VS Code for current directory

🎯 Next steps:
  1. Restart your shell or run:
     source ~/.zshrc
  2. Install a Go version:
     goenv install 1.23.2
  3. Set it as your default:
     goenv global 1.23.2

🎉 Done! Run 'goenv doctor' to verify your setup.
```

**Options:**

- `--yes`, `-y` - Auto-accept all prompts (non-interactive mode)
- `--shell <name>` - Force specific shell (bash, zsh, fish, powershell, cmd)
- `--skip-vscode` - Skip VS Code integration setup
- `--skip-shell` - Skip shell profile setup
- `--dry-run` - Show what would be done without making changes
- `--non-interactive` - Disable all interactive prompts
- `--verify` - Run `goenv doctor` after setup to verify configuration

**What It Does:**

1. **Shell Detection**

   - Automatically detects your shell (bash, zsh, fish, PowerShell, cmd)
   - Finds the appropriate profile file (`.bashrc`, `.zshrc`, `config.fish`, etc.)
   - Shows you what will be added before making changes

2. **Shell Profile Setup**

   - Creates backup of your profile file
   - Adds goenv initialization code
   - Checks if already configured to avoid duplicates
   - Safe to run multiple times

3. **VS Code Integration** (optional)

   - Detects if VS Code is being used (looks for `.vscode/` directory)
   - Configures VS Code settings to use goenv
   - Sets up `go.goroot`, `go.gopath`, and `go.toolsGopath`
   - Prompts before making changes

4. **Summary & Next Steps**
   - Shows what was changed
   - Provides clear next steps
   - Optionally runs `goenv doctor` to verify

**Examples:**

```shell
# Interactive setup (recommended for first-time users)
> goenv setup

# Quick setup with auto-accept
> goenv setup --yes

# Setup with verification
> goenv setup --yes --verify

# Setup for specific shell
> goenv setup --shell bash

# Dry run to see what would happen
> goenv setup --dry-run

# Skip VS Code setup
> goenv setup --skip-vscode

# CI/CD non-interactive setup
> goenv setup --yes --skip-vscode --verify
```

**Shell-Specific Configuration:**

The command automatically handles shell-specific syntax:

- **bash/zsh**: Adds to `~/.bashrc` or `~/.zshrc`

  ```bash
  eval "$(goenv init -)"
  ```

- **fish**: Adds to `~/.config/fish/config.fish`

  ```fish
  status --is-interactive; and goenv init - | source
  ```

- **PowerShell**: Adds to `$PROFILE`

  ```powershell
  & goenv init - | Invoke-Expression
  ```

- **cmd**: Updates `AUTORUN` registry key (Windows)

**Safety Features:**

- **Backup Creation**: Always creates timestamped backup before modifying files
- **Duplicate Detection**: Won't add initialization code if already present
- **Dry Run Mode**: Preview changes without applying them
- **Safe Re-runs**: Can be run multiple times without issues

**Common Scenarios:**

```shell
# New installation - first time setup
> goenv setup
# Follow prompts to configure shell and VS Code

# After installing on a new system
> goenv setup --yes --verify
# Quick setup with verification

# Multiple shells - configure different shell
> goenv setup --shell fish

# Project-specific VS Code setup
cd my-project
goenv setup --skip-shell
# Only configure VS Code for this project

# CI/CD pipeline setup
> goenv setup --yes --skip-vscode --non-interactive
# Automated setup without prompts or VS Code
```

**Troubleshooting:**

If setup doesn't work as expected:

1. **Verify initialization**: Run `goenv status` to check
2. **Source profile manually**: `source ~/.zshrc` (or your shell's profile)
3. **Check for conflicts**: Look for other Go installations in PATH
4. **Run diagnostics**: `goenv doctor --fix` for automated repairs
5. **Re-run setup**: Safe to run `goenv setup` again

**Related Commands:**

- [`goenv init`](#goenv-init) - Manual shell initialization
- [`goenv doctor`](#goenv-doctor) - Diagnose and fix configuration issues
- [`goenv status`](#goenv-status) - Quick health check
- [`goenv get-started`](#goenv-get-started) - Interactive beginner's guide
- [`goenv vscode`](#goenv-vscode) - VS Code-specific configuration

## `goenv install`

Install a Go version (using `go-build`). It's required that the version is a known installable definition by `go-build`. Alternatively, supply `latest` as an argument to install the latest version available to goenv.

**Smart Version Detection:** When no version is specified, `goenv install` automatically detects the desired version:

1. Checks for `.go-version` in current directory or parent directories
2. Checks for `go.mod` and uses the `go` directive version
3. Falls back to installing the latest stable version

```shell
# Auto-detect from .go-version or go.mod (recommended)
> cd my-project
> cat .go-version
1.23.0
> goenv install
📍 Detected version 1.23.0 from .go-version
Installing Go 1.23.0...

# Explicit version (overrides project files)
> goenv install 1.11.1

# Install latest stable (in directory without version files)
> cd /tmp/empty-dir
> goenv install
ℹ️  No version file found, installing latest stable: 1.25.3
Installing Go 1.25.3...
```

**Benefits:**

- 🎯 No need to specify version twice (once in `.go-version`, again in command)
- ✅ Matches industry standards (nvm, rbenv, pyenv)
- 🚀 Works seamlessly in CI/CD and development
- 🔄 Explicit versions always override auto-detection

**Partial Version Resolution:** When you specify a partial version (e.g., `1.21`), goenv automatically resolves it to the latest matching patch version:

```shell
# Install latest 1.21.x patch
> goenv install 1.21
🔍 Resolved 1.21 to 1.21.13 (latest patch)
Installing Go 1.21.13...
✓ Installed Go 1.21.13

# Works with auto-detection too
> echo "go 1.22" > go.mod
> goenv install
📍 Detected version 1.22 from go.mod
🔍 Resolved 1.22 to 1.22.9 (latest patch)
Installing Go 1.22.9...
✓ Installed Go 1.22.9

# Exact versions work as always
> goenv install 1.21.5
Installing Go 1.21.5...
✓ Installed Go 1.21.5
```

**How resolution works:**

- `1.21` → resolves to latest `1.21.x` (e.g., `1.21.13`)
- `1.22` → resolves to latest `1.22.x` (e.g., `1.22.9`)
- `1.21.5` → exact match, installs `1.21.5`
- Version not found → shows helpful error with available versions

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

### Security: Checksum Verification

**All Go downloads are automatically verified** using SHA256 checksums from go.dev's official API.

**How it works:**

1. goenv fetches official SHA256 checksums from `go.dev/dl/?mode=json&include=all`
2. During download, computes SHA256 hash in real-time
3. After download completes, compares computed hash against expected hash
4. Installation **fails immediately** if checksums don't match

**Implementation:** `internal/install/installer.go:156-221`

```shell
# Example output with verbose mode
> goenv install 1.24.3 --verbose
Downloading Go 1.24.3 for darwin-arm64...
Downloading from: https://go.dev/dl/go1.24.3.darwin-arm64.tar.gz
Download completed and verified  # ✅ Checksum verified
Installing to /Users/user/.goenv/versions/1.24.3...
Successfully installed Go 1.24.3
```

**What happens on checksum mismatch:**

```shell
> goenv install 1.24.3
Downloading Go 1.24.3...
Error: checksum verification failed: expected abc123..., got def456...
# Installation aborted, temp file removed
```

**Security guarantees:**

- ✅ Protects against corrupted downloads
- ✅ Protects against man-in-the-middle attacks (when using HTTPS)
- ✅ Protects against compromised mirrors (mirror downloads fallback to official if checksums fail)
- ✅ Automatic - no configuration needed

**Custom mirrors:** Checksum verification still applies even when using `GO_BUILD_MIRROR_URL`. If mirror serves corrupted files, installation fails and falls back to official go.dev source.

See [Using a Custom Mirror](#using-a-custom-mirror) for mirror configuration.

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

# Conditional installation in Makefile
.PHONY: setup
setup:
	@if ! goenv installed $(GO_VERSION) &>/dev/null; then \
		echo "Installing Go $(GO_VERSION)..."; \
		goenv install $(GO_VERSION); \
	fi
	@goenv use $(GO_VERSION)

# CI: Install only if missing (cache-friendly)
#!/bin/bash
VERSION_FILE=".go-version"
if [ -f "$VERSION_FILE" ]; then
  REQUIRED_VERSION=$(cat "$VERSION_FILE")
  if ! goenv installed "$REQUIRED_VERSION" &>/dev/null; then
    echo "::group::Installing Go $REQUIRED_VERSION"
    goenv install "$REQUIRED_VERSION"
    echo "::endgroup::"
  else
    echo "✓ Go $REQUIRED_VERSION already installed"
  fi
  goenv use "$REQUIRED_VERSION"
fi

# Get installed count for reporting
INSTALLED_COUNT=$(goenv list --bare | wc -l)
echo "Installed Go versions: $INSTALLED_COUNT"

# Check system Go availability
if goenv installed system &>/dev/null; then
  SYSTEM_GO=$(goenv installed system)
  echo "System Go available at: $SYSTEM_GO"
else
  echo "No system Go installation found"
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

- Compliance audits (SOC 2, ISO 27001, etc.)
- System inventory reports
- Tracking installed versions across environments
- License compliance verification
- Vulnerability management
- Change management documentation

**For detailed compliance examples, see:** [Compliance Use Cases Guide](../advanced/COMPLIANCE_USE_CASES.md)

## `goenv local`

> **Note:** This is a legacy command. Use [`goenv use <version>`](#goenv-use) instead for a more consistent interface.
>
> **For backward compatibility:** This command still works as expected, but `goenv use` is recommended as it provides a unified interface for both local and global version management.

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

## `goenv tools sync-tools`

Sync/replicate installed Go tools from one Go version to another.

This command discovers tools in the source version and reinstalls them (from source) in the target version. The source version remains unchanged - think of this as "syncing" or "replicating" your tool setup rather than "moving" tools.

**Smart defaults:**

- No args: Sync from version with most tools → current version
- One arg: Sync from that version → current version
- Two args: Sync from source → target (explicit control)

**Usage:**

```shell
# Auto-sync: finds best source, syncs to current version
> goenv tools sync-tools

# Sync from specific version to current version
> goenv tools sync-tools 1.24.1

# Explicit source and target
> goenv tools sync-tools 1.24.1 1.25.2

# Preview auto-sync
> goenv tools sync-tools --dry-run

# Sync only specific tools
> goenv tools sync-tools 1.24.1 --select gopls,delve

# Exclude certain tools
> goenv tools sync-tools 1.24.1 --exclude staticcheck
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

### ⚠️ Current State (v3.0)

**What this is:** A convenience wrapper that runs SBOM tools (cyclonedx-gomod, syft) with the correct Go version and environment.

**What this is NOT:** A full SBOM management solution. It does not validate, sign, or analyze SBOMs—it only generates them using the underlying tools.

**Roadmap:** Future versions will add validation, policy enforcement, signing, vulnerability scanning, and compliance reporting. See [SBOM Roadmap](../roadmap/SBOM_ROADMAP.md) for details.

**Alternative approach:** Advanced users can run SBOM tools directly:

```shell
# Instead of the wrapper:
goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json

# You can run directly:
goenv exec cyclonedx-gomod -json -output sbom.json
```

Both approaches produce identical results. The wrapper provides convenience and cross-tool consistency.

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

| Option           | Type   | Description                                                                                   | Example                                    |
| ---------------- | ------ | --------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `--tool`         | string | SBOM tool to use                                                                              | `cyclonedx-gomod`, `syft`                  |
| `--format`       | string | Output format                                                                                 | `cyclonedx-json`, `spdx-json`, `table`     |
| `-o, --output`   | string | Output file path (default: `sbom.json`, or stdout if `-`)                                     | `sbom.cdx.json`, `-` (stdout)              |
| `--dir`          | string | Project directory to scan (default: `.`)                                                      | `/path/to/project`                         |
| `--image`        | string | Container image to scan (**syft only**)                                                       | `ghcr.io/myapp:v1.0.0`                     |
| `--modules-only` | bool   | Only scan Go modules, exclude stdlib/main (**cyclonedx-gomod only**)                          | `--modules-only`                           |
| `--offline`      | bool   | Offline mode - avoid network access for module metadata (**recommended for reproducibility**) | `--offline`                                |
| `--tool-args`    | string | Additional arguments to pass directly to the underlying tool                                  | `--tool-args="--quiet --exclude-dev-deps"` |

**Supported Tools:**

| Tool              | Best For                           | Image Support | Offline Mode | Formats Supported                      |
| ----------------- | ---------------------------------- | ------------- | ------------ | -------------------------------------- |
| `cyclonedx-gomod` | Go modules (fast, lightweight)     | ❌ No         | ✅ Yes       | `cyclonedx-json`, `cyclonedx-xml`      |
| `syft`            | Multi-language, containers, images | ✅ Yes        | ⚠️ Partial   | `spdx-json`, `cyclonedx-json`, `table` |

**Current Benefits (v3.0):**

- ✅ **Reproducible SBOMs** - Pinned Go and tool versions ensure consistent output
- ✅ **Fast generation** - Shared module cache across Go versions
- ✅ **Cross-platform** - Same command works on Linux, macOS, Windows
- ✅ **CI-friendly** - Provenance to stderr, SBOM to stdout/file

**Future Benefits (Roadmap):**

- 🔜 **Validation** - Enforce SBOM completeness and policy compliance (v3.1)
- 🔜 **Signing** - Cryptographic attestation for supply chain security (v3.2)
- 🔜 **Automation** - Hooks integration for zero-touch SBOM generation (v3.3)
- 🔜 **Analysis** - Diff SBOMs to detect dependency drift (v3.4)
- 🔜 **Security** - Integrated vulnerability scanning workflows (v3.5)

See [SBOM Roadmap](../roadmap/SBOM_ROADMAP.md) for timeline and details.

### 🔒 Secured CI/CD Example (Recommended)

For reproducible, auditable SBOMs in production CI/CD pipelines:

**Characteristics of secured SBOM generation:**

- 🔒 **Fixed versions** - Pinned Go and tool versions (no `@latest`)
- 📴 **Offline mode** - No network calls during generation (reproducibility)
- 🔐 **Verification** - Validate SBOM format and content
- 📦 **Attestation-ready** - Upload as signed artifact

**GitHub Actions Example:**

```yaml
# .github/workflows/sbom.yml
name: Generate SBOM

on:
  push:
    branches: [main]
  pull_request:
  release:
    types: [published]

jobs:
  sbom:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write # For uploading SBOM artifacts

    steps:
      - uses: actions/checkout@v4

      - name: Setup goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
          eval "$(goenv init -)"

      - name: Setup Go version (PINNED)
        run: goenv use 1.25.2 --yes # ✅ Fixed version

      - name: Install SBOM tool (PINNED VERSION)
        run: goenv tools install cyclonedx-gomod@v1.6.0 # ✅ Fixed tool version

      - name: Generate SBOM (OFFLINE, REPRODUCIBLE)
        run: |
          goenv sbom project \
            --tool=cyclonedx-gomod \
            --format=cyclonedx-json \
            --output=sbom.cdx.json \
            --offline  # ✅ No network calls during generation

      - name: Verify SBOM integrity
        run: |
          # Validate CycloneDX format
          jq -e '.bomFormat == "CycloneDX"' sbom.cdx.json
          jq -e '.specVersion' sbom.cdx.json

          # Verify components exist
          COMPONENT_COUNT=$(jq '.components | length' sbom.cdx.json)
          echo "SBOM contains $COMPONENT_COUNT components"
          test $COMPONENT_COUNT -gt 0

          # Check for required metadata
          jq -e '.metadata.component.name' sbom.cdx.json

      - name: Upload SBOM artifact
        uses: actions/upload-artifact@v4
        with:
          name: sbom-cyclonedx
          path: sbom.cdx.json
          retention-days: 90

      # Optional: Sign and attest SBOM
      - name: Sign SBOM with cosign
        if: github.event_name == 'release'
        run: |
          # Requires cosign setup
          cosign sign-blob --key cosign.key sbom.cdx.json > sbom.cdx.json.sig
```

**GitLab CI Example:**

```yaml
# .gitlab-ci.yml
generate-sbom:
  stage: security
  image: ubuntu:latest

  variables:
    GO_VERSION: "1.25.2" # ✅ Fixed Go version
    SBOM_TOOL_VERSION: "v1.6.0" # ✅ Fixed tool version

  before_script:
    - curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
    - export PATH="$HOME/.goenv/bin:$PATH"
    - eval "$(goenv init -)"

  script:
    - goenv use $GO_VERSION --yes
    - goenv tools install cyclonedx-gomod@$SBOM_TOOL_VERSION
    - goenv sbom project \
      --tool=cyclonedx-gomod \
      --format=cyclonedx-json \
      --output=sbom.cdx.json \
      --offline # ✅ Offline mode

    # Verification
    - jq -e '.bomFormat == "CycloneDX"' sbom.cdx.json
    - jq '.components | length' sbom.cdx.json

  artifacts:
    paths:
      - sbom.cdx.json
    expire_in: 90 days
    reports:
      cyclonedx: sbom.cdx.json # GitLab SBOM integration
```

**Why this approach is secured:**

1. **Reproducibility** - Same versions → same SBOM → verifiable builds
2. **No surprises** - Offline mode prevents unexpected network dependencies
3. **Auditability** - Fixed versions allow audit trail verification
4. **Supply chain security** - Pinned tools reduce attack surface
5. **Compliance-ready** - Meets SLSA, SSDF, and SBOM requirements

### Alternative: Direct Tool Execution

For advanced users or when you need full access to tool-specific features, you can run SBOM tools directly with `goenv exec`:

```bash
# CycloneDX with all native flags
goenv exec cyclonedx-gomod \
  -json \
  -output sbom.cdx.json \
  -licenses \
  -std \
  -verbose

# Syft with advanced filtering
goenv exec syft . \
  -o cyclonedx-json=sbom.json \
  --exclude '/test/**' \
  --catalogers go-module-file-cataloger \
  --platform linux/amd64
```

**When to use direct execution:**

- ✅ You need tool-specific flags not exposed by the wrapper
- ✅ You're comfortable with the underlying tool's CLI
- ✅ You want to chain tools with pipes or custom scripts
- ✅ You're debugging SBOM generation issues

**When to use the wrapper:**

- ✅ You want a standardized, cross-tool interface
- ✅ You're onboarding new team members
- ✅ You need CI templates that "just work"
- ✅ You want future features (validation, signing, etc.) when available

### CycloneDX vs SPDX Format

**When to use CycloneDX (cyclonedx-gomod):**

- ✅ Fast, lightweight Go module scanning
- ✅ Offline mode supported
- ✅ Excellent for pure Go projects
- ✅ Standard in many compliance frameworks

**When to use SPDX/Syft:**

- ✅ Multi-language projects (Go + Python + Node.js)
- ✅ Container image scanning required
- ✅ Broader tool ecosystem integration
- ✅ SPDX specifically required by policy

### Advanced: Container Image Scanning (Syft)

Scan container images for vulnerabilities and dependencies:

```bash
# Build and scan your container
docker build -t myapp:latest .

goenv sbom project \
  --tool=syft \
  --image=myapp:latest \
  --format=spdx-json \
  --output=sbom.spdx.json
```

**Note:** Image scanning requires Docker and the image to be available locally or in a registry.

### Troubleshooting

**"Tool not found" error:**

```bash
# Install the required tool first
goenv tools install cyclonedx-gomod@latest
goenv tools install syft@latest
```

**"Module not found" in offline mode:**

```bash
# Download dependencies first
go mod download
go mod verify

# Then generate SBOM offline
goenv sbom project --tool=cyclonedx-gomod --offline
```

**Empty or minimal SBOM:**

```bash
# Ensure go.mod exists and dependencies are vendored/downloaded
go mod tidy
go mod download

# Try without --modules-only flag
goenv sbom project --tool=cyclonedx-gomod
```

**CI/CD specific issues:**

```bash
# Ensure goenv is initialized in CI environment
eval "$(goenv init -)"

# Use --yes flag to avoid prompts
goenv use 1.25.2 --yes

# Verify tool installation
goenv tools list
```

### 🔐 Security Best Practices

with:
name: sbom-cyclonedx
path: sbom.cdx.json

````

**2. Syft with Fixed Versions (Multi-language projects)**

```yaml
# GitLab CI - SBOM with Syft (SPDX format)
sbom:
  stage: security
  script:
    - goenv use 1.25.2 --yes
    - goenv tools install syft@v1.0.0 # Pin syft version

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
````

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

Uninstalls the specified Go version(s). Supports partial version matching and multi-version operations.

**Usage:**

```shell
# Interactive mode - uninstall specific version
> goenv uninstall 1.21.13
Really uninstall Go 1.21.13? This cannot be undone [y/N]: y
Uninstalling Go 1.21.13...
✓ Uninstalled Go 1.21.13

# Partial version - automatically resolves to latest installed
> goenv uninstall 1.21
🔍 Resolved 1.21 to 1.21.13
Really uninstall Go 1.21.13? This cannot be undone [y/N]: y
✓ Uninstalled Go 1.21.13

# Multiple matches - shows interactive menu
> goenv uninstall 1.22
Found 3 installed versions matching 1.22. Which would you like to uninstall?
  1) 1.22.8 (latest)
  2) 1.22.2
  3) 1.22.0
  4) All of the above

Enter selection: 1
Really uninstall Go 1.22.8? This cannot be undone [y/N]: y
✓ Uninstalled Go 1.22.8

# Uninstall all matching versions
> goenv uninstall 1.21 --all
✓ Uninstalled Go 1.21.13
✓ Uninstalled Go 1.21.5
✓ Uninstalled Go 1.21.0

# Non-interactive mode (picks latest automatically)
> goenv uninstall 1.21 --yes
🔍 Resolved 1.21 to 1.21.13 (latest installed)
✓ Uninstalled Go 1.21.13
```

**Options:**

- `--all` - Uninstall all versions matching the given prefix (e.g., all 1.21.x versions)
- `--yes, -y` - Auto-confirm all prompts (non-interactive mode, picks latest if multiple matches)
- `--debug, -d` - Enable debug output
- `--quiet, -q` - Suppress progress output (only show errors)

**Smart Version Resolution:**

When you specify a partial version (e.g., `1.21`), goenv intelligently resolves it:

- **Single match**: Uninstalls that version directly
- **Multiple matches (interactive mode)**: Shows a menu to choose which version(s) to uninstall
- **Multiple matches (non-interactive mode)**: Uninstalls the latest matching version
- **Multiple matches with `--all`**: Uninstalls all matching versions

**Interactive Selection:**

When multiple versions match your request in interactive mode, you get a numbered menu:

```shell
> goenv uninstall 1.22
Found 3 installed versions matching 1.22. Which would you like to uninstall?
  1) 1.22.8 (latest)
  2) 1.22.2
  3) 1.22.0
  4) All of the above

Enter selection [1-4, default=1]: _
```

Choose a number to uninstall that specific version, or select "All of the above" to uninstall all matching versions.

**Non-Interactive Environments (CI/CD):**

For scripts and automation, use `--yes` to bypass prompts:

```bash
# CI/CD pipeline - uninstall specific version
goenv uninstall 1.23.0 --yes

# Uninstall latest matching version
goenv uninstall 1.21 --yes
# Resolves to latest 1.21.x and uninstalls without prompting

# Uninstall all matching versions
goenv uninstall 1.20 --all --yes
# Uninstalls all 1.20.x versions without prompting

# Automated cleanup script - remove all old versions
for version in $(goenv list --bare | grep -v "$(goenv current --bare)"); do
  goenv uninstall "$version" --yes
done

# Docker multi-stage build cleanup
RUN goenv uninstall 1.22.0 --yes
```

**Examples:**

```bash
# Clean up all 1.21.x versions at once
goenv uninstall 1.21 --all

# Remove a specific version
goenv uninstall 1.22.5

# Interactive selection from multiple versions
goenv uninstall 1.22
# Shows menu if you have 1.22.0, 1.22.2, 1.22.8 installed

# Non-interactive: remove latest 1.21.x
goenv uninstall 1.21 --yes
# Automatically picks 1.21.13 if that's the latest installed

# Batch removal in scripts
goenv uninstall 1.20 --all --yes
# Removes all 1.20.x versions without any prompts
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

- `--check`, `-c` - Check for updates without installing
- `--force`, `-f` - Force update even if already up-to-date

### Installation Type Detection

The `update` command automatically detects how goenv was installed and uses the appropriate update method. The detection follows this priority order:

1. **Check if binary is in a git repository**

   - Resolves symlinks to find actual binary location
   - Checks for `.git` directory in binary's parent directory
   - Validates it's a real git repo with `git rev-parse --git-dir`

2. **Check if GOENV_ROOT is a git repository**

   - Standard installation via git clone
   - Most common for development setups
   - Checks `$GOENV_ROOT/.git` directory

3. **Fall back to binary installation**
   - No git repository found
   - Assumes standalone binary installation
   - Uses GitHub releases API for updates

**Example detection output (debug mode):**

```shell
> GOENV_DEBUG=1 goenv update
Debug: Installation type: git
Debug: Installation path: /Users/user/.goenv
```

### Installation Methods

#### Git-based installations (recommended)

**Advantages:**

- Instant updates via `git pull`
- See exact changes being applied
- Easy to rollback with `git reset`
- No binary replacement needed

**How it works:**

1. Fetches latest changes from `origin`
2. Shows commit log of changes
3. Checks for uncommitted local changes
4. Runs `git pull` to update
5. Displays before/after commit hashes

**Requirements:**

- Git must be installed and in PATH
- GOENV_ROOT must be a git clone of the repository
- Network access to GitHub

**Example output:**

```shell
> goenv update
🔄 Checking for goenv updates...
📡 Fetching latest changes...
📝 Changes:
   • abc1234 Fix vscode init backup creation
   • def5678 Add comprehensive test coverage
   • ghi9012 Update documentation

⬇️  Updating goenv...
✅ goenv updated successfully!
   Updated from a1b2c3d to x7y8z9a

💡 Restart your shell to use the new version:
   exec $SHELL
```

#### Binary installations

**Advantages:**

- No git dependency required
- Works in restricted environments
- Simple standalone deployment
- SHA256 checksum verification

**How it works:**

1. Queries GitHub releases API for latest version
2. Uses ETag caching to avoid redundant downloads
3. Downloads platform-specific binary (goenv_VERSION_OS_ARCH)
4. Verifies SHA256 checksum from SHA256SUMS file
5. Creates backup of current binary
6. Replaces binary atomically
7. Removes backup on success

**Requirements:**

- Write permission to binary location
- Network access to GitHub releases
- HTTPS support (uses strict TLS)

**Example output:**

```shell
> goenv update
🔄 Checking for goenv updates...
📦 Detected binary installation
   Binary location: /usr/local/bin/goenv

🔍 Checking GitHub releases...
⬇️  Downloading goenv v3.0.0...
🔐 Verifying checksum...
✅ Checksum verified
💾 Creating backup...
🔄 Replacing binary...

✅ goenv updated successfully!
   Updated from v2.9.0 to v3.0.0
```

### GitHub API Implementation

#### ETag Caching

To minimize bandwidth and respect GitHub API rate limits, binary updates use HTTP conditional requests:

**Cache file location:** `$GOENV_ROOT/cache/update-etag`

**How it works:**

1. **First request:** GET to GitHub releases API, saves ETag header
2. **Subsequent requests:** Sends `If-None-Match: <etag>` header
3. **No update:** GitHub returns `304 Not Modified` (no body, minimal bandwidth)
4. **New release:** GitHub returns `200 OK` with full release data

**Benefits:**

- Reduces bandwidth usage (304 responses are ~200 bytes)
- Faster update checks (no JSON parsing needed)
- Lower API rate limit consumption
- Cache persists across goenv invocations

**Cache file structure:**

```shell
# File: ~/.goenv/cache/update-etag
# Contents: Raw ETag value from GitHub
W/"a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

**Cache permissions:**

- Directory: `0700` (user read/write/execute only)
- File: `0600` (user read/write only)
- Prevents unauthorized access to cache data

#### Rate Limits & Authentication

**Unauthenticated requests:**

- **Limit:** 60 requests per hour per IP address
- **Resets:** Every hour on the hour
- **Sufficient for:** Manual updates, occasional checks

**Authenticated requests (recommended for CI/CD):**

- **Limit:** 5,000 requests per hour
- **Requires:** `GITHUB_TOKEN` environment variable
- **Use case:** Automated update checks, CI pipelines, frequent updates

**Setting up authentication:**

```shell
# 1. Create a GitHub Personal Access Token (PAT)
#    - Go to: https://github.com/settings/tokens
#    - Fine-grained PAT: No permissions needed (public repo read)
#    - Classic PAT: No scopes needed (public data only)

# 2. Export token in your shell profile (~/.bashrc, ~/.zshrc)
export GITHUB_TOKEN=ghp_your_personal_access_token_here

# 3. Verify it's working (check debug output)
GOENV_DEBUG=1 goenv update --check
# Should show: "Using GitHub token for higher rate limits"
```

**Token best practices:**

- Use fine-grained tokens with minimal permissions
- Set expiration dates (e.g., 90 days)
- Rotate tokens regularly
- Store in secure password manager
- Never commit tokens to version control
- Use CI-specific tokens for pipelines

**Rate limit headers returned:**

- `X-RateLimit-Limit` - Maximum requests per hour
- `X-RateLimit-Remaining` - Requests remaining in current window
- `X-RateLimit-Reset` - Unix timestamp when limit resets

**Handling rate limit exceeded:**

```shell
> goenv update
Error: GitHub API rate limit exceeded. Resets at 2025-10-28T15:00:00Z (in 42m30s)

# Solution 1: Wait for reset
# Solution 2: Set GITHUB_TOKEN
export GITHUB_TOKEN=ghp_your_token
goenv update

# Solution 3: Download manually
https://github.com/go-nv/goenv/releases
```

#### Retry Logic & Backoff

**Rate limiting (403/429 responses):**

- Attempts up to 3 retries with exponential backoff
- Backoff delays: 1s, 2s, 4s
- Honors `Retry-After` header if present (max 60s)
- Displays helpful error with reset time

**Network errors:**

- Single attempt (no retries for connection failures)
- Timeout: 10 seconds for API requests
- Timeout: 60 seconds for binary downloads

#### Security Features

**HTTPS enforcement:**

- All API requests use HTTPS only
- No fallback to insecure HTTP
- TLS certificate verification enabled

**SHA256 checksum verification:**

- Downloads `SHA256SUMS` file from release
- Verifies binary matches published checksum
- Format: `<hash>  <filename>` (standard sha256sum format)
- Fails update if checksums mismatch
- Warns if checksums unavailable (older releases)

**Atomic binary replacement:**

1. Download to temporary file
2. Verify checksum
3. Create backup of current binary
4. Replace binary atomically with `os.Rename`
5. Remove backup on success
6. Restore backup on failure

### Error Recovery

#### Git not found (git-based installations)

**Error message:**

```
Error: git not found in PATH - cannot update git-based installation
```

**Platform-specific solutions:**

**macOS:**

```shell
# Option 1: Install Xcode Command Line Tools
xcode-select --install

# Option 2: Install via Homebrew
brew install git
```

**Windows:**

```powershell
# Option 1: Install Git for Windows
# Download from: https://git-scm.com/download/win

# Option 2: Install via winget
winget install Git.Git
```

**Linux:**

```shell
# Debian/Ubuntu
sudo apt-get install git

# RHEL/CentOS/Fedora
sudo yum install git

# Arch Linux
sudo pacman -S git
```

**Alternative:** Switch to binary installation by downloading from [GitHub releases](https://github.com/go-nv/goenv/releases)

#### Permission denied (binary installations)

**Error message:**

```
Error: cannot update binary: permission denied

To fix this:
  • Run with elevated permissions: sudo goenv update
  • Or install goenv to a user-writeable path (e.g., ~/.local/bin/)
```

**Solutions by platform:**

**macOS/Linux:**

```shell
# Option 1: Run with sudo
sudo goenv update

# Option 2: Move to user-owned directory
mkdir -p ~/.local/bin
sudo mv /usr/local/bin/goenv ~/.local/bin/
# Add ~/.local/bin to PATH in ~/.bashrc or ~/.zshrc

# Option 3: Change binary ownership
sudo chown $USER /usr/local/bin/goenv
goenv update
```

**Windows:**

```powershell
# Option 1: Run as Administrator
# Right-click PowerShell → "Run as Administrator"
goenv update

# Option 2: Install to user directory
# Move goenv.exe to: %LOCALAPPDATA%\goenv\bin\
# e.g., C:\Users\YourName\AppData\Local\goenv\bin\
# Add to PATH in System Environment Variables
```

**Manual installation alternative:**

1. Download latest release: https://github.com/go-nv/goenv/releases
2. Extract platform-specific binary
3. Replace existing binary manually

#### Uncommitted changes (git-based installations)

**Warning message:**

```
⚠️  Warning: You have uncommitted changes in goenv directory
   The update may fail or overwrite your changes.

Use --force to update anyway, or commit/stash your changes first.
```

**Solutions:**

```shell
# Option 1: Commit your changes
cd $GOENV_ROOT
git add .
git commit -m "Local customizations"
goenv update

# Option 2: Stash your changes
cd $GOENV_ROOT
git stash
goenv update
git stash pop

# Option 3: Force update (may lose changes)
goenv update --force
```

### Examples

**Check for updates without installing:**

```shell
> goenv update --check
🔄 Checking for goenv updates...
🆕 Update available!
   Current:  v2.9.0
   Latest:   v3.0.0

Run 'goenv update' to install the update.
```

**Update with debug output:**

```shell
> GOENV_DEBUG=1 goenv update
Debug: Installation type: binary
Debug: Installation path: /usr/local/bin/goenv
Debug: Latest version: v3.0.0
Debug: Download URL: https://github.com/go-nv/goenv/releases/download/v3.0.0/goenv_3.0.0_darwin_arm64
🔄 Checking for goenv updates...
Using GitHub token for higher rate limits
...
```

**Force update when already current:**

```shell
> goenv update --force
🔄 Checking for goenv updates...
📝 Changes:
   (No new commits)

⬇️  Updating goenv...
✅ goenv updated successfully!
```

### Troubleshooting

**"GitHub API rate limit exceeded"**

- Wait for rate limit to reset (check error message for time)
- Set `GITHUB_TOKEN` environment variable for 5,000/hour limit
- Use `--check` to verify status without triggering download

**"Checksum verification failed"**

- Possible causes: corrupted download, network proxy, or security issue
- Run with `GOENV_DEBUG=1` to see checksum details
- Try download again: network hiccup may have corrupted file
- If persistent, report as security issue

**"304 Not Modified" in debug output**

- This is normal and good! ETag cache is working
- Means no update available since last check
- No bandwidth wasted, no rate limit consumed

**Update succeeds but version unchanged**

- Git installations: Run `exec $SHELL` to reload shell
- Binary installations: Restart terminal or re-run command
- Check installation: `which goenv` and `goenv version`

## `goenv tools update`

Update installed Go tools to their latest versions for the current or all Go versions.

**Usage:**

```shell
# Update all tools for current Go version
goenv tools update

# Update tools across ALL Go versions
goenv tools update --all

# Check for updates without installing
goenv tools update --check

# Update only a specific tool
goenv tools update --tool gopls

# Update specific tool across all versions
goenv tools update --tool gopls --all

# Show what would be updated (dry run)
goenv tools update --dry-run

# Update to specific version (default: latest)
goenv tools update --version v1.2.3
```

**Options:**

- `--all` - Update tools across all installed Go versions
- `--check`, `-c` - Check for updates without installing
- `--tool <name>`, `-t` - Update only the specified tool
- `--dry-run`, `-n` - Show what would be updated without actually updating
- `--version <version>` - Target version (default: latest)

**Example Output:**

```shell
$ goenv tools update

🔄 Checking for tool updates in Go 1.23.0...

Found 3 tool(s):

  • gopls (v0.12.0) → v0.13.2 available ⬆️
  • staticcheck (v0.4.6) - up to date ✅
  • gofmt - up to date ✅

📦 Updating tools...

  Updating gopls... ✅

✅ Updated 1 tool(s) successfully
```

**With --all flag:**

```shell
$ goenv tools update --all

🔄 Checking for tool updates across 3 Go version(s)...

Outdated tools found:
  • gopls: 2 versions need updating
  • staticcheck: 1 version needs updating

📦 Updating tools...

Go 1.21.0:
  Updating gopls... ✅
  Updating staticcheck... ✅

Go 1.23.0:
  Updating gopls... ✅

✅ Updated 3 tool(s) across 2 version(s)
```

This command updates tools installed with `go install` in your Go version's GOPATH. Use `--all` to maintain consistency across all installed Go versions.

## `goenv vscode`

Manage Visual Studio Code integration with goenv.

```shell
> goenv vscode
Commands to configure and manage Visual Studio Code integration with goenv

Available Commands:
  setup       Complete VS Code setup (init + sync + doctor) - NEW!
  init        Initialize VS Code workspace for goenv
  sync        Update VS Code settings with current Go version
```

### `goenv vscode setup`

**NEW!** Complete VS Code setup in one command - perfect for new users!

This unified command combines `init`, `sync`, and `doctor` to set up VS Code integration in a single step. It's the fastest way to get started.

**Usage:**

```shell
# Navigate to your project
cd ~/projects/myapp

# Complete setup (does everything)
> goenv vscode setup
✓ Initializing VS Code workspace...
✓ Created/updated .vscode/settings.json
✓ Created/updated .vscode/extensions.json
✓ Syncing with current Go version (1.25.2)...
✓ Updated settings for Go 1.25.2
✓ Running diagnostics...
✓ All checks passed!

✨ VS Code is ready to use with goenv!
```

**What it does:**

1. **Initialize** - Creates `.vscode/settings.json` and `.vscode/extensions.json`
2. **Sync** - Updates settings with current Go version
3. **Doctor** - Verifies installation and shows any issues

**This is equivalent to running:**

```bash
goenv vscode init    # Create configuration
goenv vscode sync    # Update with current version
goenv doctor         # Verify everything works
```

**Flags:**

- `--template <name>` - Use specific template (basic, advanced, monorepo)
- `--force` - Overwrite existing settings instead of merging
- `--env-vars` - Use environment variables mode (for terminal-only users)
- `--workspace-paths` - Use workspace-relative paths for portability
- `--versioned-tools` - Use per-version tools directory
- `--dry-run` - Show what would be done without actually doing it
- `--strict` - Exit with error if any diagnostics fail

**Examples:**

```bash
# Basic setup (most common)
goenv vscode setup

# Advanced template with gopls settings
goenv vscode setup --template advanced

# Environment variables mode (terminal-only users)
goenv vscode setup --env-vars

# Force overwrite existing settings
goenv vscode setup --force

# Portable setup for teams
goenv vscode setup --workspace-paths --versioned-tools

# Check what would be done (dry run)
goenv vscode setup --dry-run

# Strict mode for CI/CD (fail on warnings)
goenv vscode setup --strict
```

**Perfect for:**

- 🆕 **First-time users** - Get started quickly without learning multiple commands
- 🚀 **Quick onboarding** - Set up new projects in seconds
- 🔍 **Troubleshooting** - Diagnoses issues and shows clear error messages
- 📦 **CI/CD** - Prepare VS Code workspaces in automated pipelines
- 👥 **Team onboarding** - Single command for new team members

**Output example with issues detected:**

```shell
> goenv vscode setup
✓ Initializing VS Code workspace...
✓ Created .vscode/settings.json
⚠ Warning: Go 1.25.2 is not installed

💡 To fix:
   goenv install 1.25.2

✨ Setup partially complete. Install Go 1.25.2 to finish.
```

**Non-interactive mode (CI/CD):**

```bash
# Use with --force in automated scripts
goenv vscode setup --force --strict

# Or disable interactivity
GOENV_NONINTERACTIVE=1 goenv vscode setup --force
```

**See also:**

- [VS Code Integration Guide](../user-guide/VSCODE_INTEGRATION.md) - Complete setup walkthrough
- [`goenv vscode init`](#goenv-vscode-init) - Initialize only (without sync/doctor)
- [`goenv vscode sync`](#goenv-vscode-sync) - Sync only (after version changes)
- [`goenv doctor`](#goenv-doctor) - Full diagnostics

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

> **Note:** This is a legacy command. Use [`goenv current`](#goenv-current) instead for a more consistent interface.
>
> **For backward compatibility:** This command still works as expected, but `goenv current` is recommended as it provides clearer semantics (showing the "current" version rather than "version").

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
>
> **For automation:** Both `goenv versions --json` and `goenv list --json` produce identical output. The `list` command is recommended as it unifies installed and remote version queries.

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

## 🔄 Legacy Commands (Backward Compatibility)

The following commands are maintained for backward compatibility with the bash implementation of goenv and other version managers. While they continue to work as expected, **we recommend using the [modern unified commands](#-modern-unified-commands-recommended)** for new scripts and workflows.

### Command Migration Guide

| Legacy Command           | Modern Equivalent                            | Why Migrate?                           |
| ------------------------ | -------------------------------------------- | -------------------------------------- |
| `goenv local <version>`  | [`goenv use <version>`](#goenv-use)          | Unified interface for local/global     |
| `goenv global <version>` | [`goenv use <version> --global`](#goenv-use) | Single command with flag               |
| `goenv version`          | [`goenv current`](#goenv-current)            | Clearer semantics                      |
| `goenv versions`         | [`goenv list`](#goenv-list)                  | Consistent with other version managers |

### Why Use Modern Commands?

1. **Consistency**: One command (`use`) for both local and global, vs two commands (`local`/`global`)
2. **Clarity**: `current` is clearer than `version` (which could mean "goenv's version")
3. **Functionality**: `list` supports `--remote` to query available versions, `versions` doesn't
4. **Convention**: Follows patterns from other modern version managers (nvm, rbenv, pyenv)

### Examples

**Before (legacy commands):**

```bash
goenv local 1.25.2
goenv global 1.24.8
goenv version
goenv versions
```

**After (modern commands):**

```bash
goenv use 1.25.2
goenv use 1.24.8 --global
goenv current
goenv list
```

### Backward Compatibility Promise

All legacy commands will continue to work indefinitely. This ensures:

- ✅ Existing scripts don't break
- ✅ CI/CD pipelines keep working
- ✅ Team members can migrate at their own pace
- ✅ Documentation from bash goenv era remains valid

However, new features and functionality will primarily be added to modern commands.

### Deprecation Timeline

**Current Status (v3.x):**

- ✅ Legacy commands work without warnings
- ✅ Hidden from `goenv help` output
- ✅ Documented in dedicated "Legacy Commands" section

**Future Plans:**

**v3.x (Current):**

- Legacy commands continue to work silently
- No breaking changes to existing scripts

**v4.0 (Future - TBD):**

- May add deprecation warnings to stderr:
  ```bash
  $ goenv local 1.23.2
  Warning: 'goenv local' is deprecated. Use 'goenv use 1.23.2' instead.
  Setting local version to 1.23.2...
  ```
- Commands still functional, just with warnings
- Warnings can be suppressed with `GOENV_SILENCE_DEPRECATION_WARNINGS=1`

**v5.0 (Far Future - TBD):**

- Consider removing legacy commands entirely
- Or keep as thin wrappers with strong warnings
- Decision based on community feedback and usage metrics

### Recommendation for New Projects

**✅ Use modern commands:**

```bash
# Modern - Recommended
goenv use 1.25.2
goenv use 1.24.8 --global
goenv current
goenv list
```

**⚠️ Avoid legacy commands in new code:**

```bash
# Legacy - Not recommended for new projects
goenv local 1.25.2
goenv global 1.24.8
goenv version
goenv versions
```

**📋 Migration checklist for existing projects:**

1. Search codebase for legacy command usage: `grep -r "goenv local\|goenv global\|goenv version\|goenv versions" .`
2. Update scripts to use modern equivalents
3. Update CI/CD pipeline configuration
4. Update team documentation
5. Test thoroughly before deploying

### Legacy Command Documentation

For detailed documentation on legacy commands, see their individual sections:

- [`goenv local`](#goenv-local) - Use [`goenv use`](#goenv-use) instead
- [`goenv global`](#goenv-global) - Use [`goenv use --global`](#goenv-use) instead
- [`goenv version`](#goenv-version) - Use [`goenv current`](#goenv-current) instead
- [`goenv versions`](#goenv-versions) - Use [`goenv list`](#goenv-list) instead
