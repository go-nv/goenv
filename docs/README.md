# goenv Documentation

Welcome to the goenv documentation! This directory contains comprehensive documentation for using, configuring, and contributing to goenv.

## 📚 Table of Contents

### Getting Started

- **[Installation Guide](user-guide/INSTALL.md)** - Complete installation instructions for all platforms
- **[Quick Reference](QUICK_REFERENCE.md)** - One-page cheat sheet ⭐ **NEW**
- **[FAQ](FAQ.md)** - Frequently asked questions ⭐ **NEW**
- **[How It Works](user-guide/HOW_IT_WORKS.md)** - Understanding goenv's architecture and workflow
- **[VS Code Integration](user-guide/VSCODE_INTEGRATION.md)** - Setting up VS Code with goenv
- **[What's New in Documentation](internal/WHATS_NEW_DOCUMENTATION.md)** - Recent documentation improvements ⭐ **NEW**
- **[New Features](internal/NEW_FEATURES.md)** - Summary of new features in Go implementation
- **[Migration Guide](user-guide/MIGRATION_GUIDE.md)** - Migrating from bash to Go implementation

### Reference Documentation

- **[Commands Reference](reference/COMMANDS.md)** - Complete command-line interface documentation
- **[Environment Variables](reference/ENVIRONMENT_VARIABLES.md)** - All configuration options via environment variables
- **[Platform Support Matrix](reference/PLATFORM_SUPPORT.md)** - OS and architecture compatibility guide
- **[Modern vs Legacy Commands](user-guide/MODERN_COMMANDS.md)** - Command modernization guide
- **[JSON Output Guide](reference/JSON_OUTPUT_GUIDE.md)** - JSON output for automation and CI/CD

### Advanced Topics

- **[Advanced Configuration](advanced/ADVANCED_CONFIGURATION.md)** - Advanced setup and customization options
- **[Smart Caching](advanced/SMART_CACHING.md)** - Understanding goenv's intelligent caching system
- **[Embedded Versions](advanced/EMBEDDED_VERSIONS.md)** - How offline mode and embedded versions work
- **[GOPATH Integration](advanced/GOPATH_INTEGRATION.md)** - Managing GOPATH binaries per version
- **[Cross-Building](advanced/CROSS_BUILDING.md)** - Cross-compilation and architecture-specific builds
- **[What NOT to Sync](advanced/WHAT_NOT_TO_SYNC.md)** - Sharing goenv across machines and containers
- **[Hooks System Quick Start](reference/HOOKS_QUICKSTART.md)** - 5-minute hooks setup guide
- **[Hooks System (Full)](reference/HOOKS.md)** - Complete hooks documentation
- **[Compliance Use Cases](advanced/COMPLIANCE_USE_CASES.md)** - SOC 2, ISO 27001, SBOM generation

### Troubleshooting & Diagnostics

- **[Cache Troubleshooting](advanced/CACHE_TROUBLESHOOTING.md)** - Cache issues, migration, and optimization
- **[System Go Coexistence](user-guide/SYSTEM_GO_COEXISTENCE.md)** - Using goenv with system-installed Go ⭐ **NEW**
- **[Environment Detection Quick Reference](reference/ENVIRONMENT_DETECTION_QUICKREF.md)** - Quick reference for environment detection issues
- **[Platform Support Matrix](reference/PLATFORM_SUPPORT.md)** - Platform compatibility and OS-specific features

### Contributing

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to goenv (includes documentation guidelines)
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community guidelines
- **[Release Process](RELEASE_PROCESS.md)** - Release workflow for maintainers

### Project Information

- **[Changelog](CHANGELOG.md)** - Version history and release notes

## 🚀 Quick Start

```bash
# Install goenv
git clone https://github.com/go-nv/goenv.git ~/.goenv

# Add to your shell profile
echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bashrc
echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bashrc
echo 'eval "$(goenv init -)"' >> ~/.bashrc

# Restart your shell or source the profile
source ~/.bashrc

# Install a Go version
goenv install 1.25.2

# Set it as your global version
goenv use 1.25.2 --global

# Verify
go version

# Install tools (isolated per Go version)
go install golang.org/x/tools/gopls@latest

# Tools are automatically available via shims
gopls version

# 💡 Version shorthand: goenv 1.25.2 is a shortcut for goenv use 1.25.2
goenv 1.25.2  # Same as: goenv use 1.25.2
```

**💡 How `go install` works with goenv:**

Tools installed via `go install` are automatically isolated per Go version:

```bash
# Example: Installing gopls with Go 1.25.2
goenv use 1.25.2
go install golang.org/x/tools/gopls@latest
# → Installs to: $HOME/go/1.25.2/bin/gopls

goenv rehash  # Creates shim at ~/.goenv/shims/gopls

# Now gopls is available and uses Go 1.25.2
gopls version

# Switch to a different Go version
goenv use 1.24.0
# → gopls shim now points to $HOME/go/1.24.0/bin/gopls (if installed)
# → Or shows "command not found" if not installed for this version
```

**Key points:**

- 🔒 **Isolation:** Each Go version has its own `$HOME/go/{version}/` directory
- 🔄 **Auto-switching:** Tools switch automatically when you change Go versions
- 🛡️ **No conflicts:** Different tool versions for different Go versions coexist
- 🔧 **Shims:** `goenv rehash` (or `goenv use`) creates shims that route to the active version's tools

See [GOPATH Integration](advanced/GOPATH_INTEGRATION.md) for configuration options and advanced usage.

---

**Note:** Tools installed with `go install` are isolated per Go version. Running `goenv rehash` (or `goenv use`) automatically creates shims for all installed tools.

### 🌐 Network Reliability & Offline Mode

goenv works reliably in all network conditions:

> **💡 Offline/Air-Gapped Environments**
>
> ```bash
> export GOENV_OFFLINE=1
> ```
>
> This disables all network calls and uses embedded versions (331 versions built-in at last update). Perfect for:
>
> - 🔒 **Air-gapped systems** - No internet access required
> - 🚀 **CI/CD pipelines** - Maximum speed (~8ms vs ~500ms) and guaranteed reproducibility
> - 📶 **Bandwidth constraints** - Mobile hotspots, metered connections
> - 🔐 **Security requirements** - No outbound network calls
>
> All version listing and installation commands work offline with embedded data.

**Other network features:**

- ✅ **Automatic caching** - Reduces API calls by 85% for active users ([Smart Caching](advanced/SMART_CACHING.md))
- ✅ **Graceful fallback** - Network failures use cached data instead of failing
- ✅ **30-second timeout** - Prevents hanging on slow connections
- ✅ **ETag support** - 99.97% bandwidth savings when cache is current (2MB → 500 bytes)

**Quick network troubleshooting:**

```bash
# Test connectivity and cache status
goenv doctor

# Force refresh after network issues
goenv refresh

# Debug network/cache behavior
GOENV_DEBUG=1 goenv install --list
```

See [Network Reliability Defaults](advanced/SMART_CACHING.md#network-reliability-defaults) for complete details on timeouts, fallback behavior, and environment variables.

## 🎯 Modern Unified Commands

goenv provides intuitive, consistent commands for common operations:

| Command                   | Purpose                           | Replaces                |
| ------------------------- | --------------------------------- | ----------------------- |
| **`goenv use <version>`** | Set Go version (local or global)  | `local`, `global`       |
| **`goenv current`**       | Show active version and source    | `version`               |
| **`goenv list`**          | List installed/available versions | `versions`, `installed` |

These modern commands provide a cleaner interface while maintaining backward compatibility with legacy commands.

**Examples:**

```bash
# Set version for current project
goenv use 1.25.2

# Set global default version
goenv use 1.24.8 --global

# Show what's active
goenv current

# List installed versions
goenv list

# List available versions from remote
goenv list --remote --stable

# JSON output for automation
goenv list --json
```

See [Commands Reference](reference/COMMANDS.md) for complete command documentation.

## 📖 Documentation Structure

```
docs/
├── README.md                           # This file - documentation index
├── QUICK_REFERENCE.md                  # One-page cheat sheet (high-traffic)
├── FAQ.md                              # Frequently asked questions (high-traffic)
├── CHANGELOG.md                        # Version history
├── CONTRIBUTING.md                     # Contribution guidelines
├── CODE_OF_CONDUCT.md                  # Community guidelines
├── RELEASE_PROCESS.md                  # Release workflow
│
├── user-guide/                         # User-facing guides
│   ├── INSTALL.md                      # Installation instructions
│   ├── HOW_IT_WORKS.md                 # Architecture overview
│   ├── VSCODE_INTEGRATION.md           # VS Code setup guide
│   ├── MIGRATION_GUIDE.md              # Migrating from bash to Go
│   ├── MODERN_COMMANDS.md              # Command modernization guide
│   └── SYSTEM_GO_COEXISTENCE.md        # Using goenv with system Go
│
├── reference/                          # Complete references
│   ├── COMMANDS.md                     # Command reference
│   ├── ENVIRONMENT_VARIABLES.md        # Environment variable reference
│   ├── PLATFORM_SUPPORT.md             # Platform compatibility matrix
│   ├── ENVIRONMENT_DETECTION_QUICKREF.md # Environment detection guide
│   ├── HOOKS.md                        # Complete hooks documentation
│   ├── HOOKS_QUICKSTART.md             # 5-minute hooks setup
│   └── JSON_OUTPUT_GUIDE.md            # JSON output for automation
│
├── advanced/                           # Advanced topics & integrations
│   ├── ADVANCED_CONFIGURATION.md       # Advanced configuration
│   ├── SMART_CACHING.md                # Caching internals
│   ├── EMBEDDED_VERSIONS.md            # Offline mode details
│   ├── GOPATH_INTEGRATION.md           # GOPATH management
│   ├── CROSS_BUILDING.md               # Cross-compilation
│   ├── WHAT_NOT_TO_SYNC.md             # Sharing goenv configs
│   ├── CACHE_TROUBLESHOOTING.md        # Cache issues & optimization
│   ├── COMPLIANCE_USE_CASES.md         # SOC 2, ISO 27001, SBOM
│   └── CI_CD_GUIDE.md                  # CI/CD integration guide
│
└── internal/                           # Internal docs (historical/development)
    ├── NEW_FEATURES.md                 # Feature tracking
    └── WHATS_NEW_DOCUMENTATION.md      # Documentation updates
```

## 🔍 Key Features

### Multi-Version Management

Install and manage multiple Go versions simultaneously:

```bash
goenv install 1.25.2
goenv install 1.24.8
goenv list  # Show installed versions
```

### Per-Project Versions

Set different Go versions for different projects:

```bash
cd my-project
goenv use 1.24.8  # Creates .go-version file
```

### Smart Caching

Intelligent version caching with three-tier freshness checking:

- Fresh cache (< 6 hours): Instant response
- Recent cache (6h-7d): Quick freshness check
- Stale cache (> 7 days): Full refresh

### Offline Mode & Network Reliability

Work completely offline or optimize for slow/metered connections using embedded versions:

```bash
# Enable offline mode (no network calls, uses embedded versions)
export GOENV_OFFLINE=1
goenv install --list  # Uses embedded data - 330+ versions, no network calls

# Force cache refresh when you DO have network
goenv refresh

# Check doctor status (includes connectivity check)
goenv doctor
```

**When to use offline mode:**

- 🌐 **Air-gapped environments** - No internet access
- 🚀 **CI/CD pipelines** - Maximum speed and reproducibility
- 📶 **Metered connections** - Mobile hotspots, bandwidth limits
- 🔒 **Security requirements** - No outbound network calls allowed
- ⚡ **Performance critical** - 5x faster than online mode (8ms vs 42ms)

**Network reliability features:**

- **Smart caching** - Three-tier freshness checking (fresh/recent/stale)
- **ETag support** - Efficient "not modified" checks (saves bandwidth)
- **Embedded versions** - 330+ Go releases built into binary
- **Timeout protection** - HTTP requests have reasonable timeouts
- **Graceful fallback** - Uses cache if network unavailable

See [Smart Caching](advanced/SMART_CACHING.md) for details on caching strategy and network options.

### Cross-Platform

Full support for:

- macOS (Intel & Apple Silicon)
- Linux (multiple architectures)
- Windows (native PowerShell & CMD support)

## 💡 Common Use Cases

### Development Environment

```bash
# Install latest stable Go
goenv install --latest

# Use it globally
goenv use $(goenv install --latest) --global

# Or per-project
cd my-go-project
goenv use 1.25.2
```

### CI/CD Pipelines

```bash
# Fast, reproducible builds
export GOENV_OFFLINE=1  # No network dependencies
goenv install 1.25.2
goenv use 1.25.2 --global
go build
```

### Air-Gapped Environments

```bash
# Works without internet
export GOENV_OFFLINE=1
goenv install --list     # Shows 330+ embedded versions
goenv install 1.24.8     # Installs from embedded data
```

### Multi-Machine & Container Environments

When using goenv across multiple machines, containers, or in dotfiles repositories, it's important to know what to sync and what to keep local:

**✅ Safe to sync (commit to git, share in containers):**

- `.go-version` files (per-project version markers)
- `goenv.yaml` hooks configuration
- Shell initialization scripts (`.bashrc`, `.zshrc` snippets)

**❌ Do NOT sync (keep local to each machine):**

- `$GOENV_ROOT/versions/` - Installed Go binaries (platform-specific)
- `$GOENV_ROOT/cache/` - Version lists and download caches
- `$GOENV_ROOT/shims/` - Generated executable wrappers
- Architecture-specific build caches

**Best practices:**

- Use `.go-version` files for consistent versions across teams
- Let each machine/container install its own Go binaries
- Share hooks configuration (`goenv.yaml`) for team workflows
- Use `goenv install --yes` in CI/CD for automatic setup

See **[What NOT to Sync](advanced/WHAT_NOT_TO_SYNC.md)** for detailed guidance on sharing goenv configurations across machines and containers.

## 🆘 Getting Help

- **Issues**: [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Discussions**: [GitHub Discussions](https://github.com/go-nv/goenv/discussions)
- **Commands**: Run `goenv help` for command-specific help

## 📝 Additional Resources

- [Main README](../README.md) - Project overview and quick start
- [Scripts Documentation](../scripts/README.md) - Development scripts reference

## 🎯 Next Steps

1. **New Users**: Start with the [Installation Guide](user-guide/INSTALL.md)
2. **Daily Use**: Bookmark the [Commands Reference](reference/COMMANDS.md)
3. **Advanced Setup**: Check out [Advanced Configuration](advanced/ADVANCED_CONFIGURATION.md)
4. **Contributors**: Read the [Contributing Guide](CONTRIBUTING.md)
