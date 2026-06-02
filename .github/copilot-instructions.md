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
```
cmd/              # CLI commands (Cobra-based)
├── core/        # Version management (install, use, list, info)
├── shims/       # Shim system (exec, rehash, which, whence)
├── tools/       # Tool management (install, sync, outdated)
├── shell/       # Shell integration (init, setup, prompt)
├── diagnostics/ # Health checks (doctor, status)
├── meta/        # Utilities (help, update, commands)
├── integrations/# IDE integration (vscode)
└── aliases/     # Version aliases

internal/         # Internal packages
├── shims/       # Shim generation logic ⚠️ WINDOWS SENSITIVE
├── manager/     # Version management
├── resolver/    # Binary/version resolution
├── install/     # Version installation
└── utils/       # Shared utilities

docs/            # User documentation
testing/         # Test utilities
```

## Platform-Specific Code

### Windows Development ⚠️ CRITICAL

**Location**: `internal/shims/manager.go`

Windows uses `.bat` files instead of bash scripts for shims.

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

## Testing Guidelines

```bash
# Run all tests
make test

# Test specific area
go test -v ./cmd/shims/...
go test -v ./internal/shims/...

# Integration tests (require installed Go versions)
go test -v ./cmd/shims/exec_integration_test.go

# Build & test locally
make build
./goenv <command>
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

- `internal/shims/manager.go` - Shim generation (Windows-sensitive!)
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
