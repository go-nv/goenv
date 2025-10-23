# Bootstrap Installation Guide

## Understanding the Bootstrap Problem

The Go-based goenv has a classic "chicken-and-egg" problem:

- **To build goenv from source**, you need Go installed
- **But goenv's purpose** is to install and manage Go versions
- **Solution**: Pre-built binaries!

## Installation Methods Compared

### 1. Pre-built Binary (⭐ Recommended)

**Pros:**

- ✅ No dependencies - works on a fresh system
- ✅ Fast - just download and run
- ✅ Perfect for CI/CD pipelines
- ✅ Ideal for users new to Go

**Cons:**

- ⚠️ Must trust the release process
- ⚠️ Slightly behind latest commits

**Use when:**

- Installing goenv for the first time
- Setting up a new machine
- You don't have Go installed
- You want the stable release

**Installation:**

```bash
# Automatic (Linux/macOS)
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Manual download from https://github.com/go-nv/goenv/releases/latest
```

### 2. Git Clone + Build

**Pros:**

- ✅ Latest development version
- ✅ Easy to contribute changes
- ✅ Full source code access

**Cons:**

- ⚠️ Requires Go to be installed
- ⚠️ Extra build step

**Use when:**

- Contributing to goenv
- You already have Go installed
- You need bleeding-edge features
- Testing development versions

**Installation:**

```bash
git clone https://github.com/go-nv/goenv.git ~/.goenv
cd ~/.goenv
make build  # Requires Go!
```

### 3. Package Manager (Homebrew on macOS)

**Pros:**

- ✅ Integrates with system package manager
- ✅ Automatic dependency handling
- ✅ Easy updates

**Cons:**

- ⚠️ Platform-specific
- ⚠️ May lag behind releases

**Use when:**

- You're on macOS
- You prefer Homebrew for package management
- You want automatic updates via `brew upgrade`

**Installation:**

```bash
brew install goenv
```

## How goenv Builds Work

### Build-time vs Runtime Dependencies

**Build-time (only needed to compile goenv):**

- Go compiler (1.21+)
- Make (for Makefile)
- Git (for version info)

**Runtime (what the binary needs to run):**

- System libraries only (libc, etc.)
- **No Go installation required!**

### What Happens During Build

```bash
make build
```

This runs:

```bash
go build -ldflags "-X main.version=... -X main.commit=..." -o goenv .
```

Which:

1. Compiles all Go source code
2. Links system libraries
3. Embeds version information
4. Creates a **standalone binary**

The resulting `goenv` binary:

- Is a native executable
- Contains all Go code compiled in
- Only depends on OS libraries
- **Does not need Go to run!**

### Verifying No Go Dependency

You can verify the binary doesn't need Go:

```bash
# macOS
otool -L goenv

# Linux
ldd goenv

# Both show only system libraries, no Go!
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build Project
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      # Install goenv (no Go needed!)
      - name: Install goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH

      # Now use goenv to install Go
      - name: Install Go via goenv
        run: |
          goenv install 1.22.0
          goenv use 1.22.0 --global
          eval "$(goenv init -)"

      # Build your project
      - name: Build
        run: go build ./...
```

### GitLab CI Example

```yaml
build:
  image: alpine:latest
  before_script:
    # Install goenv (no Go needed!)
    - apk add --no-cache curl bash
    - curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
    - export PATH="$HOME/.goenv/bin:$PATH"
    - goenv install 1.22.0
    - goenv use 1.22.0 --global
    - eval "$(goenv init -)"
  script:
    - go build ./...
```

## Docker Integration

### Multi-stage Build with goenv

```dockerfile
# Stage 1: Install goenv and Go
FROM alpine:latest AS goenv
RUN apk add --no-cache curl bash
RUN curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
ENV PATH="/root/.goenv/bin:$PATH"
RUN goenv install 1.22.0
RUN goenv use 1.22.0 --global

# Stage 2: Build application
FROM alpine:latest
COPY --from=goenv /root/.goenv /root/.goenv
ENV PATH="/root/.goenv/bin:/root/.goenv/shims:$PATH"
WORKDIR /app
COPY . .
RUN go build -o myapp .

# Stage 3: Runtime
FROM alpine:latest
COPY --from=goenv /root/.goenv/versions/1.22.0 /usr/local/go
COPY --from=build /app/myapp /myapp
CMD ["/myapp"]
```

## Bootstrapping a New System

### Fresh Linux Install

```bash
# 1. Install goenv (no dependencies!)
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# 2. Configure shell
cat >> ~/.bashrc << 'EOF'
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
EOF

# 3. Reload shell
source ~/.bashrc

# 4. Install Go
goenv install 1.22.0
goenv use 1.22.0 --global

# 5. Start coding!
go version
```

### Fresh macOS Install

```bash
# Option 1: Homebrew (handles everything)
brew install goenv

# Option 2: Binary install
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Configure shell
echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.zshrc
echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.zshrc
echo 'eval "$(goenv init -)"' >> ~/.zshrc

# Reload and use
source ~/.zshrc
goenv install 1.22.0
goenv use 1.22.0 --global
```

### Fresh Windows Install

```powershell
# Install goenv
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex

# Configure PowerShell profile
Add-Content $PROFILE @"
`$env:GOENV_ROOT = "`$HOME\.goenv"
`$env:PATH = "`$env:GOENV_ROOT\bin;`$env:PATH"
& goenv init - | Invoke-Expression
"@

# Reload and use
. $PROFILE
goenv install 1.22.0
goenv use 1.22.0 --global
```

## Release Process

### How Pre-built Binaries Are Created

1. **Version Tag**: Maintainer creates a git tag (e.g., `v2.1.0`)
2. **GitHub Release**: Tag triggers `.github/workflows/release-binaries.yml`
3. **GoReleaser**: Builds binaries for all platforms:
   - Linux: amd64, arm64, arm (v6, v7)
   - macOS: amd64, arm64
   - Windows: amd64
   - FreeBSD: amd64
4. **Upload**: Binaries uploaded to GitHub Releases
5. **Checksums**: SHA256 checksums generated for verification

### Verifying Download Integrity

```bash
# Download binary and checksum
VERSION=2.1.0
curl -LO https://github.com/go-nv/goenv/releases/download/v${VERSION}/goenv_${VERSION}_linux_amd64.tar.gz
curl -LO https://github.com/go-nv/goenv/releases/download/v${VERSION}/goenv_${VERSION}_checksums.txt

# Verify checksum
sha256sum -c goenv_${VERSION}_checksums.txt --ignore-missing
```

## Frequently Asked Questions

### Q: Why does building require Go but running doesn't?

**A:** Go compiles to native machine code. The compiler needs Go, but the output is a standalone binary that only needs OS libraries.

### Q: Can I use goenv without internet?

**A:** Yes! After downloading the goenv binary, it works offline. It has embedded version data and caches API responses for 24 hours.

### Q: What if I already have Go installed?

**A:** No problem! You can either:

1. Use the binary (ignore your existing Go)
2. Build from source using your existing Go
3. Use Homebrew (uses system Go to build)

### Q: How do I update goenv?

**A:** Depends on installation method:

- **Binary**: Download new version from releases
- **Git clone**: `cd ~/.goenv && git pull && make build`
- **Homebrew**: `brew upgrade goenv`
- **Self-update**: `goenv update` (planned feature)

### Q: Which installation method is fastest?

**A:**

1. Binary download: ~5 seconds
2. Homebrew: ~30 seconds (downloads + builds)
3. Git clone: ~60 seconds (clone + build)

### Q: Can I switch between installation methods?

**A:** Yes, but clean up the old installation first:

```bash
# Remove old installation
rm -rf ~/.goenv  # or brew uninstall goenv

# Install with new method
curl -sfL ... | bash
```

## Troubleshooting

### Binary won't run on Linux

**Issue**: `./goenv: /lib/x86_64-linux-gnu/libc.so.6: version 'GLIBC_2.XX' not found`

**Solution**: Your system libc is too old. Either:

1. Upgrade your OS
2. Build from source with older Go version
3. Use Docker with Alpine Linux

### Permission denied on macOS

**Issue**: `"goenv" cannot be opened because the developer cannot be verified`

**Solution**:

```bash
xattr -d com.apple.quarantine ~/.goenv/bin/goenv
```

### PATH not working after install

**Issue**: `goenv: command not found`

**Solution**:

```bash
# Check if binary exists
ls -l ~/.goenv/bin/goenv

# Add to PATH manually
export PATH="$HOME/.goenv/bin:$PATH"

# Make permanent (add to ~/.bashrc or ~/.zshrc)
```

## Summary

| Method              | Go Required? | Use Case                             |
| ------------------- | ------------ | ------------------------------------ |
| **Binary Download** | ❌ No        | First install, CI/CD, fresh systems  |
| **Git Clone**       | ✅ Yes       | Development, contributors            |
| **Homebrew**        | ⚠️ Automatic | macOS users, prefer package managers |

**Recommendation**: Start with binary download, switch to git clone if you want to contribute.
