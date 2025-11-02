# Frequently Asked Questions (FAQ)

Common questions about goenv with quick answers. Use Ctrl/Cmd+F to search.

## Table of Contents

- [General](#general)
- [Installation](#installation)
- [Version Management](#version-management)
- [Tools](#tools)
- [VS Code Integration](#vs-code-integration)
- [Cache](#cache)
- [Performance](#performance)
- [Platform-Specific](#platform-specific)
- [Troubleshooting](#troubleshooting)
- [CI/CD](#cicd)
- [Hooks](#hooks)
- [Compliance](#compliance)

## General

### What is goenv?

goenv is a Go version manager that lets you install and switch between multiple Go versions on the same machine. It's now written in Go (previously bash) for better performance and cross-platform support.

### How does goenv differ from other Go version managers?

- **Pure Go implementation** - No bash scripts, works natively on Windows
- **Dynamic version fetching** - Always up-to-date without manual updates
- **Offline support** - Works without internet via embedded versions
- **Smart caching** - Intelligent cache with 3-tier freshness
- **VS Code integration** - First-class VS Code support
- **Hooks system** - Declarative automation without scripts

### Is goenv compatible with the bash version?

Yes! goenv maintains backward compatibility:
- Same commands work
- Same file locations (`~/.goenv`)
- Same configuration files (`.go-version`)
- Legacy commands still supported

[Migration Guide](./user-guide/MIGRATION_GUIDE.md)

### What platforms does goenv support?

- **Linux** - All major distributions (Ubuntu, Debian, Fedora, RHEL, Arch, Alpine)
- **macOS** - Intel and Apple Silicon (M1/M2/M3)
- **Windows** - Native PowerShell and CMD support
- **FreeBSD** - Full support
- **Architectures** - AMD64, ARM64, ARMv6/7, 386, PPC64LE, S390X

[Platform Support Matrix](./reference/PLATFORM_SUPPORT.md)

## Installation

### How do I install goenv?

**Binary installation (recommended):**
```bash
# Linux/macOS
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Windows (PowerShell)
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex
```

After installation, run the setup wizard for automatic configuration:

```bash
goenv setup
```

[Complete Installation Guide](./user-guide/INSTALL.md)

### What's the easiest way to get started with goenv?

After installing goenv, use the interactive setup and beginner guide:

```bash
# Automatic shell configuration
goenv setup

# Interactive beginner guide
goenv get-started

# Quick health check
goenv status
```

These commands guide you through first-time setup and teach you common tasks.

### Do I need Go installed to use goenv?

No! goenv provides pre-built binaries that don't require Go. This is perfect for bootstrapping.

### How do I update goenv?

```bash
goenv update
```

This works for both git-based and binary installations.

### Where does goenv install files?

- **goenv itself:** `~/.goenv/` (or `$GOENV_ROOT`)
- **Go versions:** `~/.goenv/versions/<version>/`
- **Tools:** `~/go/<version>/bin/` (version-isolated)
- **Cache:** `~/.goenv/cache/`
- **Shims:** `~/.goenv/shims/`

### How do I uninstall goenv?

```bash
# Remove goenv directory
rm -rf ~/.goenv

# Remove from shell config
# Edit ~/.bashrc, ~/.zshrc, or $PROFILE
# Remove goenv-related lines

# Restart shell
```

### I already have Go installed globally. Will goenv interfere?

**No, they coexist peacefully!** This is a designed feature, not a bug.

**What happens:**

1. **goenv comes first in PATH** (if configured correctly):
   ```bash
   # Correct order in ~/.bashrc
   export GOENV_ROOT="$HOME/.goenv"
   export PATH="$GOENV_ROOT/bin:$GOENV_ROOT/shims:$PATH"  # goenv BEFORE system
   ```

2. **System Go becomes "system" version**:
   ```bash
   goenv list
   # Output:
   #   1.25.2
   #   1.24.8
   #   system (set by /usr/local/go)

   # You can explicitly use it
   goenv use system
   ```

3. **When goenv version is active**, goenv's paths take precedence:
   - `GOROOT` → `~/.goenv/versions/1.25.2/`
   - `GOPATH` → `~/go/1.25.2/`
   - Tools → goenv-managed

4. **When `system` is active**, system Go is used:
   - `GOROOT` → `/usr/local/go/` (or system location)
   - `GOPATH` → System default or `~/go/`
   - Tools → System-installed tools

**Verify your setup:**
```bash
# Check PATH order
echo $PATH
# goenv directories should appear FIRST

# Check active version
goenv current

# Check what Go binary is being used
which go
# Should show: /Users/you/.goenv/shims/go (if goenv version active)

# Check GOROOT
go env GOROOT
# Should show goenv path if goenv version active
```

**Common issues:**

❌ **Problem: System Go is being used instead of goenv**

**Cause:** goenv shims not first in PATH

**Fix:**
```bash
# Check PATH
echo $PATH | tr ':' '\n' | head -5

# Should show:
# /Users/you/.goenv/shims
# /Users/you/.goenv/bin
# /usr/local/bin  ← system go is here
# ...

# If not, fix your shell config:
export PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:$PATH"
```

❌ **Problem: Tools installed with system Go don't work with goenv**

**Cause:** Tool isolation by design

**Expected behavior:** Tools are version-specific:
```bash
# System Go tools
/usr/local/go → tools in system GOPATH

# goenv Go tools
~/.goenv/versions/1.25.2/ → tools in ~/go/1.25.2/
```

**Solution:** Reinstall tools with goenv:
```bash
goenv use 1.25.2
go install golang.org/x/tools/gopls@latest
# Installs to ~/go/1.25.2/bin/gopls
```

**Best practices:**

✅ **Keep both** - Use goenv for development, system Go as fallback:
```bash
# Development projects
cd ~/my-project
goenv use 1.25.2

# Quick system scripts
goenv use system
# Or just use system Go directly: /usr/local/go/bin/go
```

✅ **Explicit system version** when needed:
```bash
# Force system Go for specific project
echo "system" > .go-version
```

✅ **Clean PATH** - Ensure correct order:
```bash
# .bashrc / .zshrc
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**When conflicts DO occur:**

Usually due to environment variable pollution:

```bash
# Check for conflicts
env | grep GO

# Common culprits:
# GOROOT=/usr/local/go  ← Set by something else
# GOPATH=/some/path      ← Set by something else

# Fix: Remove from shell config, let goenv manage them
```

**See also:**
- [System Go Coexistence Guide](./user-guide/SYSTEM_GO_COEXISTENCE.md) - Complete guide to using both
- [How It Works - Version Detection](./user-guide/HOW_IT_WORKS.md#version-detection)
- [GOPATH Integration](./advanced/GOPATH_INTEGRATION.md)

## Version Management

### How do I install a specific Go version?

```bash
goenv install 1.25.2
```

### How do I see what versions are available?

```bash
# All versions
goenv list --remote

# Stable only
goenv list --remote --stable

# JSON output
goenv list --remote --json
```

### How do I set the Go version for my project?

```bash
cd ~/my-project
goenv use 1.25.2
```

This creates a `.go-version` file.

### How do I set a global default Go version?

```bash
goenv use 1.25.2 --global
```

### What's the difference between `goenv use` and `goenv local`?

`goenv use` is the modern command. `goenv local` is legacy (bash goenv compatibility).

Use `goenv use` for new code.

[Modern Commands Guide](./user-guide/MODERN_COMMANDS.md)

### How does goenv choose which Go version to use?

Version precedence (highest to lowest):

1. `GOENV_VERSION` environment variable
2. `.go-version` file (current dir, then parents)
3. `go.mod` toolchain directive (Go 1.21+)
4. `~/.goenv/version` (global default)
5. `system` (OS-installed Go)

### Can I use goenv with go.mod toolchain directives?

Yes! goenv automatically detects and respects `toolchain` directives in `go.mod`:

```go
// go.mod
go 1.21

toolchain go1.25.2  // goenv will use this
```

[Smart Version Detection](./user-guide/HOW_IT_WORKS.md#version-detection)

### How do I check what Go version is active?

```bash
goenv current
# Output: 1.25.2 (set by /path/to/project/.go-version)
```

### How do I get detailed information about a Go version?

```bash
goenv info 1.25.2
# Shows: install status, release date, EOL status, size, recommendations
```

See also: `goenv info --json` for machine-readable output.

### How do I compare two Go versions?

```bash
goenv compare 1.21.5 1.23.2
# Shows: side-by-side comparison of release dates, support status, size diff
```

Useful for deciding whether to upgrade.

### How do I check if my Go version is EOL (end-of-life)?

```bash
goenv info $(goenv current --bare)
# Shows EOL status and upgrade recommendations
```

### How do I quickly check my goenv installation health?

```bash
# Quick overview
goenv status

# Comprehensive diagnostics
goenv doctor

# Interactive repair mode
goenv doctor --fix
```

### How do I discover goenv commands by category?

```bash
goenv explore
# Interactive browser showing commands grouped by:
# - Getting Started
# - Version Management
# - Tools & Diagnostics
# - etc.
```

### How do I list installed versions?

```bash
goenv list
```

### How do I uninstall a Go version?

```bash
goenv uninstall 1.24.0

# Non-interactive mode
goenv uninstall 1.24.0 --force
```

### Can I have multiple Go versions installed?

Yes! That's the whole point of goenv. Install as many as you need:

```bash
goenv install 1.23.2
goenv install 1.24.8
goenv install 1.25.2
```

Switch between them per-project with `.go-version` files.

## Tools

### How do tools work with goenv?

Tools installed with `go install` are automatically isolated per Go version:

```bash
goenv use 1.25.2
go install golang.org/x/tools/gopls@latest
# Installs to ~/go/1.25.2/bin/gopls

goenv use 1.24.0
# gopls is not available (or uses 1.24.0 version if installed)
```

[GOPATH Integration](./advanced/GOPATH_INTEGRATION.md)

### How do I install common tools?

```bash
goenv default-tools
```

This installs gopls, staticcheck, golangci-lint, and other common tools.

### How do I install tools across all Go versions?

Use the `--all` flag to install across all versions at once:

```bash
# Install gopls for all Go versions
goenv tools install gopls@latest --all

# Install multiple tools everywhere
goenv tools install gopls@latest staticcheck@latest --all
```

### How do I uninstall tools across all Go versions?

Use the `--all` flag with uninstall:

```bash
# Uninstall from all versions
goenv tools uninstall gopls --all

# Uninstall multiple tools from all versions
goenv tools uninstall gopls staticcheck --all

# Preview what would be removed (dry run)
goenv tools uninstall gopls --all --dry-run
```

### How do I check tool consistency across versions?

Use `goenv tools status` to see which tools are installed where:

```bash
goenv tools status
```

This shows:
- **Consistent tools**: Installed in all versions
- **Partial tools**: Installed in some versions
- **Version-specific tools**: Only in one version

### How do I check which tools need updating?

```bash
# Check for outdated tools across all versions
goenv tools outdated

# Update tools for current version
goenv tools update

# Update tools across all versions
goenv tools update --all
```

### How do I sync tools between Go versions?

```bash
# Sync all tools from active version to another version
goenv tools sync 1.25.2 1.24.0

# Auto-detect and sync
goenv tools sync
```

### Why can't I find my tool after installing it?

Run `goenv rehash` to regenerate shims:

```bash
go install example.com/tool@latest
goenv rehash
tool --version  # Now works
```

**Note:** `goenv use` automatically runs `goenv rehash`, but `go install` doesn't.

### Can goenv automatically update my tools?

Yes! Enable auto-update in `~/.goenv/default-tools.yaml`:

```yaml
auto_update_enabled: true
auto_update_strategy: "on_use"  # Check when switching versions
auto_update_interval: "24h"     # Check at most once per day
```

When you run `goenv use 1.21.5`, you'll see:
```
💡 2 tool update(s) available for Go 1.21.5
   Run 'goenv tools update' to update
```

[Complete auto-update guide](./user-guide/AUTO_UPDATE_TOOLS.md)

### Will auto-update slow down `goenv use`?

No. Checks are throttled (default: 24h) and cached. Most of the time no network calls are made.

### Can I pin tools to specific versions per Go version?

Yes! Use version overrides in `~/.goenv/default-tools.yaml`:

```yaml
tools:
  - name: gopls
    version_overrides:
      "1.18": "v0.11.0"  # Go 1.18 gets gopls v0.11.0
      "1.23+": "@latest" # Go 1.23+ gets latest
```

## VS Code Integration

### How do I set up VS Code with goenv?

One command:

```bash
goenv vscode setup
```

[VS Code Integration Guide](./user-guide/VSCODE_INTEGRATION.md)

### Do I need to restart VS Code after changing Go versions?

Depends on your setup mode:

**Absolute paths mode (default):**
```bash
goenv use 1.25.2
goenv vscode sync
# Then: Cmd+Shift+P → "Developer: Reload Window"
```

**Environment variables mode:**
- Quit VS Code completely
- Reopen from terminal: `code .`

### VS Code doesn't see my Go installation. Help?

```bash
# Run setup
goenv vscode setup

# Verify
goenv doctor

# Check VS Code settings
cat .vscode/settings.json
```

### Can I use goenv with VS Code's Go extension?

Yes! That's exactly what `goenv vscode` commands configure. The official Go extension works perfectly with goenv.

## Cache

### What does goenv cache?

1. **Version list** - Available Go versions from golang.org
2. **Build cache** - Compiled packages (GOCACHE)
3. **Module cache** - Downloaded modules (GOMODCACHE)

[Cache Types](./CACHE_TROUBLESHOOTING.md#cache-types)

### How do I clear the cache?

```bash
# Clear all caches
goenv cache clean all --force

# Clear build cache only
goenv cache clean build

# Clear old caches (30+ days)
goenv cache clean all --older-than 30d
```

### How big is the cache?

```bash
goenv cache status
```

### Why is my version list stale?

Cache is older than 7 days. Refresh it:

```bash
goenv refresh
```

### What's offline mode?

Offline mode uses embedded versions (no network calls):

```bash
export GOENV_OFFLINE=1
goenv list --remote  # Uses embedded data, ~8ms
```

Perfect for CI/CD, air-gapped systems, or fast local development.

[Smart Caching Guide](./advanced/SMART_CACHING.md)

## Performance

### Why is `goenv list --remote` slow?

First time fetches from network. Use offline mode for speed:

```bash
GOENV_OFFLINE=1 goenv list --remote
# 8ms vs 500ms
```

Cache is used for subsequent calls (within 6 hours).

### How can I make installs faster?

1. **Use offline mode in CI:**
   ```bash
   export GOENV_OFFLINE=1
   goenv install 1.25.2
   ```

2. **Cache goenv versions directory:**
   ```yaml
   # GitHub Actions
   - uses: actions/cache@v4
     with:
       path: ~/.goenv/versions
       key: goenv-${{ runner.os }}-${{ hashFiles('.go-version') }}
   ```

3. **Use binary installation** (not git-based)

[Performance Optimization](./CACHE_TROUBLESHOOTING.md#performance-optimization)

### Does goenv slow down my builds?

No! goenv only runs during version selection. Once the Go version is chosen, it's native Go performance. The shim overhead is negligible (< 1ms).

## Platform-Specific

### macOS: Does goenv work on Apple Silicon (M1/M2/M3)?

Yes! Native ARM64 support. goenv automatically downloads ARM64 binaries.

```bash
# Check if you're running natively
goenv doctor
# Output: Platform: darwin/arm64
```

[Apple Silicon Support](./PLATFORM_SUPPORT.md#macos)

### macOS: Do I need Rosetta 2?

No, but goenv can use it if needed. AMD64 binaries run via Rosetta 2 if ARM64 unavailable (rare for modern Go versions).

### Windows: Does goenv work on Windows?

Yes! Full native Windows support:
- PowerShell (recommended)
- CMD (basic support)
- Git Bash (Unix-like experience)

[Windows Support](./PLATFORM_SUPPORT.md#windows)

### Windows: Do I need WSL?

No! goenv works natively on Windows. WSL is optional but supported.

### WSL: Does goenv work in WSL?

Yes! Full support for WSL1 and WSL2.

```bash
# Check detection
goenv doctor
# Output: WSL: WSL2 detected
```

[WSL Support](./PLATFORM_SUPPORT.md#wsl-specific)

### Linux: Which distributions are supported?

All major distributions:
- Ubuntu/Debian
- Fedora/RHEL/CentOS
- Arch/Manjaro
- Alpine (musl)
- Any Linux with glibc or musl

### Can I use goenv in Docker?

Yes!

```dockerfile
FROM golang:1.25-alpine

RUN apk add --no-cache curl bash
RUN curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

ENV GOENV_ROOT="/root/.goenv"
ENV PATH="$GOENV_ROOT/bin:$PATH"
ENV GOENV_OFFLINE=1

RUN goenv install 1.25.2 && goenv use 1.25.2 --global
```

[Docker Configuration](./PLATFORM_SUPPORT.md#docker-configuration)

## Troubleshooting

### `goenv: command not found`

Add goenv to your PATH:

```bash
# Linux/macOS (~/.bashrc or ~/.zshrc)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"

# Windows (PowerShell $PROFILE)
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
goenv init - powershell | Out-String | Invoke-Expression
```

Restart your shell after adding.

### Wrong Go version is active

Check version precedence:

```bash
goenv current
# Shows where version is set from

# Check environment variable
echo $GOENV_VERSION

# Check .go-version files
cat .go-version
cat ../.go-version  # Check parent dirs

# Set explicitly
goenv use 1.25.2
```

### Tool not found after `go install`

Regenerate shims:

```bash
goenv rehash
```

Or use `goenv use` which automatically rehashes.

### `permission denied` errors

Check goenv directory permissions:

```bash
ls -ld ~/.goenv
chmod -R u+w ~/.goenv
```

### Stale cache warnings

Refresh the version cache:

```bash
goenv refresh
```

### Can't connect to golang.org

1. **Check network:** `curl -I https://go.dev`
2. **Use offline mode:** `GOENV_OFFLINE=1 goenv list --remote`
3. **Check proxy:** `echo $HTTP_PROXY $HTTPS_PROXY`
4. **Check firewall:** Ensure HTTPS (443) is allowed

### Cache corruption errors

Clear and refresh:

```bash
goenv cache clean all --force
goenv refresh
```

[Complete Troubleshooting Guide](./advanced/CACHE_TROUBLESHOOTING.md)

## CI/CD

### How do I use goenv in CI/CD?

```yaml
# GitHub Actions
steps:
  - name: Install goenv
    run: |
      curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
      echo "$HOME/.goenv/bin" >> $GITHUB_PATH

  - name: Install Go
    run: |
      export GOENV_OFFLINE=1
      goenv install $(cat .go-version)
      goenv use $(cat .go-version)
```

[CI/CD Guide](./advanced/CI_CD_GUIDE.md)

### Should I cache goenv in CI?

Yes! Cache the versions directory:

```yaml
# GitHub Actions
- uses: actions/cache@v4
  with:
    path: ~/.goenv/versions
    key: goenv-${{ runner.os }}-${{ hashFiles('.go-version') }}
```

Don't cache `~/.goenv/cache/` - use offline mode instead.

### Should I use offline mode in CI?

Yes! Faster and more reproducible:

```bash
export GOENV_OFFLINE=1
goenv install 1.25.2
```

### How do I get JSON output for parsing?

```bash
goenv list --json | jq -r '.[] | select(.active) | .version'
goenv current --json
goenv inventory go --json
```

[JSON Output Guide](./reference/JSON_OUTPUT_GUIDE.md)

## Hooks

### What are hooks?

Declarative automation that runs during goenv lifecycle events:

```yaml
# ~/.goenv/hooks.yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Installed Go {version}"
```

[Hooks Quick Start](./reference/HOOKS_QUICKSTART.md)

### How do I enable hooks?

```bash
goenv hooks init           # Generate config
# Edit ~/.goenv/hooks.yaml
goenv hooks validate       # Check config
```

Set `enabled: true` and `acknowledged_risks: true` in `hooks.yaml`.

### Are hooks safe?

Yes, with secure defaults:
- HTTPS enforced (HTTP blocked)
- Internal IPs blocked (SSRF protection)
- Timeouts on all actions
- No shell injection (use args array)

[Hooks Security Model](./HOOKS.md#security-model)

### Can I send webhooks on install?

Yes!

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Go {version} installed"}'
```

### Can I run commands in hooks?

Yes! Always use args array for security:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go"
        args: ["version"]
```

**Never use shell strings for security reasons.**

[run_command Security](./HOOKS.md#run_command)

## Compliance

### Can I generate an SBOM?

Yes!

```bash
# Install SBOM tool
goenv tools install cyclonedx-gomod@v1.6.0

# Generate SBOM
goenv sbom project --tool=cyclonedx-gomod --output=sbom.json
```

[SBOM Documentation](./reference/COMMANDS.md#goenv-sbom)

### How do I track Go installations for compliance?

```bash
# Generate inventory
goenv inventory go --json --checksums > inventory.json

# Automate with hooks
# See Compliance Use Cases guide
```

[Compliance Use Cases](./advanced/COMPLIANCE_USE_CASES.md)

### Is goenv SOC 2 compliant?

goenv provides tools for SOC 2 compliance:
- Software inventory tracking
- Change management logging
- Audit trail generation
- Checksum verification

[SOC 2 Examples](./COMPLIANCE_USE_CASES.md#soc-2-compliance)

### Can I track version changes?

Yes, with hooks:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: /var/audit/go-changes.log
        message: "[{timestamp}] INSTALL {version} - ${USER}@${HOSTNAME}"

  post_uninstall:
    - action: log_to_file
      params:
        path: /var/audit/go-changes.log
        message: "[{timestamp}] UNINSTALL {version} - ${USER}@${HOSTNAME}"
```

## Still Have Questions?

- **Search documentation:** Use Ctrl/Cmd+F on [docs/README.md](./README.md)
- **Quick reference:** [QUICK_REFERENCE.md](./QUICK_REFERENCE.md)
- **Run diagnostics:** `goenv doctor`
- **Open an issue:** [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Start a discussion:** [GitHub Discussions](https://github.com/go-nv/goenv/discussions)

## Related Documentation

- [Quick Reference](./QUICK_REFERENCE.md) - One-page cheat sheet
- [Modern Commands Guide](./user-guide/MODERN_COMMANDS.md) - Recommended commands
- [Platform Support](./reference/PLATFORM_SUPPORT.md) - Platform compatibility
- [Troubleshooting](./advanced/CACHE_TROUBLESHOOTING.md) - Detailed troubleshooting
- [Complete Command Reference](./reference/COMMANDS.md) - All commands
