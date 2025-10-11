# 🚀 goenv 2.0 - Go Implementation

**goenv has been completely rewritten in Go!** 🎉

The new Go implementation provides:

- ✅ **Native cross-platform support** - Single binary, no bash required
- ✅ **Visual progress bars** - See download progress in real-time
- ✅ **Mirror support** - Faster downloads with automatic fallback
- ✅ **Better performance** - Native Go is faster than bash scripts
- ✅ **Enhanced error messages** - Clear, actionable error reporting
- ✅ **Type safety** - Compile-time checks prevent runtime errors
- ✅ **100% compatible** - Works with existing goenv installations

## Quick Comparison

| Feature             | Bash Version   | Go Version                 |
| ------------------- | -------------- | -------------------------- |
| Progress Indication | Basic text     | Visual bar with speed/size |
| Error Handling      | Basic          | Comprehensive              |
| Cross-Platform      | Requires bash  | Native binary              |
| Dependencies        | curl/wget/bash | None                       |
| Performance         | Good           | Better                     |
| Type Safety         | No             | Yes                        |

## Installation

### From Source

```bash
git clone https://github.com/go-nv/goenv.git
cd goenv
make build
export PATH="$PWD:$PATH"
```

### Using go install (recommended)

```bash
go install github.com/go-nv/goenv@latest
```

### Pre-built Binaries

Download from [Releases](https://github.com/go-nv/goenv/releases)

## New Features in 2.0

### Visual Progress Bar

```bash
$ goenv install 1.21.5
Installing Go go1.21.5...
Downloading go1.21.5.darwin-arm64.tar.gz
████████████████████░░░░░░░ 67% | 45 MB/67 MB | 5.2 MB/s
```

### Mirror Support

```bash
export GO_BUILD_MIRROR_URL=https://golang.google.cn/dl
goenv install 1.21.5
# Automatically falls back to official source if mirror fails
```

### Verbose Mode

```bash
$ goenv install --verbose 1.21.5
Installing Go go1.21.5...
Downloading from: https://go.dev/dl/go1.21.5.darwin-arm64.tar.gz
Download completed and verified
Extracting archive...
Extraction completed
Successfully installed Go go1.21.5
```

### Quiet Mode (for scripts)

```bash
$ goenv install --quiet 1.21.5
# Silent unless error occurs
```

### Keep Downloaded Files

```bash
$ goenv install --keep 1.21.5
Keeping downloaded file: /tmp/goenv-download-12345.tar.gz
```

## All Commands Available

### Version Management

```bash
goenv install 1.21.5          # Install Go version
goenv uninstall 1.21.5        # Remove Go version
goenv versions                # List installed versions
goenv version                 # Show current version
goenv global 1.21.5           # Set global version
goenv local 1.21.5            # Set local version (per-project)
```

### Installation Options

```bash
goenv install --list          # List available versions
goenv install -v 1.21.5       # Verbose installation
goenv install -q 1.21.5       # Quiet installation
goenv install -k 1.21.5       # Keep downloaded files
goenv install -f 1.21.5       # Force reinstall
```

### Shell Integration

```bash
eval "$(goenv init -)"        # Initialize in shell
goenv init -                  # Show init script
goenv rehash                  # Rebuild shims
goenv which go                # Show path to executable
```

### Advanced

```bash
goenv exec go version         # Execute command with version
goenv shims                   # List shims
goenv prefix 1.21.5           # Show version install path
goenv root                    # Show GOENV_ROOT
```

## Shell Support

goenv 2.0 supports:

- ✅ bash
- ✅ zsh
- ✅ fish
- ✅ ksh

## Platform Support

Pre-built binaries available for:

- ✅ macOS (Intel & Apple Silicon)
- ✅ Linux (amd64, arm64, 386, armv6l)
- ✅ Windows (amd64, 386)
- ✅ FreeBSD (amd64, 386)
- ✅ OpenBSD (amd64, 386)

## Migration from Bash Version

**Good news!** No migration needed! The Go version:

- Uses the same `GOENV_ROOT` directory structure
- Reads the same `.go-version` files
- Has the same command-line interface
- Works with your existing setup

Simply replace the bash version with the Go binary and everything continues to work.

## Environment Variables

```bash
GOENV_ROOT=/path/to/goenv        # Installation directory
GOENV_VERSION=1.21.5             # Override version
GO_BUILD_MIRROR_URL=https://...  # Custom mirror URL
GOENV_DEBUG=1                    # Enable debug output
```

## Testing

goenv 2.0 includes:

- ✅ 238+ automated tests
- ✅ 100% test pass rate
- ✅ Behavioral parity verification
- ✅ Cross-platform testing

```bash
go test ./...
# All tests pass! ✅
```

## Building from Source

Requirements:

- Go 1.21 or later

```bash
git clone https://github.com/go-nv/goenv.git
cd goenv
make build
go test ./...  # Run tests
```

## Contributing

Contributions welcome! The Go codebase is:

- Well-structured with clear packages
- Fully tested with table-driven tests
- Type-safe with compile-time checks
- Easy to understand and modify

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Documentation

- [Installation Guide](INSTALL.md)
- [Command Reference](COMMANDS.md)
- [How It Works](HOW_IT_WORKS.md)
- [Advanced Configuration](ADVANCED_CONFIGURATION.md)
- [Migration Status](MIGRATION_COMPLETE.md)

## License

MIT License - see [LICENSE](LICENSE)

## Acknowledgments

- Original bash version by the goenv team
- Go implementation built on the solid foundation of the bash version
- Progress bar by [schollz/progressbar](https://github.com/schollz/progressbar)

---

**goenv 2.0 - Manage Go versions like a pro!** 🚀
