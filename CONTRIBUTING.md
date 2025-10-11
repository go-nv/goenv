# Contributing

The goenv source code is [hosted on GitHub](https://github.com/go-nv/goenv).
It's clean, modular, and easy to understand - now written in Go for better cross-platform support!

Please feel free to submit pull requests and file bugs on the [issue tracker](https://github.com/go-nv/goenv/issues).

## Prerequisites

### Required
- **Go 1.21+** - For building and testing goenv
- **Git** - For cloning and contributing

### Platform-Specific
- **Linux/macOS**: Any shell (`bash`, `zsh`, `fish`, etc.)
- **Windows**: PowerShell 5.1+ or Command Prompt (cmd.exe)

## Development Setup

### Unix/Linux/macOS

1. **Clone the repository**:
   ```bash
   git clone https://github.com/go-nv/goenv.git
   cd goenv
   ```

2. **Build**:
   ```bash
   make build
   ```

3. **Run tests**:
   ```bash
   make test
   ```

### Windows

goenv provides **three ways** to build on Windows:

#### Option 1: PowerShell (Recommended)
```powershell
# Build
.\build.ps1 build

# Run tests
.\build.ps1 test

# See all options
.\build.ps1 help
```

#### Option 2: Batch Script (cmd.exe)
```batch
REM Build
build.bat build

REM Run tests
build.bat test

REM See all options
build.bat help
```

#### Option 3: Make (if installed via WSL, Cygwin, or MSYS2)
```bash
make build
make test
```

#### Option 4: Visual Studio Code
Press `Ctrl+Shift+B` to run build tasks, or use the Command Palette (`Ctrl+Shift+P`) and search for "Run Task".

## Common Commands

### Building

**Unix/macOS:**
```bash
make build              # Build goenv binary
make bin/goenv          # Build and create bin/ directory
```

**Windows (PowerShell):**
```powershell
.\build.ps1 build       # Build goenv.exe
```

**Windows (Batch):**
```batch
build.bat build         # Build goenv.exe
```

### Testing

**Unix/macOS:**
```bash
make test               # Run all Go tests
```

**Windows (PowerShell):**
```powershell
.\build.ps1 test        # Run all Go tests
```

**Windows (Batch):**
```batch
build.bat test          # Run all Go tests
```

### Cleaning

**Unix/macOS:**
```bash
make clean              # Remove built binaries
```

**Windows (PowerShell):**
```powershell
.\build.ps1 clean       # Remove built binaries
```

**Windows (Batch):**
```batch
build.bat clean         # Remove built binaries
```

### Installing Locally

**Unix/macOS:**
```bash
make install            # Install to /usr/local (default)
PREFIX=$HOME/.local make install  # Custom location
```

**Windows (PowerShell):**
```powershell
.\build.ps1 install     # Install to %LOCALAPPDATA%\goenv
$env:PREFIX = "C:\tools\goenv"; .\build.ps1 install  # Custom location
```

### Cross-Compilation

**Unix/macOS:**
```bash
make cross-build        # Build for all platforms
```

**Windows (PowerShell):**
```powershell
.\build.ps1 cross-build # Build for all platforms
```

## Build Scripts Reference

| Feature | Makefile (Unix) | build.ps1 (PowerShell) | build.bat (Batch) |
|---------|----------------|------------------------|-------------------|
| Build | ✅ | ✅ | ✅ |
| Test | ✅ | ✅ | ✅ |
| Clean | ✅ | ✅ | ✅ |
| Install | ✅ | ✅ | ❌ (use PS) |
| Cross-build | ✅ | ✅ | ❌ (use PS) |
| Version info | ✅ | ✅ | ✅ |

## IDE Support

### Visual Studio Code

Tasks are pre-configured in `.vscode/tasks.json`:

- **Build goenv** - `Ctrl+Shift+B` (default build task)
- **Test goenv** - `Ctrl+Shift+P` → "Run Test Task"
- **Clean**, **Install**, **Cross-build** - Available in "Run Task" menu

Tasks automatically detect your OS and use the appropriate build method.

### GoLand / IntelliJ IDEA

1. Open the project
2. Go Run/Debug Configurations
3. Add a "Go Build" configuration pointing to the project root
4. Add test configurations as needed

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./cmd/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestExec ./cmd/
```

### Test Coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Workflows

### Submitting an issue

1. Check existing issues and verify that your issue is not already submitted.
   If it is, it's highly recommended to add to that issue with your reports.
2. Open issue
3. Be as detailed as possible - Linux distribution, shell, what did you do,
   what did you expect to happen, what actually happened.

### Submitting a PR

1. Find an existing issue to work on or follow `Submitting an issue` to create one
   that you're also going to fix.
   Make sure to notify that you're working on a fix for the issue you picked.
1. Branch out from latest `master`.
1. Code, add, commit and push your changes in your branch.
1. Make sure that tests (or let the CI do the heavy work for you).
1. Submit a PR.
1. Make sure to clarify if the PR is ready for review or work-in-progress.
   A simple `[WIP]` (in any form) is a good indicator whether the PR is still being actively developed.
1. Collaborate with the codeowners/reviewers to merge this in `master`.

### Release process

Described in details at [RELEASE_PROCESS](./RELEASE_PROCESS.md).
