# Scripts Directory

This directory contains utility scripts for goenv development and releases.

## Structure

Each script is in its own subdirectory with a `main.go` file. This prevents `main()` redeclaration errors when working with multiple scripts.

```
scripts/
├── generate_embedded_versions/
│   └── main.go              # Generates embedded version data
└── test_windows_compatibility/
    └── main.go              # Tests Windows compatibility
```

## Scripts

### generate_embedded_versions

Fetches all Go versions from the official API and generates `internal/version/embedded_versions.go` with embedded version data for offline use.

**Usage:**

```bash
# Using make (recommended)
make generate-embedded

# Direct execution
go run scripts/generate_embedded_versions/main.go
```

**When to run:**

- Before creating a release (automatically done by `make cross-build`)
- Monthly to keep embedded data fresh
- After major Go releases

**What it does:**

1. Fetches all versions from https://go.dev/dl/?mode=json&include=all
2. Filters to major platforms (darwin, linux, windows, freebsd)
3. Generates `internal/version/embedded_versions.go` (~817 KB, 331 versions)
4. Takes ~2.5 seconds

**See also:** `EMBEDDED_VERSIONS.md` for detailed documentation

### test_windows_compatibility

Comprehensive test suite that verifies Windows support throughout the goenv stack.

**Usage:**

```bash
# Linux/macOS
make test-windows

# Windows (PowerShell)
.\build.ps1 test-windows

# Windows (Batch)
build.bat test-windows

# Direct execution (any platform)
go run scripts/test_windows_compatibility/main.go
```

**What it tests:**

1. Embedded versions include Windows files
2. `GetFileForPlatform()` works for Windows
3. Platform distribution (Windows representation)
4. All Windows architectures (amd64, 386, arm64)
5. Correct file extensions (.zip for Windows)

**See also:** `WINDOWS_SUPPORT_VERIFICATION.md` for test details

## Adding New Scripts

When adding a new script:

1. Create a new subdirectory under `scripts/`
2. Place your script in `main.go` within that directory
3. Add a target to the Makefile (and build.ps1/build.bat for Windows)
4. Document it in this README

Example:

```bash
mkdir -p scripts/my_new_script
cat > scripts/my_new_script/main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("My new script!")
}
EOF
```

Then add to Makefile:

```makefile
my-script:
	go run scripts/my_new_script/main.go
```

## Why This Structure?

**Problem:** Having multiple Go programs with `main()` functions in the same directory causes conflicts when you try to `go run` multiple files or work with IDE tools.

**Solution:** Each script gets its own directory with a `main.go` file. This:

- ✅ Prevents `main()` redeclaration errors
- ✅ Keeps scripts organized
- ✅ Makes it clear what each script does
- ✅ Allows for script-specific dependencies in the future
- ✅ Works well with IDE tools and language servers

## Common Commands

```bash
# Generate embedded versions (before release)
make generate-embedded

# Test Windows compatibility
make test-windows

# Run both (recommended before PR)
make generate-embedded test-windows

# Direct execution if make not available
go run scripts/generate_embedded_versions/main.go
go run scripts/test_windows_compatibility/main.go
```
