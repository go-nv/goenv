# goenv Quick Reference

One-page cheat sheet for common goenv commands. Perfect for printing or quick lookup.

## Essential Commands

| Task | Command | Notes |
|------|---------|-------|
| **Install goenv** | `curl -sfL https://.../.../install.sh \| bash` | Binary installation (fastest) |
| **First-time setup** | `goenv setup` | 🆕 Automatic shell & IDE configuration |
| **Getting started** | `goenv get-started` | 🆕 Interactive beginner guide |
| **Install Go version** | `goenv install 1.25.2` | Downloads and installs |
| **Set project version** | `goenv use 1.25.2` | Creates `.go-version` file |
| **Set global version** | `goenv use 1.25.2 --global` | Sets default for all projects |
| **Check active version** | `goenv current` | Shows version and source |
| **Check installation status** | `goenv status` | 🆕 Quick health overview |
| **List installed** | `goenv list` | Includes `*` for active |
| **List available** | `goenv list --remote` | All versions from golang.org |
| **Check version usage** | `goenv versions --used` | 🆕 Scan projects for version usage |
| **Version information** | `goenv info 1.25.2` | 🆕 Detailed info & lifecycle status |
| **Compare versions** | `goenv compare 1.21.5 1.23.2` | 🆕 Side-by-side comparison |
| **Uninstall version** | `goenv uninstall 1.24.0` | Removes installation |
| **Update goenv** | `goenv update` | Self-update (git or binary) |
| **Diagnostics** | `goenv doctor` | Health check |
| **Fix issues** | `goenv doctor --fix` | 🆕 Interactive repair mode |
| **Discover commands** | `goenv explore` | 🆕 Browse commands by category |

## Getting Started (New Users)

```bash
# Automatic first-time setup
goenv setup
# Detects shell, adds to profile, configures VS Code
# Safe to run multiple times - won't duplicate config

# Interactive guide
goenv get-started
# Shows step-by-step instructions based on your setup status

# Quick health check
goenv status
# Shows initialization status, current version, installed count

# Comprehensive diagnostics
goenv doctor
# Checks 26+ aspects of your installation

# Fix issues interactively
goenv doctor --fix
# Repairs shell config, duplicates, stale cache

# Discover commands by category
goenv explore
# Browse: Getting Started, Versions, Tools, Diagnostics, etc.
```

## Version Management

```bash
# Install specific version
goenv install 1.25.2

# Install latest
goenv install --latest

# Set for current project
goenv use 1.25.2

# Set globally
goenv use 1.25.2 --global

# Check what's active
goenv current

# Show all installed
goenv list

# Check which versions are used by projects (🆕)
cd ~/work
goenv versions --used
# Shows which versions are in use by scanning .go-version and go.mod files

# Quick scan (immediate subdirectories only)
goenv versions --used --depth 1

# Browse available versions
goenv list --remote

# Only stable releases
goenv list --remote --stable

# JSON output (automation)
goenv list --json
```

## Version Usage Analysis (🆕)

```bash
# Navigate to your projects directory
cd ~/work

# Scan for version usage
goenv versions --used
# Output shows:
#   ✓ Versions used by projects
#   ⚠️  Versions not found (may be safe to remove)
#   📁 List of projects using each version

# Control scan depth
goenv versions --used --depth 5  # Deep scan
goenv versions --used --depth 1  # Quick scan

# Before removing old versions
goenv versions --used
# Check if version shows "Not found" before uninstalling
```

## Version Information & Lifecycle

```bash
# Get detailed version info
goenv info 1.23.2
# Shows: install status, release date, EOL status, size

# JSON output for automation
goenv info 1.23.2 --json

# Compare two versions side-by-side
goenv compare 1.21.5 1.23.2
# Shows: release dates, support status, size diff, recommendations

# Check if version is EOL
goenv info 1.20.0
# Warns if version is no longer supported

# Find recommended upgrade
goenv info $(goenv current --bare)
# Shows upgrade path if current version is EOL

# Compare current with latest
goenv compare $(goenv current --bare) $(goenv list --remote --stable --bare | tail -1)
```

## File-Based Version Selection

```bash
# Create .go-version file
echo "1.25.2" > .go-version

# goenv automatically uses it
goenv current
# Output: 1.25.2 (set by /path/to/project/.go-version)
```

## Tools Management

```bash
# Install specific tool for current version
goenv tools install gopls@latest

# Install tool across ALL Go versions at once
goenv tools install gopls@latest --all

# Uninstall tool from current version
goenv tools uninstall gopls

# Uninstall tool from ALL Go versions
goenv tools uninstall gopls --all

# List tools for current version
goenv tools list

# List tools across all versions
goenv tools list --all

# Check which tools are outdated
goenv tools outdated

# View tool consistency across versions
goenv tools status

# Update tools for current version
goenv tools update

# Update tools across all versions
goenv tools update --all

# Sync tools between specific versions
goenv tools sync-tools 1.21.0 1.23.0

# Manage default tools (auto-installed with new Go versions)
goenv tools default list
goenv tools default init
```

## Cache Management

```bash
# Check cache status
goenv cache status

# Clean build caches
goenv cache clean build

# Clean all caches
goenv cache clean all

# Prune old caches
goenv cache clean all --older-than 30d

# Force cleanup (no prompt)
goenv cache clean all --force

# Refresh version list
goenv refresh
```

## VS Code Integration

```bash
# Complete setup (one command!)
goenv vscode setup

# Or step by step:
goenv vscode init    # Create config
goenv vscode sync    # Update with current version

# After changing Go version:
goenv use 1.25.2
goenv vscode sync    # Update VS Code settings
```

## Hooks System

```bash
# Initialize hooks
goenv hooks init

# Validate configuration
goenv hooks validate

# Test hooks (dry run)
goenv hooks test post_install

# List available hooks
goenv hooks list
```

**Common hook example:**
```yaml
# ~/.goenv/hooks.yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Installed Go {version}"
```

## Compliance & Inventory

```bash
# Generate inventory
goenv inventory go

# With checksums
goenv inventory go --checksums

# JSON output
goenv inventory go --json

# Generate SBOM
goenv sbom project --tool=cyclonedx-gomod
```

## CI/CD Patterns

```bash
# Non-interactive mode
goenv install 1.25.2 --yes

# Use --force for automation
goenv cache clean all --force
goenv uninstall 1.24.0 --force

# Offline mode (fastest, reproducible)
export GOENV_OFFLINE=1
goenv install 1.25.2

# JSON for parsing
goenv list --json | jq -r '.[] | select(.active) | .version'
```

## Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `GOENV_ROOT` | Install location | `$HOME/.goenv` |
| `GOENV_VERSION` | Force version | `1.25.2` |
| `GOENV_OFFLINE` | Offline mode | `1` |
| `GOENV_DEBUG` | Debug logging | `1` |
| `GOENV_DISABLE_GOPATH` | Disable isolation | `1` |
| `GOENV_ASSUME_YES` | 🆕 Auto-confirm prompts (CI) | `1` |

```bash
# Set in shell config (~/.bashrc, ~/.zshrc)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

## Version Precedence

goenv checks for version in this order:

1. `GOENV_VERSION` environment variable
2. `.go-version` file (current directory, then parents)
3. `go.mod` toolchain directive
4. `~/.goenv/version` (global version)
5. `system` (if no version set)

```bash
# Override everything
GOENV_VERSION=1.24.8 go version

# Per-project
echo "1.25.2" > .go-version

# Global default
goenv use 1.25.2 --global
```

## Common Flags

| Flag | Purpose | Works With |
|------|---------|------------|
| `--help, -h` | Show help | All commands |
| `--version, -v` | Show version | `goenv` |
| `--json` | JSON output | `list`, `current`, `inventory` |
| `--bare` | Plain text | `list`, `current` |
| `--force, -f` | Skip prompts | `cache clean`, `uninstall` |
| `--dry-run, -n` | Preview | `cache clean` |
| `--global` | Global scope | `use` |
| `--remote` | Remote versions | `list` |
| `--stable` | Stable only | `list --remote` |
| `--yes, -y` | Auto-confirm | `install` |

## Quick Troubleshooting

| Problem | Solution |
|---------|----------|
| `goenv: command not found` | 🆕 Run: `goenv setup` or add to PATH: `export PATH="$GOENV_ROOT/bin:$PATH"` |
| Not sure what's wrong | 🆕 Run: `goenv status` for quick check, `goenv doctor` for full diagnostic |
| Shell not initialized | 🆕 Run: `goenv setup` or `eval "$(goenv init -)"` |
| Issues detected by doctor | 🆕 Run: `goenv doctor --fix` for interactive repairs |
| Wrong Go version active | Check: `goenv current`, fix with `goenv use <version>` |
| Multiple goenv installations | 🆕 Run: `goenv doctor` to detect, `goenv doctor --fix` to resolve |
| Tool not found after install | Run: `goenv rehash` |
| Stale version list | Run: `goenv refresh` |
| Cache corruption | Run: `goenv cache clean all --force` then `goenv refresh` |
| Slow `list --remote` | Use: `GOENV_OFFLINE=1 goenv list --remote` |
| VS Code wrong version | Run: `goenv vscode sync` |
| Don't know which version to use | 🆕 Run: `goenv info <version>` or `goenv compare <v1> <v2>` |
| Version is EOL/unsupported | 🆕 Run: `goenv info $(goenv current --bare)` for upgrade recommendations |

## Platform-Specific Commands

### Linux / macOS

```bash
# Installation
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Shell config (~/.bashrc or ~/.zshrc)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Windows (PowerShell)

```powershell
# Installation
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex

# PowerShell profile ($PROFILE)
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
goenv init - powershell | Out-String | Invoke-Expression
```

## JSON Output Examples

```bash
# Get installed versions as JSON
goenv list --json

# Get active version info
goenv current --json

# Parse with jq
goenv list --json | jq -r '.[] | select(.active) | .version'

# Get inventory with checksums
goenv inventory go --json --checksums

# Remote versions
goenv list --remote --json
```

## Useful Aliases

Add to your shell config:

```bash
# Short aliases
alias g='goenv'
alias gi='goenv install'
alias gu='goenv use'
alias gc='goenv current'
alias gl='goenv list'

# Common tasks
alias goenv-latest='goenv install $(goenv list --remote --stable --bare | tail -1)'
alias goenv-doctor='goenv doctor && goenv cache status'
alias goenv-clean='goenv cache clean all --force'

# New commands (v3.0+)
alias goenv-setup='goenv setup --verify'
alias goenv-fix='goenv doctor --fix'
alias goenv-health='goenv status && echo && goenv doctor'
```

## Tips & Tricks

**Install and use in one line:**
```bash
goenv install 1.25.2 && goenv use 1.25.2
```

**Check if version is installed:**
```bash
goenv list --bare | grep -q "^1.25.2$" && echo "Installed" || echo "Not installed"
```

**Install only if missing:**
```bash
goenv list --bare | grep -q "^1.25.2$" || goenv install 1.25.2
```

**Get latest stable version:**
```bash
goenv list --remote --stable --bare | tail -1
```

**Active version without decoration:**
```bash
goenv current --bare
```

**Export environment for other tools:**
```bash
export GOVERSION=$(goenv current --bare)
export GOROOT=$(goenv prefix)
```

## Modern vs Legacy Commands

Use modern commands for new code:

| Task | Modern (Use This) | Legacy (Avoid) |
|------|-------------------|----------------|
| Set local version | `goenv use 1.25.2` | `goenv local 1.25.2` |
| Set global version | `goenv use 1.25.2 --global` | `goenv global 1.25.2` |
| Check version | `goenv current` | `goenv version` |
| List versions | `goenv list` | `goenv versions` |

Legacy commands still work, but modern commands are recommended.

## Getting Help

```bash
# General help
goenv --help

# Command-specific help
goenv install --help
goenv use --help
goenv cache --help

# Run diagnostics
goenv doctor

# Check version
goenv --version
```

## Documentation Links

**Quick Start:**
- [Installation Guide](./user-guide/INSTALL.md)
- [Modern Commands](./user-guide/MODERN_COMMANDS.md)
- [FAQ](./FAQ.md)

**Reference:**
- [Complete Command Reference](./reference/COMMANDS.md)
- [Environment Variables](./reference/ENVIRONMENT_VARIABLES.md)
- [Platform Support](./reference/PLATFORM_SUPPORT.md)

**Advanced:**
- [Hooks System](./reference/HOOKS_QUICKSTART.md)
- [Compliance Use Cases](./advanced/COMPLIANCE_USE_CASES.md)
- [JSON Output Guide](./reference/JSON_OUTPUT_GUIDE.md)
- [Cache Troubleshooting](./advanced/CACHE_TROUBLESHOOTING.md)

---

**Print-friendly version:** Remove this section and print the rest.

**Online version:** https://github.com/go-nv/goenv/blob/main/docs/QUICK_REFERENCE.md
