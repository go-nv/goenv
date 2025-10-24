# goenv Documentation

Welcome to the goenv documentation! This directory contains comprehensive documentation for using, configuring, and contributing to goenv.

## 📚 Table of Contents

### Getting Started

- **[Installation Guide](user-guide/INSTALL.md)** - Complete installation instructions for all platforms
- **[How It Works](user-guide/HOW_IT_WORKS.md)** - Understanding goenv's architecture and workflow
- **[VS Code Integration](user-guide/VSCODE_INTEGRATION.md)** - Setting up VS Code with goenv
- **[New Features](NEW_FEATURES.md)** - Summary of new features in Go implementation
- **[Migration Guide](MIGRATION_GUIDE.md)** - Migrating from bash to Go implementation

### Reference Documentation

- **[Commands Reference](reference/COMMANDS.md)** - Complete command-line interface documentation
- **[Environment Variables](reference/ENVIRONMENT_VARIABLES.md)** - All configuration options via environment variables

### Advanced Topics

- **[Advanced Configuration](advanced/ADVANCED_CONFIGURATION.md)** - Advanced setup and customization options
- **[Smart Caching](advanced/SMART_CACHING.md)** - Understanding goenv's intelligent caching system
- **[Embedded Versions](advanced/EMBEDDED_VERSIONS.md)** - How offline mode and embedded versions work
- **[GOPATH Integration](advanced/GOPATH_INTEGRATION.md)** - Managing GOPATH binaries per version
- **[Cross-Building](advanced/CROSS_BUILDING.md)** - Cross-compilation and architecture-specific builds
- **[What NOT to Sync](advanced/WHAT_NOT_TO_SYNC.md)** - Sharing goenv across machines and containers
- **[Hooks System](HOOKS.md)** - Extending goenv with custom hooks

### Troubleshooting & Diagnostics

- **[Environment Detection](../ENVIRONMENT_DETECTION.md)** - Container, WSL, and filesystem detection
- **[Environment Detection Quick Reference](ENVIRONMENT_DETECTION_QUICKREF.md)** - Quick reference for environment issues
- **[Platform-Specific Enhancements](../PLATFORM_SPECIFIC_ENHANCEMENTS.md)** - macOS, Windows, and Linux platform checks

### Contributing

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to goenv
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

**💡 Pro tips:**

- 🌐 **Offline/air-gapped?** Set `export GOENV_OFFLINE=1` to use embedded versions (no network required)
- 🚀 **CI/CD?** Use `GOENV_OFFLINE=1` for maximum speed and reproducibility
- 📶 **Slow connection?** Cache is automatically used when available (see [Smart Caching](advanced/SMART_CACHING.md))

**Note:** Tools installed with `go install` are isolated per Go version. Running `goenv rehash` (or `goenv use`) automatically creates shims for all installed tools.

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
├── README.md                    # This file - documentation index
├── user-guide/                  # User-facing documentation
│   ├── INSTALL.md              # Installation instructions
│   ├── HOW_IT_WORKS.md         # Architecture overview
│   └── VSCODE_INTEGRATION.md   # VS Code setup guide
├── reference/                   # Reference documentation
│   ├── COMMANDS.md             # Command reference
│   └── ENVIRONMENT_VARIABLES.md # Environment variable reference
├── advanced/                    # Advanced topics
│   ├── ADVANCED_CONFIGURATION.md # Advanced configuration
│   ├── SMART_CACHING.md        # Caching internals
│   ├── EMBEDDED_VERSIONS.md    # Offline mode details
│   └── GOPATH_INTEGRATION.md   # GOPATH management
├── CONTRIBUTING.md              # Contribution guidelines
├── CODE_OF_CONDUCT.md          # Community guidelines
├── CHANGELOG.md                 # Version history
└── RELEASE_PROCESS.md          # Release workflow
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
