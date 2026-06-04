# GitHub Copilot Instructions for goenv

## Branch Structure

### Active Branches

- **`main`** - Active development for v3.x (Go rewrite) - **USE THIS FOR ALL PRS**
- **`master`** - Legacy v2.x maintenance only (bash-based)

### PR Guidelines

✅ Target `main` for: features, bug fixes, docs, improvements
❌ Target `master` only for: critical v2.x security patches

### Branch Naming

- `fix/issue/NNN` - Bug fixes
- `feature/description` - New features
- `docs/description` - Documentation
- `refactor/description` - Code refactoring

## Project Architecture

### Tech Stack

- Language: Go (100% rewritten from bash in v3.x)
- CLI Framework: Cobra
- Testing: Go testing + some legacy BATS

### Directory Structure

#### cmd/ - CLI Commands (Cobra-based)

| Directory       | Purpose                                                    |
| --------------- | ---------------------------------------------------------- |
| `aliases/`      | Create, list, and manage Go version aliases                |
| `compliance/`   | Generate Software Bill of Materials (SBOM)                 |
| `core/`         | Core version management: install, use, list, info, compare |
| `diagnostics/`  | System diagnostics: doctor, status, cache management       |
| `hooks/`        | Manage declarative hooks configuration                     |
| `integrations/` | IDE integrations (VS Code setup, config)                   |
| `legacy/`       | Deprecated commands for backward compatibility             |
| `meta/`         | Metadata commands: help, update, explore                   |
| `shell/`        | Shell integration: init, setup, completions, prompt        |
| `shims/`        | Shim management: exec, rehash, which, whence               |
| `tools/`        | Go tool management across versions                         |
| `version/`      | Version file utilities: read/write, detect origin          |

#### internal/ - Internal Packages

| Directory      | Purpose                                                  |
| -------------- | -------------------------------------------------------- |
| `binarycheck/` | Analyze and validate binary compatibility                |
| `cache/`       | Manage Go build and module caches                        |
| `cgo/`         | Track CGO toolchain metadata                             |
| `cmdtest/`     | Test fixtures and command testing helpers                |
| `cmdutil/`     | Command utilities: context, prompts, helpers             |
| `completions/` | Embedded shell completions (bash, zsh, fish, PowerShell) |
| `config/`      | Configuration: paths, file locations, settings           |
| `envdetect/`   | Detect runtime environment (containers, WSL, Rosetta)    |
| `errors/`      | Custom error types and enhanced error messages           |
| `goenv/`       | ABI variable management for platform-specific builds     |
| `helptext/`    | Help text registry and formatting                        |
| `hooks/`       | Hook execution engine (webhooks, logging, commands)      |
| `install/`     | Version installation: download, extract, validate        |
| `lifecycle/`   | Go version lifecycle and EOL tracking                    |
| `manager/`     | Version discovery and management operations              |
| `migration/`   | v2→v3 migration utilities                                |
| `osinfo/`      | OS/architecture detection (zero-dependency foundation)   |
| `pathutil/`    | Path utilities: expansion, normalization                 |
| `platform/`    | Platform and environment detection                       |
| `resolver/`    | Binary resolution for version-specific execution         |
| `sbom/`        | SBOM generation, scanning, compliance                    |
| `session/`     | Session-scoped state memoization                         |
| `shellutil/`   | Shell detection and initialization                       |
| `shims/`       | **Cross-platform shim generation (Unix + Windows)**      |
| `tools/`       | Go tool management and versioning                        |
| `toolupdater/` | Automatic tool update functionality                      |
| `utils/`       | General utilities: file ops, binary detection            |
| `version/`     | Version fetching from official API                       |
| `vscode/`      | VS Code integration and config management                |
| `workflow/`    | Interactive workflow: setup wizard, discovery            |

#### Other Directories

- `docs/` - User documentation
- `testing/` - Test utilities
- `schemas/` - JSON schemas for validation
- `scripts/` - Build and utility scripts

## Platform-Specific Code

### Windows Batch File Constraints ⚠️ CRITICAL

**File**: `internal/shims/manager.go` (cross-platform shim generator)
**Function**: `createWindowsShim()` - generates `.bat` files for Windows
**Note**: This file also contains `createUnixShim()` for bash-based systems

Windows uses `.bat` batch files instead of bash scripts for shims.

**Batch File Constraints** (Reference: Issue #555):

❌ **NEVER DO**:

```bat
# BAD - Breaks CMD parsing!
if "%var%"=="value" (
    goto :label
)
:label

# BAD - Raw expansion in nested loop
for %%a in (%*) do (...)
```

✅ **ALWAYS DO**:

```bat
# GOOD - Use subroutines
if "%var%"=="value" call :subroutine
exit /b 0

:subroutine
# Logic here
exit /b 0
```

**Key Rules**:

1. NO goto/labels inside parenthesized `if (...)` blocks
2. NO raw `%*` expansion in `for` loops
3. Use single-line conditionals when possible: `if "%DEBUG%"=="1" echo on`
4. Always propagate exit codes: `exit /b %ERRORLEVEL%`
5. Use subroutines with `call :label` for complex logic

**Testing on macOS/Linux**:

- Go tests validate template logic ✅
- Cannot execute `.bat` files on Unix
- CI runs Windows tests automatically

## Key Implementation Patterns

### Shim System Flow

1. User runs `go version`
2. Hits shim at `~/.goenv/shims/go` (or `go.bat` on Windows)
3. Shim executes `goenv exec go version`
4. `goenv exec` resolves current version from:
   - `GOENV_VERSION` env var (highest priority)
   - `.go-version` file (walks up tree)
   - `go.mod` toolchain directive
   - `~/.goenv/version` (global default)
   - `system` (fallback)
5. Executes actual Go binary from resolved version

### Shim Generation

- Triggered by `goenv rehash`
- Auto-rehashes after `goenv install` and `go install`
- Scans all versions for binaries
- Creates platform-specific wrapper (bash or .bat)

### Version Resolution

- `internal/resolver/resolver.go` - Core logic
- Smart precedence handling
- Walks directory tree for `.go-version`
- Parses `go.mod` for toolchain directives

## Testing Standards

### Testing Philosophy

- **Test-Driven Development (TDD)**: Write failing tests first, then implement
- **Coverage Goal**: Aim for >80% coverage on critical paths (shims, resolver, manager)
- **Fast Tests**: Unit tests should run in milliseconds; avoid slow integration tests unless necessary
- **Platform Testing**: Validate cross-platform behavior (Unix/Windows) even if you can't execute on all platforms

### Test Organization

**File Naming**:

- `<name>_test.go` - Unit tests for `<name>.go`
- `<name>_integration_test.go` - Integration tests requiring external setup

**Test Function Naming**:

```go
func TestFunctionName(t *testing.T)           // Simple test
func TestFunctionName_Scenario(t *testing.T)   // Specific scenario
func TestFunctionName_ErrorCase(t *testing.T)  // Error handling
```

### Environment Requirements

⚠️ **CRITICAL**: Always unset `GOENV_DEBUG` before running tests:

```bash
unset GOENV_DEBUG && go test ./...
```

The Makefile does this automatically. Debug output interferes with test assertions.

### Available Test Targets

| Command                | Purpose                            | Use When                        |
| ---------------------- | ---------------------------------- | ------------------------------- |
| `make test`            | Standard test run (all tests)      | CI/CD, pre-commit               |
| `make test-quick`      | Clean output with gotestsum        | Development, quick feedback     |
| `make test-verbose`    | Full verbose output                | Debugging test failures         |
| `make test-report`     | Generate JUnit XML + HTML coverage | CI reporting, coverage analysis |
| `make test-debug`      | Show only failures                 | Finding broken tests quickly    |
| `make test-watch`      | Watch mode, rerun on changes       | Active development              |
| `make test-coverage`   | Quick coverage summary             | Coverage check                  |
| `make test-windows`    | Windows compatibility tests        | Before Windows-related changes  |
| `go test -v ./pkg/...` | Test specific package              | Focused testing                 |

### Test Output Locations

All test artifacts are written to `.test-results/`:

- `.test-results/full-output.log` - Complete test output
- `.test-results/failures.txt` - Failed tests summary
- `.test-results/coverage.html` - HTML coverage report
- `.test-results/junit.xml` - JUnit XML for CI integration
- `.test-results/test-output.json` - JSON test results

### Writing Tests

**Unit Test Pattern**:

```go
func TestShimGeneration(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid input", input: "test", want: "expected", wantErr: false},
        {name: "error case", input: "bad", want: "", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionUnderTest(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Integration Test Pattern**:

```go
// +build integration

func TestVersionInstallation(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // Test requires actual Go versions installed
}
```

### Platform-Specific Testing

**Windows Batch File Tests**:

- Go tests validate template generation logic ✅
- Cannot execute `.bat` files on macOS/Linux ❌
- Use `internal/cmdtest` helpers for command testing
- CI automatically runs Windows-specific tests
- Test batch syntax without execution: validate template output

**Cross-Platform Validation**:

```go
// Test both Unix and Windows shim generation
func TestShimGeneration_AllPlatforms(t *testing.T) {
    t.Run("unix", func(t *testing.T) {
        got := createUnixShim("go")
        if !strings.Contains(got, "#!/usr/bin/env bash") {
            t.Error("Unix shim missing shebang")
        }
    })

    t.Run("windows", func(t *testing.T) {
        got := createWindowsShim("go")
        if !strings.Contains(got, "@echo off") {
            t.Error("Windows shim missing @echo off")
        }
        // Validate batch file syntax rules
        if strings.Contains(got, "goto :label\n)") {
            t.Error("goto inside parenthesized block (Issue #555)")
        }
    })
}
```

### Test Requirements for PRs

Before submitting a PR:

1. ✅ All tests pass: `make test`
2. ✅ New code has test coverage
3. ✅ Integration tests pass (if applicable)
4. ✅ No race conditions: `go test -race ./...` (done by `make test`)
5. ✅ Windows compatibility validated (for shim/path/shell changes)

### Local Cross-Platform Testing (Optional)

**For developers who want to test against Windows/other platforms locally**, you can use [nektos/act](https://github.com/nektos/act) to run GitHub Actions workflows on your machine:

**Installation**:

```bash
# macOS
brew install act

# Linux
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Windows
choco install act-cli
```

**Usage**:

```bash
# Run all CI tests (includes Windows)
act

# Run specific job (e.g., Windows tests only)
act -j test-windows

# Run with specific platform
act --platform ubuntu-latest=catthehacker/ubuntu:act-latest

# Dry run to see what would execute
act --dryrun
```

**Note**:

- Windows containers require Docker Desktop with Windows containers enabled
- For Windows-specific tests, you may need larger runners: `act -P windows-latest=-self-hosted`
- CI will always run full cross-platform tests automatically when you push

**When to use local cross-platform testing**:

- ✅ Testing platform-specific code (Windows batch files, Unix shell scripts, path handling)
- ✅ Verifying OS-specific features (environment variables, file permissions, executables)
- ✅ Testing changes to shims, install scripts, or platform detection
- ✅ Debugging cross-platform CI failures locally
- ❌ Not needed for most changes (CI handles it automatically)

### Common Testing Commands

```bash
# Quick development workflow
make test-quick              # Fast feedback during development
make test-watch              # Auto-rerun on file changes

# Pre-commit checks
make test                    # Full test suite
make test-coverage           # Check coverage

# Debugging failed tests
make test-debug              # Show only failures
make test-verbose            # Full output for debugging

# Test specific areas
go test -v ./internal/shims/...                    # Package
go test -v -run TestShimGeneration ./internal/...  # Specific test
go test -v -short ./...                            # Skip slow tests

# CI/CD
make test-report             # Generate reports for CI
```

## Common Development Tasks

### Adding a New Command

1. Create in appropriate `cmd/<category>/` directory
2. Use Cobra framework pattern
3. Add to parent command's `init()`
4. Write tests in `<name>_test.go`
5. Update documentation

### Modifying Shim Templates

1. Edit `internal/shims/manager.go`
2. Update `createUnixShim()` OR `createWindowsShim()`
3. ⚠️ Windows changes require extra care - see constraints above
4. Test: `go test ./internal/shims/... ./cmd/shims/...`
5. Consider Windows batch file gotchas

### Adding Platform Support

1. Update `internal/utils/` for OS detection
2. Add platform-specific shim generator
3. Update install scripts
4. Test on target platform

## Code Style

### Conventional Commits

```
feat(area): description
fix(area): description
docs(area): description
refactor(area): description
test(area): description
```

### Error Handling

- Use `internal/errors` package
- Provide context: `errors.FailedTo("action", err)`
- User-friendly messages

### Configuration

- Respect environment variables (`GOENV_*`)
- Use `internal/config` for paths
- Support `GOENV_ROOT` customization

## Environment Variables

- `GOENV_ROOT` - Installation directory (default: `~/.goenv`)
- `GOENV_VERSION` - Override current version
- `GOENV_DEBUG` - Enable debug logging
- `GOENV_NO_AUTO_REHASH` - Disable auto-rehash (for CI/CD)

## Build & Release

```bash
make build         # Current platform
make test          # Run tests
make cross-build   # All platforms
make install       # Install to GOENV_ROOT
```

GitHub Actions builds release binaries for all platforms.

## Key Files to Know

- `internal/shims/manager.go` - Shim generation (cross-platform: Unix + Windows)
- `internal/resolver/resolver.go` - Version resolution
- `internal/manager/manager.go` - Version management
- `cmd/root.go` - Root command & global flags
- `main.go` - Entry point

## Documentation

- User docs: `docs/`
- Contributing: `CONTRIBUTING.md`
- Quick reference: `docs/QUICK_REFERENCE.md`
- Code should be self-documenting with clear comments

## Issue Resolution Checklist

When fixing issues:

1. ✅ Read issue thoroughly including comments
2. ✅ Identify affected files/components
3. ✅ Check for similar issues in issue tracker
4. ✅ Write failing test first (TDD)
5. ✅ Implement fix
6. ✅ Verify all tests pass
7. ✅ Update documentation if needed
8. ✅ Create PR against `main` branch
9. ✅ Reference issue number in commit: `Fixes #NNN`

## Windows-Specific Issue Patterns

Common Windows issues to watch for:

- Batch file syntax errors (goto/label problems)
- Path separators (use `filepath.Join`)
- Line endings (CRLF vs LF - see `.gitattributes`)
- Executable detection (file extensions)
- Permission model differences

## Need Help?

- Check existing tests for patterns
- Review similar commands for consistency
- Ask in PR for architectural guidance
- See `CONTRIBUTING.md` for full guidelines
