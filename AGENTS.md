# AI Agent Instructions for goenv

This document provides guidance for AI coding assistants working with the goenv codebase.

## 🎯 Quick Start for AI Agents

### Target Branch for PRs
- ✅ **`main`** - All v3.x development (features, fixes, docs)
- ❌ **`master`** - Only v2.x critical security patches

### Project Type
- **Language**: Go (100% rewrite from bash)
- **CLI Framework**: Cobra
- **Version**: v3.x (current), v2.x (legacy)

## 🏗️ Architecture Overview

### What is goenv?
Version manager for Go - like pyenv/rbenv. Allows multiple Go versions to coexist and switch between them per-project or globally.

### How It Works
1. **Shim System**: `~/.goenv/shims` added to front of PATH
2. **Command Interception**: When user runs `go`, hits shim wrapper
3. **Version Resolution**: Shim calls `goenv exec` to resolve current version
4. **Execution**: Real Go binary executed from selected version

### Version Selection (Precedence Order)
1. `GOENV_VERSION` environment variable
2. `.go-version` file (local, walks up directory tree)
3. `go.mod` toolchain directive (Go 1.21+)
4. `~/.goenv/version` (global default)
5. `system` (fallback to system Go)

## 📁 Directory Structure

### cmd/ - CLI Commands (Cobra-based)

| Directory | Purpose |
|-----------|---------|
| `aliases/` | Create, list, and manage Go version aliases |
| `compliance/` | Generate Software Bill of Materials (SBOM) |
| `core/` | Core version management: install, use, list, info, compare |
| `diagnostics/` | System diagnostics: doctor, status, cache management |
| `hooks/` | Manage declarative hooks configuration |
| `integrations/` | IDE integrations (VS Code setup, config) |
| `legacy/` | Deprecated commands for backward compatibility |
| `meta/` | Metadata commands: help, update, explore |
| `shell/` | Shell integration: init, setup, completions, prompt |
| `shims/` | Shim management: exec, rehash, which, whence |
| `tools/` | Go tool management across versions |
| `version/` | Version file utilities: read/write, detect origin |

### internal/ - Internal Packages

| Directory | Purpose |
|-----------|---------|
| `binarycheck/` | Analyze and validate binary compatibility |
| `cache/` | Manage Go build and module caches |
| `cgo/` | Track CGO toolchain metadata |
| `cmdtest/` | Test fixtures and command testing helpers |
| `cmdutil/` | Command utilities: context, prompts, helpers |
| `completions/` | Embedded shell completions (bash, zsh, fish, PowerShell) |
| `config/` | Configuration: paths, file locations, settings |
| `envdetect/` | Detect runtime environment (containers, WSL, Rosetta) |
| `errors/` | Custom error types and enhanced error messages |
| `goenv/` | ABI variable management for platform-specific builds |
| `helptext/` | Help text registry and formatting |
| `hooks/` | Hook execution engine (webhooks, logging, commands) |
| `install/` | Version installation: download, extract, validate |
| `lifecycle/` | Go version lifecycle and EOL tracking |
| `manager/` | Version discovery and management operations |
| `migration/` | v2→v3 migration utilities |
| `osinfo/` | OS/architecture detection (zero-dependency foundation) |
| `pathutil/` | Path utilities: expansion, normalization |
| `platform/` | Platform and environment detection |
| `resolver/` | Binary resolution for version-specific execution |
| `sbom/` | SBOM generation, scanning, compliance |
| `session/` | Session-scoped state memoization |
| `shellutil/` | Shell detection and initialization |
| `shims/` | **Cross-platform shim generation (Unix + Windows)** |
| `tools/` | Go tool management and versioning |
| `toolupdater/` | Automatic tool update functionality |
| `utils/` | General utilities: file ops, binary detection |
| `version/` | Version fetching from official API |
| `vscode/` | VS Code integration and config management |
| `workflow/` | Interactive workflow: setup wizard, discovery |

### Other Directories

- `docs/` - User documentation and guides
- `testing/` - Test utilities and fixtures
- `schemas/` - JSON schemas for validation
- `scripts/` - Build and utility scripts

## ⚠️ Windows Batch File Constraints - CRITICAL

**File**: `internal/shims/manager.go` (cross-platform shim generator)
**Function**: `createWindowsShim()` - Windows-specific batch file generation
**Note**: Same file contains `createUnixShim()` for bash on macOS/Linux

Windows uses `.bat` batch files instead of bash scripts for shims.

### Batch File Constraints (Reference: Issue #555)

**What Breaks CMD Parsing**:
```bat
# ❌ NEVER: goto/label inside parenthesized if block
if "%var%"=="value" (
    goto :label
)
:label

# ❌ NEVER: Raw %* expansion in nested for loop  
for %%a in (%*) do (...)
```

**What Works**:
```bat
# ✅ ALWAYS: Use subroutines with call
if "%var%"=="value" call :subroutine
exit /b 0

:subroutine
# Logic here
exit /b 0
```

### Windows Batch File Rules
1. ❌ NO `goto`/labels inside `if (...)` blocks
2. ❌ NO raw `%*` expansion in `for` loops
3. ✅ USE single-line conditionals: `if "%DEBUG%"=="1" echo on`
4. ✅ ALWAYS propagate exit codes: `exit /b %ERRORLEVEL%`
5. ✅ USE subroutines with `call :label` for complex logic

### Testing on macOS/Linux
- ✅ Go tests validate template generation logic
- ❌ Cannot execute `.bat` files on Unix
- ✅ CI runs actual Windows tests automatically

## 🧪 Testing Standards

### Core Principles
- ✅ **TDD**: Write failing test first, then implement
- ✅ **Coverage**: >80% on critical paths (shims, resolver, manager)
- ✅ **Fast**: Unit tests in milliseconds
- ✅ **Cross-platform**: Validate Unix + Windows behavior

### Environment Setup

⚠️ **CRITICAL**: Unset `GOENV_DEBUG` before running tests (Makefile does this automatically):
```bash
unset GOENV_DEBUG && go test ./...
```

### Test Targets Reference

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `make test` | All tests | CI/CD, pre-commit |
| `make test-quick` | Clean output, fast feedback | Active development |
| `make test-verbose` | Full verbose output | Debugging failures |
| `make test-report` | JUnit + HTML coverage | CI reporting |
| `make test-debug` | Failures only | Quick error finding |
| `make test-watch` | Auto-rerun on changes | Development loop |
| `make test-coverage` | Coverage summary | Coverage check |
| `make test-windows` | Windows compatibility | Platform-specific changes |

### Test Organization

**Files**: `<name>_test.go` (unit), `<name>_integration_test.go` (integration)

**Functions**: 
```go
func TestFunction(t *testing.T)              // Simple
func TestFunction_Scenario(t *testing.T)     // Specific case
func TestFunction_ErrorCase(t *testing.T)    // Error handling
```

### Platform Testing Patterns

**Windows Batch Validation**:
```go
func TestWindowsShim(t *testing.T) {
    bat := createWindowsShim("go")
    
    // Validate batch syntax (Issue #555)
    if strings.Contains(bat, "goto :label\n)") {
        t.Error("goto inside parenthesized block")
    }
    
    // Check required elements
    if !strings.Contains(bat, "@echo off") {
        t.Error("missing @echo off")
    }
}
```

**Cross-Platform Tests**:
```go
func TestShims_AllPlatforms(t *testing.T) {
    t.Run("unix", func(t *testing.T) { /* test createUnixShim */ })
    t.Run("windows", func(t *testing.T) { /* test createWindowsShim */ })
}
```

### Test Artifacts

Output written to `.test-results/`:
- `full-output.log` - Complete output
- `failures.txt` - Failed tests summary
- `coverage.html` - Coverage report
- `junit.xml` - CI integration

### PR Checklist

Before submitting:
1. ✅ `make test` passes
2. ✅ New code has tests
3. ✅ No race conditions (`-race` flag used)
4. ✅ Windows compatibility validated (if applicable)
5. ✅ Coverage maintained or improved

### Quick Commands

```bash
# Development
make test-quick              # Fast feedback
make test-watch              # Auto-rerun

# Pre-commit
make test                    # Full suite
make test-coverage           # Check coverage

# Debugging
make test-debug              # Failures only
make test-verbose            # Full details

# Focused testing
go test -v ./internal/shims/...                   # Package
go test -v -run TestName ./...                    # Specific test
go test -v -short ./...                           # Skip slow tests
```

## 🛠️ Common Development Tasks

### Adding a New Command
1. Create file in `cmd/<category>/<command>.go`
2. Implement using Cobra framework
3. Register in parent command's `init()`
4. Write tests in `<command>_test.go`
5. Update documentation

### Modifying Shim Templates
1. Edit `internal/shims/manager.go` (handles all platforms)
2. Update the appropriate function:
   - `createUnixShim()` for macOS/Linux (bash scripts)
   - `createWindowsShim()` for Windows (batch files)
3. ⚠️ **Windows changes**: Review batch file constraints above
4. Run tests: `go test ./internal/shims/... ./cmd/shims/...`
5. Verify template syntax

### Adding Platform Support
1. Update `internal/utils/` for OS detection
2. Add platform-specific shim generator
3. Update install scripts (`install.sh`, `install.ps1`)
4. Test on target platform

## 📝 Code Conventions

### Commit Messages (Conventional Commits)
```
feat(area): add new feature
fix(area): resolve bug
docs(area): update documentation
refactor(area): restructure code
test(area): add tests
```

### Error Handling
```go
// Use internal/errors package
return errors.FailedTo("action", err)

// Provide user-friendly context
return fmt.Errorf("failed to %s: %w", action, err)
```

### Configuration
- Respect `GOENV_*` environment variables
- Use `internal/config` for path management
- Support `GOENV_ROOT` customization

## 🔑 Key Files

| File | Purpose | Notes |
|------|---------|-------|
| `internal/shims/manager.go` | Shim generation | Cross-platform (Unix + Windows) |
| `internal/resolver/resolver.go` | Version resolution | Smart precedence logic |
| `internal/manager/manager.go` | Version management | Install/uninstall |
| `cmd/root.go` | Root command | Global flags |
| `main.go` | Entry point | Binary entry |

## 🐛 Issue Resolution Checklist

When fixing issues:

1. ✅ Read issue and all comments thoroughly
2. ✅ Identify affected files/components
3. ✅ Check for similar/related issues
4. ✅ Write failing test first (TDD)
5. ✅ Implement fix
6. ✅ Verify all tests pass
7. ✅ Update documentation if needed
8. ✅ Create PR against `main` branch
9. ✅ Reference issue: `Fixes #NNN` in commit

## 🪟 Common Windows Issues

Watch out for:
- Batch file syntax errors (goto/label problems)
- Path separators (use `filepath.Join`)
- Line endings (CRLF vs LF - see `.gitattributes`)
- Executable detection (need `.exe`, `.bat` extensions)
- Permission model differences

## 🌍 Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOENV_ROOT` | `~/.goenv` | Installation directory |
| `GOENV_VERSION` | - | Override current version |
| `GOENV_DEBUG` | - | Enable debug logging |
| `GOENV_NO_AUTO_REHASH` | `0` | Disable auto-rehash |

## 🚀 Build & Release

```bash
make build         # Build for current platform
make test          # Run all tests
make cross-build   # Build for all platforms
make install       # Install to GOENV_ROOT
```

GitHub Actions automatically builds release binaries for all platforms.

## 📚 Documentation Locations

- `docs/` - User guides and reference
- `CONTRIBUTING.md` - Development guidelines
- `docs/QUICK_REFERENCE.md` - Command reference
- `.github/copilot-instructions.md` - Detailed AI guidance
- Code comments - Implementation details

## 💡 Pro Tips for AI Agents

1. **Branch Awareness**: Always target `main` for PRs (not `master`)
2. **Windows Testing**: Can't execute `.bat` on Unix - rely on Go tests
3. **Batch File Edits**: Extra careful with Windows shim template changes
4. **Test Coverage**: Run relevant tests before committing
5. **Documentation**: Update docs when changing behavior
6. **Conventional Commits**: Use semantic commit messages
7. **Issue References**: Link commits to issues with `Fixes #NNN`

## 🆘 Getting Help

- Check existing tests for patterns
- Review similar commands for consistency
- Read `CONTRIBUTING.md` for detailed guidelines
- Ask in PR comments for architectural guidance
- Reference this file and `.github/copilot-instructions.md`

---

**Last Updated**: 2026-06-02 (Issue #555 resolution)
