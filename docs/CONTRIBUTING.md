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
make build              # Build goenv binary in root directory
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

| Feature      | Makefile (Unix) | build.ps1 (PowerShell) | build.bat (Batch) |
| ------------ | --------------- | ---------------------- | ----------------- |
| Build        | ✅              | ✅                     | ✅                |
| Test         | ✅              | ✅                     | ✅                |
| Clean        | ✅              | ✅                     | ✅                |
| Install      | ✅              | ✅                     | ❌ (use PS)       |
| Cross-build  | ✅              | ✅                     | ❌ (use PS)       |
| Version info | ✅              | ✅                     | ✅                |

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

### Testing Priorities

**📊 See [TESTING_ROADMAP.md](TESTING_ROADMAP.md) for current testing gaps and priorities.**

We especially welcome contributions in:

1. **High Priority:**
   - Integration tests for hooks system (lifecycle verification)
   - Cache migration tests (protect user data)
   - Error handling edge cases (file permissions, corrupted files)
   - CI interaction tests (prevent hangs in non-interactive mode)

2. **Medium Priority:**
   - VS Code integration E2E tests (setup/sync/revert flows)
   - Doctor command environment simulation (NFS, WSL, containers)
   - SBOM output verification tests

3. **Low Priority:**
   - Completions script validation
   - Custom command discovery tests

**Getting Started:**

1. Check the [Testing Roadmap](TESTING_ROADMAP.md) for specific test examples
2. Pick a high-priority gap that interests you
3. Open an issue to discuss your approach
4. Submit a PR with tests and any necessary refactoring

**Test Guidelines:**

- Follow the existing test structure in `*_test.go` files
- Include both positive and negative test cases
- Test edge cases and error conditions
- Add comments explaining complex test scenarios
- Ensure tests are deterministic (no flaky tests)
- Run `go test ./...` before submitting

## Coding Best Practices

### Use Helper Functions from `internal/utils`

goenv provides consolidated helper functions for common operations. Always use these instead of direct stdlib calls to maintain consistency and reduce boilerplate.

#### File Operations

**✅ Preferred - Use helpers:**

```go
// File existence checks
if utils.FileExists(path) { ... }           // Check if file exists (not directory)
if utils.DirExists(dir) { ... }             // Check if directory exists
if utils.PathExists(path) { ... }           // Check if path exists (file or dir)
if utils.FileNotExists(path) { ... }        // Inverted check (clearer than !FileExists)

// Executable checks (cross-platform)
if utils.IsExecutableFile(path) { ... }     // Checks existence + executable bit

// When you need FileInfo
info, exists, err := utils.StatWithExistence(path)
if !exists { ... }
```

**❌ Avoid - Direct os.Stat calls:**

```go
// Don't do this (unless you specifically need FileInfo for metadata)
if _, err := os.Stat(path); os.IsNotExist(err) { ... }
if info, err := os.Stat(path); err == nil && !info.IsDir() { ... }
```

**Exception:** Use `os.Stat` directly only when you need `FileInfo` for metadata (ModTime, Size, Mode).

```go
// OK - Need ModTime for cache validation
info, err := os.Stat(cachePath)
if err == nil {
    age := time.Since(info.ModTime())
    if age < ttl { ... }
}
```

#### Directory Creation

**✅ Preferred:**

```go
// With contextual error messages
if err := utils.EnsureDirWithContext(dir, "create cache directory"); err != nil {
    return err  // Error will be "failed to create cache directory: <underlying error>"
}

// Simple case
if err := utils.EnsureDir(dir); err != nil { ... }

// For file's parent directory
if err := utils.EnsureDirForFile(filePath); err != nil { ... }
```

**❌ Avoid:**

```go
// Manual error wrapping (redundant)
if err := os.MkdirAll(dir, 0755); err != nil {
    return errors.FailedTo("create cache directory", err)
}
```

#### JSON Operations

**✅ Preferred - Standard JSON:**

```go
// Reading JSON from file
var config Config
if err := utils.UnmarshalJSONFile(path, &config); err != nil { ... }

// Writing JSON to file
if err := utils.MarshalJSONFile(path, config); err != nil { ... }

// Pretty-printing JSON to string
jsonStr, err := utils.MarshalJSONPretty(data)

// Compact JSON to string
jsonStr, err := utils.MarshalJSONCompact(data)
```

**✅ OK - Special cases (document why):**

```go
// Example 1: JSONC (VS Code settings need comment stripping)
data, err := os.ReadFile(settingsPath)
if err != nil { ... }
data = jsonc.ToJSON(data)  // Strip comments
if err := json.Unmarshal(data, &settings); err != nil { ... }

// Example 2: Security-sensitive permissions
data, err := json.MarshalIndent(cache, "", "  ")
if err != nil { ... }
// Cache file needs 0600 permissions for security
if err := utils.WriteFileWithContext(path, data, 0600, "write cache"); err != nil { ... }
```

#### Command Execution

**✅ Preferred - Use helpers when possible:**

```go
// Simple execution
if err := utils.RunCommand("git", "add", "."); err != nil { ... }

// Get output
output, err := utils.RunCommandOutput("git", "rev-parse", "HEAD")

// Run in directory
err := utils.RunCommandInDir(dir, "git", "pull")

// With custom I/O
err := utils.RunCommandWithIO("git", []string{"pull"}, os.Stdout, os.Stderr)
```

**✅ OK - Direct exec.Command for special cases:**

```go
// Shell-specific checks (need "-c" flag)
cmd := exec.Command(shell, "-c", "declare -F goenv")
output, err := cmd.CombinedOutput()

// Context-aware execution
cmd := exec.CommandContext(ctx, "go", "build")
```

#### HTTP Operations

**✅ Preferred:**

```go
// Create clients with timeouts
client := utils.NewHTTPClient(30 * time.Second)
client := utils.NewHTTPClientDefault()  // 30s timeout
client := utils.NewHTTPClientForDownloads()  // 10min timeout

// Fetch and parse JSON
var result MyStruct
if err := utils.FetchJSON(ctx, url, &result); err != nil { ... }

// With timeout
if err := utils.FetchJSONWithTimeout(url, &result, 30*time.Second); err != nil { ... }
```

### Error Handling

**✅ Use consistent error wrapping:**

```go
// For operations that might fail
if err := someOperation(); err != nil {
    return errors.FailedTo("operation description", err)
}

// For validation errors
if !isValid {
    return errors.New("validation failed: reason")
}
```

**❌ Avoid bare errors:**

```go
// Don't do this - no context
if err != nil {
    return err
}
```

### Helper Function Reference

| Category | Location | Key Functions |
|----------|----------|---------------|
| **File operations** | `internal/utils/file.go` | FileExists, DirExists, PathExists, IsExecutableFile, StatWithExistence |
| **Directory creation** | `internal/utils/file.go` | EnsureDir, EnsureDirWithContext, EnsureDirForFile |
| **JSON** | `internal/utils/json.go` | UnmarshalJSONFile, MarshalJSONFile, MarshalJSONPretty |
| **Commands** | `internal/utils/command.go` | RunCommand, RunCommandOutput, RunCommandInDir, RunCommandWithIO |
| **HTTP** | `internal/utils/http.go` | NewHTTPClient, FetchJSON, FetchJSONWithTimeout |
| **Errors** | `internal/errors/errors.go` | FailedTo, New |

### When NOT to Use Helpers

Document your reasoning when you need to use direct stdlib calls:

```go
// OK - Need FileInfo for metadata
// We need the modification time for cache validation, so using os.Stat directly
info, err := os.Stat(cachePath)
if err != nil { ... }
age := time.Since(info.ModTime())

// OK - JSONC format requires comment stripping
// VS Code settings use JSONC format, need to strip comments before parsing
data = jsonc.ToJSON(data)
json.Unmarshal(data, &settings)

// OK - Shell-specific execution
// Need to run shell-specific command with -c flag
cmd := exec.Command(shell, "-c", "declare -F goenv")
```

### Code Review Checklist

When reviewing code, check for:

- [ ] File operations use helpers (FileExists, DirExists, etc.)
- [ ] Directory creation uses EnsureDirWithContext
- [ ] JSON operations use helpers (unless special case documented)
- [ ] Commands use helpers when appropriate
- [ ] HTTP clients have timeouts
- [ ] Errors are wrapped with context
- [ ] Special cases are documented with comments

## Documentation Contributions

Good documentation is just as important as good code! We welcome contributions to improve our documentation.

### Documentation Standards

All documentation should follow these quality standards:

✅ **Structure:**
- Include a table of contents for documents > 100 lines
- Use clear, hierarchical headings (H1 → H2 → H3)
- Group related content logically
- Add "See Also" cross-references at the end

✅ **Content:**
- Start with a brief purpose statement
- Include copy-paste working examples
- Cover common use cases first, advanced topics later
- Explain the "why" not just the "what"
- Include troubleshooting sections

✅ **Examples:**
- All code examples must be tested and work
- Show both success and failure cases
- Include platform-specific examples when relevant
- Use realistic, practical examples (not "foo/bar")

✅ **Cross-Platform:**
- Note platform differences (Linux, macOS, Windows)
- Include examples for all major platforms
- Mention any platform limitations

✅ **Security:**
- Highlight security considerations
- Show secure patterns first
- Warn about insecure alternatives
- Explain the security implications

### Documentation Types

We have several documentation patterns you can follow:

#### Quick Start Guides

**Example:** [HOOKS_QUICKSTART.md](./reference/HOOKS_QUICKSTART.md)

**Purpose:** Get users productive in 5 minutes

**Template:**
```markdown
# Feature Quick Start

Get started with [feature] in 5 minutes.

## What Is [Feature]?

Brief explanation (2-3 sentences).

## 5-Minute Setup

### 1. Step One
### 2. Step Two
### 3. Test It

## Common Use Cases

### Use Case 1
### Use Case 2

## Troubleshooting

## Next Steps
```

#### Reference Guides

**Example:** [COMMANDS.md](./reference/COMMANDS.md), [PLATFORM_SUPPORT.md](./reference/PLATFORM_SUPPORT.md)

**Purpose:** Complete, authoritative reference

**Template:**
```markdown
# Feature Reference

Complete reference for [feature].

## Table of Contents

## Overview

## Reference Tables

| Item | Description | Notes |
|------|-------------|-------|

## Detailed Sections

## See Also
```

#### How-To Guides

**Example:** [COMPLIANCE_USE_CASES.md](./advanced/COMPLIANCE_USE_CASES.md), [CACHE_TROUBLESHOOTING.md](./advanced/CACHE_TROUBLESHOOTING.md)

**Purpose:** Solve specific problems

**Template:**
```markdown
# How to [Task]

Complete guide to [task].

## Problem Statement

## Prerequisites

## Step-by-Step Solution

## Common Issues

## Best Practices

## See Also
```

#### Concept Guides

**Example:** [MODERN_COMMANDS.md](./user-guide/MODERN_COMMANDS.md), [SMART_CACHING.md](./advanced/SMART_CACHING.md)

**Purpose:** Explain concepts and design decisions

**Template:**
```markdown
# Understanding [Concept]

## Overview

## Why This Matters

## How It Works

## Trade-offs and Alternatives

## Best Practices
```

### Documentation Checklist

Use this checklist before submitting documentation PRs:

**📋 Use the checklist above to ensure high-quality documentation.**

Quick checklist:
- [ ] Table of contents included (for docs > 100 lines)
- [ ] All code examples tested and work
- [ ] Cross-references added to related docs
- [ ] Platform-specific notes where needed
- [ ] Security considerations highlighted
- [ ] Troubleshooting section included
- [ ] "See Also" section at end
- [ ] Markdown linter passes
- [ ] No broken links

### Where to Add Documentation

**Reference Documentation:**
- Commands: `docs/reference/COMMANDS.md`
- Environment variables: `docs/reference/ENVIRONMENT_VARIABLES.md`
- Platform support: `docs/reference/PLATFORM_SUPPORT.md`

**User Guides:**
- Getting started: `docs/user-guide/`
- Installation: `docs/user-guide/INSTALL.md`
- VS Code: `docs/user-guide/VSCODE_INTEGRATION.md`

**Advanced Topics:**
- Configuration: `docs/advanced/`
- Caching: `docs/advanced/SMART_CACHING.md`
- Cross-building: `docs/advanced/CROSS_BUILDING.md`

**Troubleshooting:**
- Cache issues: `docs/advanced/CACHE_TROUBLESHOOTING.md`
- Platform issues: `docs/reference/PLATFORM_SUPPORT.md`
- FAQ: `docs/FAQ.md`

**Compliance & Operations:**
- Hooks: `docs/reference/HOOKS_QUICKSTART.md` (quick) or `docs/reference/HOOKS.md` (complete)
- Compliance: `docs/advanced/COMPLIANCE_USE_CASES.md`
- CI/CD: `docs/advanced/CI_CD_GUIDE.md`

### Updating the Documentation Index

When adding new documentation:

1. **Add to main README.md:**
   ```markdown
   #### Advanced Topics
   - **[Your New Guide](./docs/YOUR_GUIDE.md)** - Brief description ⭐ **NEW**
   ```

2. **Add to docs/README.md:**
   ```markdown
   ### Your Category
   - **[Your New Guide](YOUR_GUIDE.md)** - Brief description
   ```

3. **Add cross-references:**
   - Link from related documents
   - Add to "See Also" sections

### Documentation Examples

Learn from our best documentation:

**Complete Guides:**
- [Hooks Quick Start](./reference/HOOKS_QUICKSTART.md) - Perfect quick start example
- [Compliance Use Cases](./advanced/COMPLIANCE_USE_CASES.md) - Comprehensive how-to
- [Platform Support Matrix](./reference/PLATFORM_SUPPORT.md) - Complete reference

**Well-Structured Commands:**
- [goenv vscode setup](./reference/COMMANDS.md#goenv-vscode-setup) - Clear, complete command documentation
- [goenv cache clean](./reference/COMMANDS.md#goenv-cache-clean) - Good examples and options

**Security Documentation:**
- [run_command action](./HOOKS.md#run_command) - Security-first approach
- [Hooks Security Model](./HOOKS.md#security-model) - Complete threat model

### Common Documentation Patterns

**Platform-specific examples:**
```markdown
### macOS / Linux

\```bash
export GOENV_ROOT="$HOME/.goenv"
\```

### Windows (PowerShell)

\```powershell
$env:GOENV_ROOT = "$HOME\.goenv"
\```
```

**Troubleshooting format:**
```markdown
### Issue: Brief Description

**Symptom:**
\```
Error message or behavior
\```

**Cause:** Explanation

**Solution:**
\```bash
command to fix
\```
```

**Example with explanation:**
```markdown
### Use Case: Descriptive Title

\```bash
# Step 1: Do this
command --flag

# Step 2: Do that
another command
\```

**What this does:**
1. First command explanation
2. Second command explanation

**Why this works:** Brief explanation
```

### Testing Documentation

Before submitting:

1. **Test all commands:**
   ```bash
   # Copy each command from your doc
   # Run it in a clean environment
   # Verify the output matches your documentation
   ```

2. **Check links:**
   ```bash
   # Use a markdown link checker
   markdown-link-check docs/YOUR_GUIDE.md
   ```

3. **Verify cross-references:**
   ```bash
   # Ensure all "See Also" links work
   # Check that related docs link back
   ```

4. **Spelling and grammar:**
   ```bash
   # Run through a spell checker
   # Use proper technical terminology
   ```

### Documentation Style Guide

**Voice and tone:**
- Use active voice: "Run this command" not "This command should be run"
- Be direct: "Do X" not "You should probably do X"
- Be helpful: Explain why, not just what

**Formatting:**
- Use **bold** for emphasis and important notes
- Use `code` for commands, files, and variables
- Use > blockquotes for tips and warnings
- Use ✅ ❌ ⚠️ for visual cues (sparingly)

**Terminology:**
- "Go version" not "golang version"
- "goenv" (lowercase) in text, `goenv` in code
- "macOS" not "Mac OS" or "OSX"
- "Windows" not "win" or "windows"
- Use proper product names: "VS Code" not "vscode"

**Code blocks:**
```markdown
\```bash
# Always include shebang or language
# Add comments to explain non-obvious parts
goenv install 1.25.2
\```
```

### Getting Help with Documentation

- **Ask questions:** Open a discussion on GitHub
- **Request review:** Tag maintainers in your PR
- **Use examples:** Reference existing good docs
- **Iterate:** Documentation improves through feedback

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
