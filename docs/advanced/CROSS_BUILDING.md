# Cross-Platform Building Playbooks

This guide provides platform-specific recipes for building portable Go binaries that work across different systems and architectures.

## Table of Contents

- [Linux: glibc vs musl](#linux-glibc-vs-musl)
- [macOS: Universal Binaries](#macos-universal-binaries)
- [Windows: MSVC vs MinGW](#windows-msvc-vs-mingw)
- [Static Linking](#static-linking)
- [Build Flags Reference](#build-flags-reference)

## Linux: glibc vs musl

### Problem

Binaries built on one Linux system may not work on another due to C library (libc) differences:

- **glibc** (GNU C Library) - Standard on most Linux distributions (Ubuntu, Fedora, Debian)
- **musl** - Lightweight alternative used by Alpine Linux and embedded systems

**Symptom**: `error while loading shared libraries: libc.so.6: cannot open shared object file`

### Solutions

#### Option 1: Static Binary (No CGO - Recommended)

Build a fully static binary that doesn't depend on system libraries:

```bash
# Disable CGO for pure Go static binary
CGO_ENABLED=0 go build \
  -ldflags="-s -w" \
  -tags netgo,osusergo \
  -o myapp .
```

**Pros:**

- Works on any Linux distribution (glibc, musl, or none)
- Smallest binary size with `-s -w` (strip symbols)
- No runtime dependencies

**Cons:**

- Cannot use C libraries or CGO
- DNS resolution uses pure Go implementation (slower)
- Some stdlib features unavailable (e.g., user lookups without `osusergo`)

#### Option 2: musl Static (With CGO Support)

Build with musl for CGO support while remaining portable:

```bash
# Install musl-gcc (Ubuntu/Debian)
sudo apt-get install musl-tools

# Build with musl
CC=musl-gcc \
CGO_ENABLED=1 \
go build \
  -ldflags="-s -w -linkmode external -extldflags '-static'" \
  -o myapp .
```

**Pros:**

- CGO support (can use C libraries)
- Static binary works everywhere
- Smaller than glibc static builds

**Cons:**

- Requires musl-gcc toolchain
- Some C libraries may not compile with musl
- Larger binary than pure Go

#### Option 3: Build in Old glibc Environment

Build with the oldest glibc version you need to support:

```bash
# Use Docker with old Debian/Ubuntu
docker run --rm -v "$PWD":/work -w /work \
  debian:stretch \
  bash -c "apt-get update && apt-get install -y golang-go && go build -o myapp ."

# Or use specific glibc version container
docker run --rm -v "$PWD":/work -w /work \
  quay.io/pypa/manylinux2014_x86_64 \
  bash -c "go build -o myapp ."
```

**Pros:**

- Full CGO support with any C library
- Uses standard glibc

**Cons:**

- Requires Docker or old system
- Binary still depends on glibc (but older version)
- Won't work on musl systems

### Checking glibc Version

```bash
# Check your system's glibc version
ldd --version

# Check what a binary needs
objdump -T myapp | grep GLIBC

# Or with goenv doctor
goenv doctor
```

### Recommended Build Matrix

For maximum compatibility:

```yaml
# .github/workflows/build.yml
strategy:
  matrix:
    include:
      # Static binary for any Linux
      - os: ubuntu-latest
        cgo: 0
        tags: "netgo,osusergo"
        output: "myapp-linux-amd64-static"

      # musl binary for CGO support
      - os: ubuntu-latest
        cgo: 1
        cc: musl-gcc
        output: "myapp-linux-amd64-musl"

      # glibc binary (built on old Ubuntu for compatibility)
      - os: ubuntu-20.04
        cgo: 1
        output: "myapp-linux-amd64-glibc"
```

## macOS: Universal Binaries

### Problem

Apple Silicon Macs (M1/M2/M3) use ARM64, while older Macs use AMD64 (x86_64). Universal binaries work on both.

### Solution: Build and Lipo

```bash
# Build for both architectures
GOOS=darwin GOARCH=amd64 go build -o myapp-amd64 .
GOOS=darwin GOARCH=arm64 go build -o myapp-arm64 .

# Combine into universal binary
lipo -create -output myapp myapp-amd64 myapp-arm64

# Verify
lipo -info myapp
# Output: Architectures in the fat file: myapp are: x86_64 arm64

# Or use file command
file myapp
# Output: myapp: Mach-O universal binary with 2 architectures
```

### CGO with Universal Binaries

If using CGO, you need to compile C code for both architectures:

```bash
# For AMD64
CGO_ENABLED=1 \
GOOS=darwin \
GOARCH=amd64 \
go build -o myapp-amd64 .

# For ARM64 (must be on ARM64 Mac or have cross-compilation setup)
CGO_ENABLED=1 \
GOOS=darwin \
GOARCH=arm64 \
go build -o myapp-arm64 .

# Combine
lipo -create -output myapp myapp-amd64 myapp-arm64
```

**Note**: Cross-compiling CGO for macOS is complex. Best practice is to build each architecture natively or use CI runners.

### Deployment Target

Set minimum macOS version:

```bash
# Support macOS 10.13+
CGO_ENABLED=1 \
CGO_CFLAGS="-mmacosx-version-min=10.13" \
CGO_LDFLAGS="-mmacosx-version-min=10.13" \
go build -o myapp .

# Or via environment
export MACOSX_DEPLOYMENT_TARGET=10.13
go build -o myapp .
```

### Code Signing & Notarization

For distribution outside the App Store:

```bash
# Build universal binary
lipo -create -output myapp myapp-amd64 myapp-arm64

# Sign the binary
codesign --sign "Developer ID Application: Your Name" \
  --options runtime \
  --timestamp \
  myapp

# Notarize (requires Apple Developer account)
ditto -c -k --keepParent myapp myapp.zip
xcrun notarytool submit myapp.zip \
  --apple-id "your@email.com" \
  --team-id "TEAMID" \
  --wait

# Staple the notarization ticket
xcrun stapler staple myapp
```

### Recommended Build Script

```bash
#!/bin/bash
set -e

# Build for both architectures
echo "Building AMD64..."
GOOS=darwin GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o build/myapp-amd64 .

echo "Building ARM64..."
GOOS=darwin GOARCH=arm64 go build \
  -ldflags="-s -w" \
  -o build/myapp-arm64 .

# Create universal binary
echo "Creating universal binary..."
lipo -create \
  -output build/myapp \
  build/myapp-amd64 \
  build/myapp-arm64

# Verify
lipo -info build/myapp

echo "✅ Universal binary created: build/myapp"
```

## Windows: MSVC vs MinGW

### Problem

Windows has two main C compiler toolchains:

- **MSVC** (Microsoft Visual C++) - Official Microsoft compiler
- **MinGW** (Minimalist GNU for Windows) - GCC port for Windows

### CGO-Free (Recommended)

```bash
# Static Windows binary (no CGO)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -H=windowsgui" \
  -o myapp.exe .
```

**Flags:**

- `-H=windowsgui` - Suppress console window (for GUI apps)
- `-H=windows` - Console application (default)

### MinGW (CGO Support)

```bash
# Install MinGW-w64 (Ubuntu/Debian)
sudo apt-get install gcc-mingw-w64-x86-64

# Or on macOS
brew install mingw-w64

# Build for Windows with CGO
CC=x86_64-w64-mingw32-gcc \
CXX=x86_64-w64-mingw32-g++ \
CGO_ENABLED=1 \
GOOS=windows \
GOARCH=amd64 \
go build -o myapp.exe .
```

### MSVC (Windows Only)

```powershell
# Install Visual Studio Build Tools
# Download from: https://visualstudio.microsoft.com/downloads/

# In Developer Command Prompt for VS
$env:CGO_ENABLED=1
$env:CC="cl"
go build -o myapp.exe .
```

### Checking Dependencies

```bash
# List DLL dependencies (Windows)
dumpbin /DEPENDENTS myapp.exe

# Or use objdump (cross-platform)
objdump -p myapp.exe | grep "DLL Name"

# Check for Visual C++ Runtime dependency
# If you see VCRUNTIME140.dll, users need VC++ Redistributable
```

### Bundling VC++ Runtime

If your binary requires `VCRUNTIME140.dll`:

**Option 1**: Static linking

```bash
# Link VC runtime statically (MSVC only)
go build \
  -ldflags="-linkmode external -extldflags '-static'" \
  -o myapp.exe .
```

**Option 2**: Include redistributable

Include `vc_redist.x64.exe` with your installer:

- Download: https://aka.ms/vs/17/release/vc_redist.x64.exe
- Run during installation

**Option 3**: Use MinGW (no VC++ runtime needed)

### Windows ARM64 and ARM64EC

Windows on ARM devices (Surface Pro X, Copilot+ PCs, etc.) support two execution modes:

- **ARM64 (native)** - Pure ARM64 binaries, best performance
- **ARM64EC (Emulation Compatible)** - Hybrid mode allowing x64 interop

```bash
# Native ARM64 (recommended for pure Go)
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -o myapp.exe .

# With CGO (requires ARM64 cross-compiler)
# Best to build on native ARM64 Windows for CGO support
# Cross-compilation is complex due to ARM64 toolchain requirements
```

**Detection and Warnings:**

The `goenv doctor` command detects ARM64 and ARM64EC execution modes:

```bash
goenv doctor
# ✅ Windows environment
#    Running on Windows ARM64 (native)
#
# OR:
#
# ⚠️  Windows environment
#    Running on Windows ARM64EC (emulation-compatible mode)
#    Consider using native ARM64 builds for better performance
```

**Best Practices:**

1. **Prefer native ARM64 builds**: Better performance, no emulation overhead
2. **Test on target hardware**: ARM64EC behavior may differ from native ARM64
3. **Check `goenv doctor`**: Warns if running in ARM64EC mode
4. **For CGO projects**: Build on native ARM64 Windows machines
5. **Static binaries work best**: `CGO_ENABLED=0` for maximum compatibility

**ARM64EC Considerations:**

ARM64EC (Emulation Compatible) allows ARM64 processes to load x64 DLLs, but:
- May have performance implications for CGO code
- Not all x64 libraries work correctly in ARM64EC mode
- Native ARM64 is always preferred when possible
- `goenv doctor` will flag ARM64EC execution for awareness

### Recommended Build Matrix

```yaml
# Build for all Windows architectures
strategy:
  matrix:
    goarch: [amd64, 386, arm64]

steps:
  - name: Build Windows binary
    run: |
      GOOS=windows GOARCH=${{ matrix.goarch }} CGO_ENABLED=0 \
      go build -ldflags="-s -w" \
      -o myapp-windows-${{ matrix.goarch }}.exe .
```

## Static Linking

### Why Static Linking?

- **Portability**: Binary works without system dependencies
- **Deployment**: Single file to distribute
- **Containers**: Smaller images (FROM scratch possible)

### Pure Go Static Binary

```bash
CGO_ENABLED=0 go build \
  -a \
  -ldflags="-s -w -extldflags '-static'" \
  -tags netgo,osusergo \
  -installsuffix nocgo \
  -o myapp .
```

**Flags explained:**

- `CGO_ENABLED=0` - Disable CGO
- `-a` - Force rebuild of all packages
- `-s -w` - Strip symbols and DWARF info (smaller binary)
- `-extldflags '-static'` - Statically link
- `-tags netgo` - Use pure Go DNS resolver
- `-tags osusergo` - Use pure Go user/group lookups
- `-installsuffix nocgo` - Separate build cache

### CGO Static Binary (Linux)

```bash
CGO_ENABLED=1 go build \
  -a \
  -ldflags="-s -w -linkmode external -extldflags '-static'" \
  -tags netgo,osusergo \
  -installsuffix cgo \
  -o myapp .
```

### Verify Static Linking

```bash
# Linux
ldd myapp
# Should output: "not a dynamic executable"

# Or use file command
file myapp
# Should contain "statically linked"

# macOS (harder to achieve true static linking)
otool -L myapp

# Windows
dumpbin /DEPENDENTS myapp.exe
```

## Build Flags Reference

### Common ldflags

```bash
# Strip symbols and debug info (smaller binary)
-ldflags="-s -w"

# Set version info
-ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)"

# Static linking
-ldflags="-linkmode external -extldflags '-static'"

# All combined
-ldflags="-s -w -X main.version=1.0.0 -linkmode external -extldflags '-static'"
```

### Build Tags

```bash
# Pure Go networking (no CGO DNS)
-tags netgo

# Pure Go user/group lookups
-tags osusergo

# Combined
-tags netgo,osusergo

# Platform-specific code
-tags linux
-tags darwin
-tags windows
```

### Trimpath (Reproducible Builds)

```bash
# Remove absolute paths from binary
-trimpath

# Full reproducible build
CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags="-s -w -buildid=" \
  -o myapp .
```

## Complete Examples

### Maximum Portability (Any Linux)

```bash
#!/bin/bash
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64 \
go build \
  -a \
  -trimpath \
  -ldflags="-s -w -extldflags '-static'" \
  -tags netgo,osusergo \
  -o myapp-linux-amd64 .

echo "Binary info:"
file myapp-linux-amd64
ldd myapp-linux-amd64 || echo "Static binary (good!)"
```

### macOS Universal with Version

```bash
#!/bin/bash
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse HEAD)
LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$COMMIT"

# AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o myapp-amd64 .

# ARM64
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o myapp-arm64 .

# Universal
lipo -create -output myapp myapp-amd64 myapp-arm64
rm myapp-amd64 myapp-arm64

echo "Universal binary created:"
lipo -info myapp
```

### Windows All Architectures

```bash
#!/bin/bash
VERSION=$(git describe --tags --always)
LDFLAGS="-s -w -X main.version=$VERSION"

for arch in amd64 386 arm64; do
  echo "Building for windows/$arch..."
  CGO_ENABLED=0 \
  GOOS=windows \
  GOARCH=$arch \
  go build -ldflags="$LDFLAGS" -o myapp-windows-$arch.exe .
done

echo "Windows binaries created:"
ls -lh myapp-windows-*.exe
```

### Windows ARM64 and ARM64EC

Windows on ARM64 systems (like Surface Pro X, Snapdragon-powered laptops) support two binary formats:

- **ARM64**: Native ARM64 binaries (best performance)
- **ARM64EC**: Emulation-compatible binaries (can mix x64 and ARM64 code)

#### Recommended Approach: Native ARM64

Build native ARM64 binaries for best performance:

```bash
CGO_ENABLED=0 \
GOOS=windows \
GOARCH=arm64 \
go build -ldflags="-s -w" -o myapp-arm64.exe .
```

**Benefits:**
- Full native performance
- No emulation overhead
- Smaller binaries
- Better battery life

**Compatibility:**
- Runs natively on Windows ARM64 devices
- Does NOT run on x64 Windows (use separate x64 build)

#### Understanding ARM64EC

ARM64EC (Emulation Compatible) is a Microsoft technology that allows mixing x64 and ARM64 code in the same process. However:

- **Go does not support ARM64EC compilation** (as of Go 1.25)
- ARM64EC is primarily for C/C++ codebases with x64 dependencies
- Pure Go applications should use native ARM64

#### goenv doctor Warnings

When running Windows binaries on ARM64 systems, `goenv doctor` will detect architecture mismatches:

```bash
# On Windows ARM64, running x64 binary
> goenv doctor
⚠️  Architecture mismatch detected
   Running x64 binary on arm64 system
   Recommendation: Use native ARM64 build for better performance

   Build command:
   GOOS=windows GOARCH=arm64 go build -o myapp-arm64.exe .
```

**What the warning means:**
- You're running an x64 (amd64) binary via emulation
- Emulation works but reduces performance and battery life
- Native ARM64 binaries are recommended

**When to ignore:**
- Third-party tools without ARM64 builds
- Temporarily testing x64 code
- Cross-platform binaries that bundle both architectures

#### Building Multi-Architecture Windows Binaries

```bash
#!/bin/bash
VERSION=$(git describe --tags --always)
LDFLAGS="-s -w -X main.version=$VERSION"

# Build for all Windows architectures
for arch in amd64 arm64; do
  echo "Building for windows/$arch..."
  CGO_ENABLED=0 \
  GOOS=windows \
  GOARCH=$arch \
  go build -ldflags="$LDFLAGS" -o myapp-windows-$arch.exe .
done

echo "Windows binaries created:"
ls -lh myapp-windows-*.exe

# Verify architectures
file myapp-windows-*.exe
```

**Distribution Strategy:**

1. **Single-architecture releases**: Ship separate downloads per architecture
   - `myapp-windows-amd64.exe` for Intel/AMD systems
   - `myapp-windows-arm64.exe` for ARM64 systems

2. **Auto-detection installer**: Create an installer that detects system architecture and installs the correct binary

3. **Universal launcher**: Use a small launcher that detects architecture and exec's the correct binary

#### Key Takeaways

- ✅ **Use native ARM64 builds** on Windows ARM64 for best performance
- ❌ **Don't rely on x64 emulation** for production workloads
- ℹ️ **ARM64EC is not needed** for pure Go applications
- 🔍 **Use `goenv doctor`** to detect architecture mismatches and get recommendations

## Troubleshooting

### "version mismatch" Errors

```bash
# Clean all caches
goenv cache clean all

# Or use cache management
goenv cache clean all
```

### Dynamic Library Errors

```bash
# Check dependencies
ldd myapp                    # Linux
otool -L myapp              # macOS
dumpbin /DEPENDENTS myapp.exe  # Windows

# If libraries are missing, rebuild statically
```

### Cross-Compilation CGO Errors

```bash
# Install cross-compilers
# Ubuntu/Debian
sudo apt-get install gcc-multilib gcc-mingw-w64

# macOS
brew install mingw-w64
```

## See Also

- [goenv doctor](../reference/COMMANDS.md#goenv-doctor) - Check system compatibility
- [Cache Management](../reference/COMMANDS.md#goenv-cache) - Manage architecture-specific caches
- [Environment Detection](../../ENVIRONMENT_DETECTION.md) - Container and filesystem detection
