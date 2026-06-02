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

```
cmd/                    # CLI commands (Cobra-based)
├── core/              # install, use, list, info, compare
├── shims/             # exec, rehash, which, whence ⚠️
├── tools/             # tool management across versions
├── shell/             # init, setup, prompt, completion
├── diagnostics/       # doctor, status
├── meta/              # help, update, commands
├── integrations/      # vscode integration
└── aliases/           # version alias management

internal/              # Internal packages
├── shims/            # ⚠️ WINDOWS SENSITIVE - see below
├── manager/          # Version management logic
├── resolver/         # Binary/version resolution
├── install/          # Version installation
├── config/           # Configuration management
├── errors/           # Error handling
└── utils/            # Shared utilities
```

## ⚠️ Windows Development - CRITICAL

### Location: `internal/shims/manager.go`

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

## 🧪 Testing Patterns

```bash
# Run all tests
make test

# Test specific packages
go test -v ./cmd/shims/...
go test -v ./internal/shims/...

# Integration tests (require Go versions installed)
go test -v ./cmd/shims/exec_integration_test.go

# Build and test locally
make build
./goenv <command>
```

## 🛠️ Common Development Tasks

### Adding a New Command
1. Create file in `cmd/<category>/<command>.go`
2. Implement using Cobra framework
3. Register in parent command's `init()`
4. Write tests in `<command>_test.go`
5. Update documentation

### Modifying Shim Templates
1. Edit `internal/shims/manager.go`
2. Update `createUnixShim()` OR `createWindowsShim()`
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
| `internal/shims/manager.go` | Shim generation | ⚠️ Windows-sensitive |
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
