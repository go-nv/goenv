# Platform Support Matrix

This document details goenv's behavior and feature support across different operating systems and architectures.

## Table of Contents

- [Supported Platforms](#supported-platforms)
- [Feature Support Matrix](#feature-support-matrix)
- [Platform-Specific Behaviors](#platform-specific-behaviors)
- [Architecture Support](#architecture-support)
- [Known Limitations](#known-limitations)
- [Platform-Specific Configuration](#platform-specific-configuration)

## Supported Platforms

goenv is fully supported on the following platforms:

| OS | Architectures | Status | Notes |
|----|---------------|--------|-------|
| **macOS** | AMD64 (Intel), ARM64 (Apple Silicon) | ✅ Full Support | Native support for both Intel and M-series Macs |
| **Linux** | AMD64, ARM64, ARMv6, ARMv7, 386, PPC64LE, S390X | ✅ Full Support | All major distributions supported |
| **Windows** | AMD64, ARM64, 386 | ✅ Full Support | Native PowerShell & CMD support |
| **FreeBSD** | AMD64, 386, ARM64 | ✅ Full Support | Via Go's FreeBSD support |

## Feature Support Matrix

### Core Commands

| Feature | Linux | macOS | Windows | Notes |
|---------|-------|-------|---------|-------|
| `goenv install` | ✅ | ✅ | ✅ | Full support across all platforms |
| `goenv uninstall` | ✅ | ✅ | ✅ | Full support across all platforms |
| `goenv use` | ✅ | ✅ | ✅ | Full support across all platforms |
| `goenv current` | ✅ | ✅ | ✅ | Full support across all platforms |
| `goenv list` | ✅ | ✅ | ✅ | Full support across all platforms |
| `goenv init` | ✅ | ✅ | ✅ | Shell-specific initialization |
| `goenv rehash` | ✅ | ✅ | ✅ | Platform-specific shim generation |

### Advanced Features

| Feature | Linux | macOS | Windows | Notes |
|---------|-------|-------|---------|-------|
| **Version Detection** | | | | |
| `.go-version` files | ✅ | ✅ | ✅ | Full support |
| `go.mod` toolchain | ✅ | ✅ | ✅ | Go 1.21+ directive support |
| `GOENV_VERSION` env var | ✅ | ✅ | ✅ | Full support |
| **Tools Management** | | | | |
| `goenv tools install` | ✅ | ✅ | ✅ | Full support |
| `goenv tools list` | ✅ | ✅ | ✅ | Full support |
| `goenv sync-tools` | ✅ | ✅ | ✅ | Full support |
| `goenv default-tools` | ✅ | ✅ | ✅ | Full support |
| **Caching** | | | | |
| Smart caching | ✅ | ✅ | ✅ | Full support |
| Cache cleaning | ✅ | ✅ | ✅ | Full support |
| Architecture-aware caches | ✅ | ✅ | ✅ | Per-platform cache isolation |
| **VS Code Integration** | | | | |
| `goenv vscode init` | ✅ | ✅ | ✅ | Full support |
| `goenv vscode sync` | ✅ | ✅ | ✅ | Full support |
| Workspace-relative paths | ✅ | ✅ | ⚠️ | Windows uses backslashes |
| **Diagnostics** | | | | |
| `goenv doctor` | ✅ | ✅ | ✅ | Platform-specific checks |
| Environment detection | ✅ | ✅ | ⚠️ | Limited WSL detection |
| Container detection | ✅ | ✅ | ❌ | Not applicable on Windows |
| **Hooks System** | | | | |
| `log_to_file` | ✅ | ✅ | ✅ | Full support |
| `http_webhook` | ✅ | ✅ | ✅ | Full support |
| `notify_desktop` | ✅ | ✅ | ✅ | Requires libnotify on Linux |
| `check_disk_space` | ✅ | ✅ | ✅ | Full support |
| `set_env` | ✅ | ✅ | ✅ | Full support |
| `run_command` | ✅ | ✅ | ✅ | Shell auto-detection |
| **SBOM Generation** | | | | |
| `goenv sbom project` | ✅ | ✅ | ✅ | Requires SBOM tool installed |
| CycloneDX support | ✅ | ✅ | ✅ | Via cyclonedx-gomod |
| SPDX support | ✅ | ✅ | ✅ | Via syft |
| **Inventory** | | | | |
| `goenv inventory go` | ✅ | ✅ | ✅ | Full support |
| SHA256 checksums | ✅ | ✅ | ✅ | Full support |

### Shell Support

| Shell | Linux | macOS | Windows | Notes |
|-------|-------|-------|---------|-------|
| Bash | ✅ | ✅ | ✅ | Full support (Git Bash on Windows) |
| Zsh | ✅ | ✅ | ✅ | Full support |
| Fish | ✅ | ✅ | ✅ | Full support |
| PowerShell | ⚠️ | ⚠️ | ✅ | Native on Windows, available on Unix |
| CMD | ❌ | ❌ | ✅ | Windows only |
| Ksh | ✅ | ✅ | ❌ | Unix only |
| Dash | ✅ | ✅ | ❌ | Unix only |

## Platform-Specific Behaviors

### macOS

**Rosetta 2 Support (Apple Silicon):**
- ✅ Native ARM64 binaries preferred
- ✅ AMD64 binaries run via Rosetta 2
- ✅ `goenv doctor` detects Rosetta mode
- ⚠️ ARM64EC not applicable (Windows only)

**Universal Binaries:**
```bash
# goenv automatically selects correct architecture
goenv install 1.25.2  # Installs ARM64 on M1/M2/M3

# Force specific architecture (rare)
GOARCH=amd64 goenv install 1.25.2  # Intel binary on Apple Silicon
```

**Keychain Integration:**
- ✅ Secure credential storage for VS Code
- ✅ Native security framework integration

**Desktop Notifications:**
- ✅ Native via `osascript` (always available)
- ✅ Notification Center integration

### Linux

**Distribution Compatibility:**
- ✅ Works on all major distributions (Ubuntu, Debian, Fedora, RHEL, Arch, Alpine, etc.)
- ✅ No distribution-specific dependencies
- ✅ Static binary deployment option

**Desktop Notifications:**
- ⚠️ Requires `libnotify` package
- Install: `sudo apt install libnotify-bin` (Ubuntu/Debian)
- Install: `sudo yum install libnotify` (RHEL/CentOS)
- ❌ Not available on headless servers

**Container Detection:**
- ✅ Docker container detection
- ✅ Podman detection
- ✅ LXC/LXD detection
- ✅ Systemd-nspawn detection

**WSL (Windows Subsystem for Linux):**
- ✅ WSL1 fully supported
- ✅ WSL2 fully supported
- ✅ `goenv doctor` detects WSL environment
- ⚠️ Windows path integration requires configuration
- ⚠️ Desktop notifications don't work (headless)

### Windows

**Shell Support:**
- ✅ PowerShell (recommended)
- ✅ CMD (legacy support)
- ✅ Git Bash (Unix-like experience)

**Path Handling:**
- ⚠️ Backslash path separators (`C:\Users\...`)
- ✅ Forward slashes work in most contexts
- ✅ VS Code paths use backslashes on Windows

**Desktop Notifications:**
- ✅ Native via PowerShell notifications
- ✅ Windows 10/11 Action Center integration

**ARM64 Support:**
- ✅ Native ARM64 Windows support
- ✅ ARM64EC emulation detection
- ✅ `goenv doctor` shows ARM64EC warnings
- ⚠️ Prefer native ARM64 builds (better performance)

**File Permissions:**
- ⚠️ No Unix-style file permissions
- ⚠️ Executable detection uses file extension
- ✅ NTFS ACLs respected

**Symlinks:**
- ⚠️ Requires Developer Mode or admin rights
- ✅ Junction points work without elevation
- ⚠️ Shims use copies instead of symlinks

## Architecture Support

### x86/AMD64

| Platform | Status | Notes |
|----------|--------|-------|
| Linux AMD64 | ✅ Full Support | Most common server platform |
| macOS AMD64 | ✅ Full Support | Intel Macs |
| Windows AMD64 | ✅ Full Support | Most common Windows platform |

### ARM

| Platform | Status | Notes |
|----------|--------|-------|
| Linux ARM64 | ✅ Full Support | Raspberry Pi 4, AWS Graviton, etc. |
| macOS ARM64 | ✅ Full Support | M1/M2/M3 Macs |
| Windows ARM64 | ✅ Full Support | Surface Pro X, Snapdragon PCs |
| Linux ARMv7 | ✅ Full Support | Raspberry Pi 3, older ARM devices |
| Linux ARMv6 | ✅ Full Support | Raspberry Pi 1/Zero |

### Other Architectures

| Platform | Status | Notes |
|----------|--------|-------|
| Linux 386 | ✅ Full Support | 32-bit x86 Linux |
| Windows 386 | ✅ Full Support | 32-bit Windows |
| Linux PPC64LE | ✅ Full Support | IBM POWER systems |
| Linux S390X | ✅ Full Support | IBM z/Architecture mainframes |
| FreeBSD AMD64 | ✅ Full Support | FreeBSD on x86-64 |

### Cross-Compilation

```bash
# Build for different platform/architecture
GOOS=linux GOARCH=arm64 goenv install 1.25.2

# Common cross-compilation targets
GOOS=linux GOARCH=amd64 goenv install 1.25.2    # Linux x86-64
GOOS=darwin GOARCH=arm64 goenv install 1.25.2   # macOS Apple Silicon
GOOS=windows GOARCH=amd64 goenv install 1.25.2  # Windows x86-64

# See available platforms
go tool dist list
```

For detailed cross-building documentation, see [Cross-Building Guide](./advanced/CROSS_BUILDING.md).

## Known Limitations

### All Platforms

- ❌ No support for pre-1.2.1 Go versions (Go bootstrap requirement)
- ⚠️ `system` Go requires pre-existing Go installation on PATH
- ⚠️ Some tools require `go install` per Go version

### macOS-Specific

- ⚠️ AMD64 binaries on Apple Silicon have performance penalty (Rosetta 2)
- ⚠️ Xcode Command Line Tools recommended for some packages
- ⚠️ Gatekeeper may block downloads (first run may prompt)

### Linux-Specific

- ⚠️ Desktop notifications require GUI session and `libnotify`
- ⚠️ Some distributions may need `gcc` for cgo packages
- ⚠️ Alpine Linux requires musl-compatible builds (not always available)

### Windows-Specific

- ⚠️ Long path support requires Windows 10 1607+ and registry setting
- ⚠️ Symlinks require Developer Mode or admin elevation
- ⚠️ Some Unix-focused tools may not work (e.g., tools expecting `/bin/sh`)
- ⚠️ CMD has limited Unicode support compared to PowerShell
- ⚠️ Git Bash provides better Unix compatibility than native shells

### WSL-Specific

- ⚠️ Windows paths not automatically accessible
- ⚠️ Performance penalty for cross-filesystem operations
- ❌ Desktop notifications unavailable (headless environment)
- ⚠️ VS Code integration requires WSL extension

### Container/Docker

- ⚠️ Persistent storage required for `$GOENV_ROOT`
- ⚠️ Cache directories should be volume-mounted for performance
- ✅ Offline mode recommended for reproducibility
- ⚠️ Multi-stage builds should clean up intermediate versions

## Platform-Specific Configuration

### macOS Configuration

```bash
# ~/.zshrc or ~/.bash_profile
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"

# Prefer native ARM64 on Apple Silicon
export GOARCH=arm64  # Optional, auto-detected
```

### Linux Configuration

```bash
# ~/.bashrc
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"

# Install libnotify for desktop notifications (optional)
sudo apt install libnotify-bin  # Ubuntu/Debian
sudo yum install libnotify       # RHEL/CentOS
```

### Windows PowerShell Configuration

```powershell
# $PROFILE (run: notepad $PROFILE)
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
goenv init - powershell | Out-String | Invoke-Expression
```

### Windows CMD Configuration

```cmd
:: Add to startup script or use setx for persistent variables
set GOENV_ROOT=%USERPROFILE%\.goenv
set PATH=%GOENV_ROOT%\bin;%PATH%
```

### WSL Configuration

```bash
# ~/.bashrc
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"

# Optional: Access Windows Go installations (not recommended)
# export PATH="/mnt/c/goenv/shims:$PATH"
```

### Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.25-alpine

# Install goenv
RUN apk add --no-cache curl git bash
RUN curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

ENV GOENV_ROOT="/root/.goenv"
ENV PATH="$GOENV_ROOT/bin:$PATH"

# Use offline mode for reproducibility
ENV GOENV_OFFLINE=1

# Install specific Go version
RUN goenv install 1.25.2 && goenv use 1.25.2 --global

# Your app
WORKDIR /app
COPY . .
RUN go build -o myapp
```

## Platform Detection

goenv automatically detects platform characteristics:

```bash
# Check platform detection
goenv doctor

# Example output
✅ Platform: darwin/arm64
✅ Shell: zsh
✅ Rosetta: Not running under Rosetta
✅ Container: Not in container
```

**Detection includes:**
- Operating system and architecture
- Shell type and version
- Rosetta 2 status (macOS)
- ARM64EC status (Windows)
- Container environment (Docker, Podman, etc.)
- WSL version (if applicable)
- Filesystem characteristics

## CI/CD Platform Support

| CI/CD Platform | Linux | macOS | Windows | Notes |
|----------------|-------|-------|---------|-------|
| GitHub Actions | ✅ | ✅ | ✅ | All runners supported |
| GitLab CI | ✅ | ✅ | ✅ | All executor types |
| CircleCI | ✅ | ✅ | ❌ | Windows support limited |
| Travis CI | ✅ | ✅ | ✅ | All platforms supported |
| Azure Pipelines | ✅ | ✅ | ✅ | All agents supported |
| Jenkins | ✅ | ✅ | ✅ | All node types |
| Drone CI | ✅ | ❌ | ❌ | Linux containers only |
| AWS CodeBuild | ✅ | ❌ | ✅ | Linux and Windows images |

**CI/CD Best Practices:**
- Use `GOENV_OFFLINE=1` for reproducibility
- Cache `$GOENV_ROOT/versions` between runs
- Use `--force` for non-interactive commands
- Set timeouts for network operations
- Pin Go versions with `.go-version` files

See [CI/CD Integration Guide](../advanced/CI_CD_GUIDE.md) for detailed examples.

## Performance Characteristics

### Installation Speed

| Platform | Typical Install Time | Notes |
|----------|---------------------|-------|
| Linux AMD64 | 30-60s | Fast on modern hardware |
| macOS ARM64 | 30-60s | Native ARM64 binaries |
| macOS AMD64 (Rosetta) | 45-90s | Slower due to emulation |
| Windows AMD64 | 45-90s | Antivirus may slow down |
| ARM64 (embedded) | 60-120s | Lower-powered hardware |

### Cache Performance

| Platform | Cache Type | Performance |
|----------|------------|-------------|
| Linux | Native filesystem | Excellent |
| macOS | APFS | Excellent |
| Windows | NTFS | Good |
| WSL2 | ext4 on VHD | Very Good |
| WSL1 | NTFS translation | Fair |
| Docker volume | Varies | Depends on driver |

## Getting Help

- **Platform-specific issues**: [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **Feature requests**: [GitHub Discussions](https://github.com/go-nv/goenv/discussions)
- **General help**: `goenv doctor` for diagnostics

## See Also

- [Installation Guide](./user-guide/INSTALL.md)
- [Cross-Building Guide](./advanced/CROSS_BUILDING.md)
- [CI/CD Integration](../advanced/CI_CD_GUIDE.md)
- [VS Code Integration](./user-guide/VSCODE_INTEGRATION.md)
- [Hooks System](HOOKS_QUICKSTART.md)
