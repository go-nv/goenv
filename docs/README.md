# goenv Documentation

Welcome to the goenv documentation! This directory contains comprehensive documentation for using, configuring, and contributing to goenv.

## 📚 Table of Contents

### Getting Started

- **[Installation Guide](user-guide/INSTALL.md)** - Complete installation instructions for all platforms
- **[How It Works](user-guide/HOW_IT_WORKS.md)** - Understanding goenv's architecture and workflow
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
goenv global 1.25.2

# Verify
go version
```

## 📖 Documentation Structure

```
docs/
├── README.md                    # This file - documentation index
├── user-guide/                  # User-facing documentation
│   ├── INSTALL.md              # Installation instructions
│   └── HOW_IT_WORKS.md         # Architecture overview
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
goenv versions
```

### Per-Project Versions
Set different Go versions for different projects:
```bash
cd my-project
goenv local 1.24.8
```

### Smart Caching
Intelligent version caching with three-tier freshness checking:
- Fresh cache (< 6 hours): Instant response
- Recent cache (6h-7d): Quick freshness check
- Stale cache (> 7 days): Full refresh

### Offline Mode
Work completely offline using embedded versions:
```bash
export GOENV_OFFLINE=1
goenv install --list  # Uses embedded data, no network calls
```

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
goenv global $(goenv install --latest)

# Or per-project
cd my-go-project
goenv local 1.25.2
```

### CI/CD Pipelines
```bash
# Fast, reproducible builds
export GOENV_OFFLINE=1  # No network dependencies
goenv install 1.25.2
goenv global 1.25.2
go build
```

### Air-Gapped Environments
```bash
# Works without internet
export GOENV_OFFLINE=1
goenv install --list     # Shows 330+ embedded versions
goenv install 1.24.8     # Installs from embedded data
```

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
