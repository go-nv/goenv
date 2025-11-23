# VS Code Integration - Quick Reference

## 🚀 Quick Start

```bash
# Initialize VS Code workspace (auto-detects go.work)
goenv vscode init

# Set version and configure VS Code in one command
goenv local 1.24.4 --vscode

# After changing versions, re-sync settings
goenv vscode sync
```

## 📋 Commands

### `goenv vscode init`

Initialize VS Code workspace with goenv configuration.

**Flags:**

- `--template <name>` - Choose template: `basic`, `advanced`, `monorepo` (default: auto-detect)
- `--env-vars` - Use environment variables instead of absolute paths
- `--dry-run` - Preview changes without writing
- `--diff` - Show diff of changes (implies --dry-run)
- `--force` - Overwrite existing settings

**Examples:**

```bash
goenv vscode init                           # Auto-detect, use absolute paths
goenv vscode init --template advanced       # Use advanced template
goenv vscode init --dry-run                 # Preview changes
goenv vscode init --env-vars                # Use ${env:GOROOT} mode
```

### `goenv vscode sync`

Sync VS Code settings with current Go version.

**Flags:**

- `--dry-run` - Preview changes
- `--diff` - Show what would change

**Examples:**

```bash
goenv local 1.24.4 && goenv vscode sync    # Change version and sync
goenv vscode sync --dry-run                # Preview sync
```

### `goenv vscode status`

Check VS Code integration status.

**Flags:**

- `--json` - Machine-readable output for CI

**Examples:**

```bash
goenv vscode status                        # Human-readable status
goenv vscode status --json                 # JSON for automation
```

**Exit codes:**

- `0` - Settings are in sync
- `1` - Version mismatch or error

### `goenv vscode revert`

Restore VS Code settings from backup.

```bash
goenv vscode revert                        # Restore last backup
```

### `goenv vscode doctor`

Run health checks on VS Code integration and Go tooling.

**Flags:**

- `--json` - Machine-readable output for CI

**Examples:**

```bash
goenv vscode doctor                        # Human-readable diagnostics
goenv vscode doctor --json                 # JSON for automation
```

**Checks performed:**

1. VS Code settings configuration
2. Go toolchain availability
3. gopls installation and version
4. go.toolsGopath writability
5. Workspace structure (go.work, go.mod)
6. Settings sync status
7. Extension recommendations
8. Common configuration issues

**Exit codes:**

- `0` - All checks passed or warnings only
- `1` - One or more critical checks failed

**See also:** [VSCODE_DOCTOR_REFERENCE.md](../../VSCODE_DOCTOR_REFERENCE.md) for detailed documentation.

## 📐 Templates

### Basic (Default)

Minimal configuration - just the essentials.

**Use when:**

- Simple projects
- Want minimal VS Code configuration
- Prefer to configure manually

**Settings:**

- `go.goroot`, `go.gopath`, `go.toolsGopath`

### Advanced

Opinionated setup with modern best practices.

**Use when:**

- Want a fully-featured setup
- Prefer gofumpt formatting
- Use test explorer
- Need linting with golangci-lint

**Settings:**

- All basic settings
- gofumpt formatting
- Test Explorer enabled
- golangci-lint integration
- Enhanced gopls settings
- Coverage on single test

### Monorepo

Optimized for workspaces with go.work files.

**Auto-detected when:**

- `go.work` file exists in workspace root

**Use when:**

- Multi-module workspace
- Need directory filters
- Flat test package display

**Settings:**

- All advanced features
- Directory filters (-vendor, -node_modules, -third_party)
- Flat test package display
- GOPATH inference disabled

## 🔄 Workflows

### New Project Setup

```bash
cd myproject
goenv local 1.24.4 --vscode
code .
```

### Switching Versions

```bash
# Absolute paths mode (default)
goenv local 1.23.2
goenv vscode sync
# Reload VS Code window (Cmd+Shift+P → Reload Window)

# Environment variables mode
goenv local 1.23.2
# Close VS Code, reopen from terminal
code .
```

### Monorepo Setup

```bash
cd mymonorepo
# Create go.work if needed
go work init ./module1 ./module2

# Auto-detects and uses monorepo template
goenv vscode init

code .
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Setup goenv
  run: |
    goenv local 1.24.4
    goenv vscode status --json > vscode-status.json

- name: Verify VS Code config
  run: |
    goenv vscode status --json
```

### Preview Changes

```bash
# See what would change without modifying files
goenv vscode init --dry-run --template advanced

# Show diff
goenv vscode sync --diff
```

### Rollback Changes

```bash
# Made a mistake? Restore from backup
goenv vscode revert

# Backups are created automatically at:
# .vscode/settings.json.bak
```

## 🎯 Configuration Modes

### Absolute Paths (Default, Recommended)

✅ Works when VS Code opened from GUI  
✅ No shell configuration required  
✅ Explicit version paths  
⚠️ Need to re-sync when changing versions

**Use when:**

- Opening VS Code from Finder/Explorer
- Team members with different shells
- Want it to "just work"

```bash
goenv vscode init                    # Uses absolute paths
```

### Environment Variables

✅ Automatically tracks version changes  
✅ No re-sync needed  
⚠️ Must launch VS Code from terminal  
⚠️ Requires shell integration

**Use when:**

- Always launch from terminal
- Want auto-tracking of version changes
- Have complex shell setup

```bash
goenv vscode init --env-vars
# Then: close VS Code, open from terminal
code .
```

## 💾 Backups

All commands that modify settings automatically create backups:

- Location: `.vscode/settings.json.bak`
- Created before every modification
- Restore with: `goenv vscode revert`

## 🔍 Troubleshooting

### VS Code not using correct Go version

```bash
# Check status
goenv vscode status

# If mismatch, sync
goenv vscode sync

# Reload VS Code window
```

### gopls not working

```bash
# Check what's configured
goenv vscode status --json

# Verify tools are installed
go install golang.org/x/tools/gopls@latest

# Check gopls version
gopls version
```

### Environment variables not working

```bash
# Must launch from terminal
code .

# Verify environment
echo $GOROOT
echo $GOPATH

# Ensure shell integration is loaded
eval "$(goenv init -)"
```

### Want to start over

```bash
# Remove VS Code settings
rm .vscode/settings.json

# Re-initialize
goenv vscode init
```

## 📊 Status Output

### Human-Readable

```
VS Code Integration Status
==========================

✓ VS Code settings found

Mode: Absolute Paths
  Configured version: 1.24.4
  Expected version: 1.24.4 (from .go-version)

✓ Settings are in sync
```

### JSON (for CI)

```json
{
  "hasSettings": true,
  "usesEnvVars": false,
  "configuredVersion": "1.24.4",
  "expectedVersion": "1.24.4",
  "mismatch": false,
  "settingsPath": "/path/to/.vscode/settings.json",
  "versionSource": ".go-version"
}
```

## 🎨 Example Configurations

For complete configuration examples, see the VS Code Integration guide in [VSCODE_INTEGRATION.md](../user-guide/VSCODE_INTEGRATION.md).

## 🆘 Getting Help

```bash
goenv vscode --help
goenv vscode init --help
goenv vscode sync --help
goenv vscode status --help
```

---

**Pro Tip:** Use `goenv local <version> --vscode` to set version and configure VS Code in one command!
