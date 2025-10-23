# VS Code Integration Guide

This guide shows how to use goenv seamlessly with Visual Studio Code and the official Go extension.

## Table of Contents

- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Setup Methods](#setup-methods)
- [Project Configuration](#project-configuration)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

### Option 1: Automatic Setup (Easiest - NEW! ⚡)

The fastest way to set up VS Code integration:

```bash
# Navigate to your project
cd ~/projects/myapp

# Set project Go version AND configure VS Code in one command
goenv use 1.22.0 --vscode
```

This automatically:

- Creates `.go-version` file
- Generates `.vscode/settings.json` with goenv configuration
- Creates `.vscode/extensions.json` (recommends Go extension)
- Makes VS Code ready to use with goenv

### Option 2: Manual Setup with One Command (NEW! ⚡)

Already have a project? Just run:

```bash
cd ~/projects/myapp
goenv vscode init
```

This generates the necessary VS Code configuration files.

### Option 3: Launch from Terminal (Traditional)

The easiest way to ensure VS Code picks up your goenv environment:

```bash
# Navigate to your project
cd ~/projects/myapp

# Set project Go version
goenv use 1.22.0

# Launch VS Code from terminal
code .
```

VS Code will inherit all goenv environment variables (`GOROOT`, `GOPATH`, `GOENV_VERSION`).

### Option 4: Advanced Setup with Templates (NEW! ⚡)

Choose from different configuration templates:

```bash
# Basic template (recommended for most projects)
goenv vscode init

# Advanced template (includes gopls settings, format on save)
goenv vscode init --template advanced

# Monorepo template (for large repositories)
goenv vscode init --template monorepo

# Force overwrite existing settings
goenv vscode init --force
```

**Manual Configuration** (if you prefer):

Add `.vscode/settings.json` to your project for consistent behavior:

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

## How It Works

### goenv Environment Variables

When you run `goenv init`, it adds a script to your shell that exports:

```bash
export GOROOT="/Users/you/.goenv/versions/1.22.0/go"
export GOPATH="/Users/you/go/1.22.0"
```

### VS Code Go Extension

The [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) looks for Go in this order:

1. **`go.goroot` setting** in workspace/user settings
2. **`GOROOT` environment variable**
3. **`go` on PATH**

### When VS Code Inherits Variables

✅ **VS Code WILL inherit goenv variables when:**

- Launched from a terminal: `code .` or `code /path/to/project`
- Integrated terminal runs shell init scripts
- Shell profile (`.bashrc`, `.zshrc`) has goenv initialization

❌ **VS Code WON'T inherit goenv variables when:**

- Opened via Finder/Spotlight/Dock on macOS
- Launched via Windows Start Menu or desktop shortcut
- Opened before shell initialization completes

## Automated Setup (NEW!)

goenv now includes **built-in VS Code integration commands** that automatically configure your workspace:

### `goenv vscode init` Command

Automatically generates VS Code configuration files:

```bash
cd ~/projects/myapp
goenv vscode init

# Output:
# ✓ Created/updated .vscode/settings.json
# ✓ Created/updated .vscode/extensions.json
# ✨ VS Code workspace configured for goenv!
```

**What it does:**

- Creates `.vscode/settings.json` with goenv environment variable references
- Creates `.vscode/extensions.json` (recommends Go extension)
- Merges with existing settings (won't overwrite your customizations)
- Supports multiple templates (basic, advanced, monorepo)

**Templates available:**

| Template   | Description                                      | Use Case                 |
| ---------- | ------------------------------------------------ | ------------------------ |
| `basic`    | Go configuration with goenv env vars             | Most projects (default)  |
| `advanced` | Includes gopls, format on save, organize imports | Professional development |
| `monorepo` | Configured for large repositories                | Multi-module projects    |

### `goenv use --vscode` Flag

Set Go version AND configure VS Code in one command:

```bash
goenv use 1.22.0 --vscode

# Output:
# Initializing VS Code workspace...
# ✓ Created/updated .vscode/settings.json
# ✓ Created/updated .vscode/extensions.json
# ✨ VS Code workspace configured for goenv!
```

**Benefits:**

- Zero friction onboarding
- One command does everything
- Perfect for starting new projects
- Safe (won't fail if VS Code setup fails)

### `goenv doctor` VS Code Check

The doctor command now checks VS Code integration:

```bash
goenv doctor

# Output includes:
# ✅ VS Code integration
#    VS Code configured to use goenv environment variables
#
# OR:
#
# ⚠️  VS Code integration
#    Found .vscode directory but no settings.json
#    💡 Run 'goenv vscode init' to configure Go extension
```

**What it checks:**

- Presence of `.vscode` directory
- Existence of `settings.json`
- Go configuration in settings
- Whether it's using goenv environment variables
- Provides actionable advice if misconfigured

## Setup Methods

### Method 1: Terminal Launch (Simplest)

**Pros:**

- No configuration needed
- Works immediately
- Automatically picks up `goenv use` changes on restart

**Cons:**

- Must remember to launch from terminal
- GUI launches won't work

**Setup:**

```bash
# Ensure goenv is initialized in your shell
# Add to ~/.bashrc, ~/.zshrc, or equivalent:
eval "$(goenv init -)"

# Launch VS Code from terminal
cd ~/projects/myapp
code .
```

**Verify it works:**

1. Open the integrated terminal in VS Code
2. Run `go version` and `echo $GOROOT`
3. Should show the goenv-managed version

### Method 2: Workspace Settings (Most Reliable)

**Pros:**

- Works regardless of how VS Code is launched
- Portable across team members
- Version controlled with your project

**Cons:**

- Requires configuration per project
- Need to update if changing goenv location

**Setup:**

Create `.vscode/settings.json` in your project:

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools",

  // Optional: Show current Go version in status bar
  "go.toolsManagement.autoUpdate": true,

  // Optional: Use goenv's version for go commands
  "go.alternateTools": {
    "go": "${env:GOROOT}/bin/go"
  }
}
```

**For hardcoded paths** (works even without goenv in shell):

```json
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.22.0/go",
  "go.gopath": "${env:HOME}/go/1.22.0"
}
```

**Add to version control:**

```bash
git add .vscode/settings.json
git commit -m "Add VS Code Go configuration"
```

### Method 3: User Settings (Global Default)

**Pros:**

- Applies to all projects
- Set once, forget it

**Cons:**

- Doesn't respect per-project `goenv use` versions
- Must manually update when changing global version

**Setup:**

1. Open VS Code Settings (`Cmd+,` or `Ctrl+,`)
2. Search for "go.goroot"
3. Edit `settings.json` (user scope):

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}"
}
```

### Method 4: Combination Approach (Best of Both Worlds)

Use both terminal launch AND workspace settings:

1. **Global settings** (user): Use environment variables
2. **Project settings** (workspace): Override for specific projects
3. **Always launch from terminal** for automatic updates

This gives you:

- Automatic goenv integration
- Per-project overrides when needed
- Consistent behavior across launch methods

## Project Configuration

### Standard Project Setup

For a project using goenv:

```bash
# 1. Set project Go version
cd ~/projects/myapp
goenv use 1.22.0

# 2. Create VS Code settings
mkdir -p .vscode
cat > .vscode/settings.json << 'EOF'
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
EOF

# 3. Add to git
git add .go-version .vscode/settings.json

# 4. Launch VS Code
code .
```

### Multi-Module Workspace

For projects with multiple Go modules:

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.useLanguageServer": true,
  "gopls": {
    "build.directoryFilters": ["-node_modules", "-vendor"]
  }
}
```

### Monorepo Setup

For large repositories with multiple Go versions:

```json
{
  "go.goroot": "${workspaceFolder}/backend/.goenv/versions/1.22.0/go",
  "go.gopath": "${workspaceFolder}/backend/go",
  "go.inferGopath": false
}
```

## Troubleshooting

### VS Code Shows Wrong Go Version After Changing Versions

**Symptom:** After running `goenv use 1.23.2 --vscode`, VS Code still shows the old Go version (e.g., 1.24.4).

**Root Cause:** VS Code inherits environment variables (`GOROOT`, `GOPATH`) from the shell that launched it. Simply running `goenv use` updates the `.go-version` file but doesn't update your current shell's environment. Reloading the VS Code window does NOT refresh environment variables.

**Solution:**

**You must restart VS Code from terminal:**

```bash
# 1. Close VS Code completely (Cmd+Q on macOS, not just close window)

# 2. Relaunch from terminal
cd ~/your/project
code .
```

**Why this works:** A new terminal session loads your shell configuration (`~/.bashrc`, `~/.zshrc`), which runs `goenv init` and sets the correct `GOROOT` environment variable. VS Code then inherits these fresh variables.

**Quick verification:**

```bash
# Check if environment needs refresh
echo $GOROOT
# If this shows the wrong version, VS Code will too

# Check what it should be
goenv version
cat .go-version
```

**Alternative:** If you don't want to restart VS Code, open a new integrated terminal in VS Code. New terminals get fresh environment variables, though the Go extension might still use the old environment until VS Code restarts.

**⚠️ Common mistake:** "Developer: Reload Window" only reloads the UI - it does NOT refresh environment variables. You must completely quit and relaunch VS Code.

See [VSCODE_VERSION_MISMATCH.md](../../VSCODE_VERSION_MISMATCH.md) for detailed troubleshooting.

### Environment Variables Not Set

**Symptom:** `echo $GOROOT` returns empty in VS Code terminal.

**Solution:**

1. **Check shell initialization:**

   ```bash
   # Add to ~/.bashrc, ~/.zshrc, etc.
   eval "$(goenv init -)"
   ```

2. **Restart VS Code from terminal:**

   ```bash
   # Close VS Code completely
   # Reopen from terminal
   code .
   ```

3. **Or use explicit settings.json:**
   ```json
   {
     "go.goroot": "${env:HOME}/.goenv/versions/1.22.0/go"
   }
   ```

### Go Extension Not Finding Tools

**Symptom:** "gopls not found" or other tool errors.

**Solution:**

1. **Install Go tools:**

   - `Cmd+Shift+P` → "Go: Install/Update Tools"
   - Select all tools

2. **Set tools path:**

   ```json
   {
     "go.toolsGopath": "${env:HOME}/go/tools"
   }
   ```

3. **Verify installation:**
   ```bash
   ls -la ~/go/tools/bin
   ```

### Changes to .go-version Not Picked Up

**Symptom:** Changed version with `goenv use` but VS Code still uses old version.

**Solution:**

1. **Reload window:**

   - `Cmd+Shift+P` → "Developer: Reload Window"

2. **Or restart VS Code from terminal:**

   ```bash
   # In VS Code: Cmd+Q to quit
   # In terminal:
   code .
   ```

3. **Check goenv sees the change:**
   ```bash
   # In terminal
   goenv version
   # Should show new version
   ```

### Different Version in Integrated Terminal vs Extension

**Symptom:** Terminal has correct version, but extension errors reference different version.

**Solution:**

1. **Check for conflicting settings:**

   - User settings vs workspace settings
   - System Go installation on PATH

2. **Use explicit GOROOT:**

   ```json
   {
     "go.goroot": "${env:GOROOT}",
     "go.alternateTools": {
       "go": "${env:GOROOT}/bin/go"
     }
   }
   ```

3. **Remove system Go from PATH** (optional):
   ```bash
   # If you only use goenv, remove system Go
   # macOS: brew uninstall go
   # Ubuntu: sudo apt remove golang-go
   ```

## Best Practices

### 1. Always Use .go-version Files

Commit `.go-version` to your repository:

```bash
goenv use 1.22.0
git add .go-version
git commit -m "Set Go version to 1.22.0"
```

This ensures all team members use the same Go version.

### 2. Include .vscode/settings.json

Make VS Code integration seamless for your team:

```bash
git add .vscode/settings.json
git commit -m "Add VS Code Go configuration"
```

**Recommended settings.json:**

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools",
  "go.useLanguageServer": true,
  "go.toolsManagement.autoUpdate": true,
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    }
  }
}
```

### 3. Document Setup in README

Add to your project's README:

```markdown
## Development Setup

This project uses goenv to manage Go versions.

1. Install goenv: https://github.com/yourusername/goenv
2. Install Go version: `goenv install`
3. Launch VS Code from terminal: `code .`

The Go version is specified in `.go-version`.
```

### 4. Use Shell Initialization

Ensure goenv is initialized in your shell profile:

**Bash** (`~/.bashrc`):

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**Zsh** (`~/.zshrc`):

```zsh
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**Fish** (`~/.config/fish/config.fish`):

```fish
set -gx GOENV_ROOT $HOME/.goenv
set -gx PATH $GOENV_ROOT/bin $PATH
goenv init - | source
```

### 5. Launch from Terminal

Make it a habit:

```bash
# Instead of: Open VS Code from Finder/Dock
# Do: Launch from terminal
cd ~/projects/myapp
code .
```

**Create an alias:**

```bash
# Add to ~/.bashrc or ~/.zshrc
alias proj='cd ~/projects && cd $(ls -1 | fzf) && code .'
```

### 6. Verify Setup Script

Create a verification script for your project:

```bash
#!/bin/bash
# verify-setup.sh

echo "Checking Go version..."
EXPECTED=$(cat .go-version)
ACTUAL=$(go version | awk '{print $3}' | sed 's/go//')

if [[ "$ACTUAL" != "$EXPECTED" ]]; then
  echo "❌ Wrong Go version!"
  echo "   Expected: $EXPECTED"
  echo "   Actual: $ACTUAL"
  echo ""
  echo "Fix it:"
  echo "  goenv install $EXPECTED"
  echo "  goenv local $EXPECTED"
  exit 1
fi

echo "✅ Go version correct: $ACTUAL"
echo ""
echo "Checking GOROOT..."
if [[ -z "$GOROOT" ]]; then
  echo "❌ GOROOT not set!"
  echo "   Run: eval \"\$(goenv init -)\""
  exit 1
fi

echo "✅ GOROOT: $GOROOT"
echo ""
echo "All checks passed! You're ready to develop."
```

Make it executable:

```bash
chmod +x verify-setup.sh
./verify-setup.sh
```

### 7. Team Onboarding Checklist

Create a checklist for new team members:

```markdown
## Go Development Setup

- [ ] Install goenv: `brew install goenv` (or see [install docs](./docs/user-guide/INSTALL.md))
- [ ] Add goenv to shell: `eval "$(goenv init -)"`
- [ ] Install project Go version: `goenv install`
- [ ] Install VS Code: https://code.visualstudio.com/
- [ ] Install Go extension: https://marketplace.visualstudio.com/items?itemName=golang.go
- [ ] Launch VS Code from terminal: `code .`
- [ ] Install Go tools: Cmd+Shift+P → "Go: Install/Update Tools"
- [ ] Verify setup: `./verify-setup.sh`
```

## Related Documentation

- **[GOPATH Integration](../advanced/GOPATH_INTEGRATION.md)** - Advanced GOPATH and tooling setup
- **[How It Works](./HOW_IT_WORKS.md)** - Understanding goenv's environment management
- **[Commands Reference](../reference/COMMANDS.md)** - All goenv commands

## Common Workflows

### Starting a New Project

**New way (with automation):**

```bash
# Create project
mkdir ~/projects/myapp && cd ~/projects/myapp

# Initialize Go module with VS Code setup in one command
goenv use 1.22.0 --vscode
go mod init github.com/you/myapp

# Initialize git
git init
git add .go-version .vscode/ go.mod
git commit -m "Initial commit with goenv and VS Code config"

# Launch VS Code
code .
```

**Traditional way (manual):**

```bash
# Create project
mkdir ~/projects/myapp && cd ~/projects/myapp

# Initialize Go module
goenv use 1.22.0
go mod init github.com/you/myapp

# Setup VS Code
goenv vscode init

# Initialize git
git init
git add .go-version .vscode/ go.mod
git commit -m "Initial commit"

# Launch VS Code
code .
```

### Switching Go Versions

```bash
# Change version
goenv use 1.21.5

# Reload VS Code
# Cmd+Shift+P → "Developer: Reload Window"

# Or restart from terminal
code .
```

### Working on Multiple Projects

```bash
# Project A uses Go 1.22
cd ~/projects/app-a
goenv use 1.22.0
code .

# Project B uses Go 1.21 (in new VS Code window)
cd ~/projects/app-b
goenv use 1.21.5
code .
```

Each VS Code window will have the correct Go version for its project.

### CI/CD Consistency

Ensure CI uses the same Go version:

**.github/workflows/test.yml:**

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Read Go version
        id: go-version
        run: echo "version=$(cat .go-version)" >> $GITHUB_OUTPUT

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ steps.go-version.outputs.version }}

      - run: go test -v ./...
```

This reads the `.go-version` file to match your local development environment.

---

**Next Steps:**

- Set up VS Code with one of the methods above
- Verify with `go version` and `echo $GOROOT` in the integrated terminal
- Share `.vscode/settings.json` with your team
- Document the setup in your project's README
