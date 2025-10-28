# goenv Quick Reference

One-page cheat sheet for common goenv commands. Perfect for printing or quick lookup.

## Essential Commands

| Task | Command | Notes |
|------|---------|-------|
| **Install goenv** | `curl -sfL https://.../.../install.sh \| bash` | Binary installation (fastest) |
| **Install Go version** | `goenv install 1.25.2` | Downloads and installs |
| **Set project version** | `goenv use 1.25.2` | Creates `.go-version` file |
| **Set global version** | `goenv use 1.25.2 --global` | Sets default for all projects |
| **Check active version** | `goenv current` | Shows version and source |
| **List installed** | `goenv list` | Includes `*` for active |
| **List available** | `goenv list --remote` | All versions from golang.org |
| **Uninstall version** | `goenv uninstall 1.24.0` | Removes installation |
| **Update goenv** | `goenv update` | Self-update (git or binary) |
| **Diagnostics** | `goenv doctor` | Health check |

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

# Browse available versions
goenv list --remote

# Only stable releases
goenv list --remote --stable

# JSON output (automation)
goenv list --json
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
# Install common tools
goenv default-tools

# Install specific tool
goenv tools install gopls@latest

# List installed tools
goenv tools list

# Update tools
goenv tools update

# Sync tools between versions
goenv sync-tools
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
| `goenv: command not found` | Add to PATH: `export PATH="$GOENV_ROOT/bin:$PATH"` |
| Wrong Go version active | Check: `goenv current`, fix with `goenv use <version>` |
| Tool not found after install | Run: `goenv rehash` |
| Stale version list | Run: `goenv refresh` |
| Cache corruption | Run: `goenv cache clean all --force` then `goenv refresh` |
| Slow `list --remote` | Use: `GOENV_OFFLINE=1 goenv list --remote` |
| VS Code wrong version | Run: `goenv vscode sync` |

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
- [Modern Commands](./MODERN_COMMANDS.md)
- [FAQ](./FAQ.md)

**Reference:**
- [Complete Command Reference](./reference/COMMANDS.md)
- [Environment Variables](./reference/ENVIRONMENT_VARIABLES.md)
- [Platform Support](./PLATFORM_SUPPORT.md)

**Advanced:**
- [Hooks System](./HOOKS_QUICKSTART.md)
- [Compliance Use Cases](./COMPLIANCE_USE_CASES.md)
- [JSON Output Guide](./JSON_OUTPUT_GUIDE.md)
- [Cache Troubleshooting](./CACHE_TROUBLESHOOTING.md)

---

**Print-friendly version:** Remove this section and print the rest.

**Online version:** https://github.com/go-nv/goenv/blob/main/docs/QUICK_REFERENCE.md
