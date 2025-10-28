# VS Code Integration Guide

This guide shows how to use goenv seamlessly with Visual Studio Code and the official Go extension.

## Table of Contents

- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [VS Code Commands](#vscode-commands)
- [Setup Methods](#setup-methods)
- [Project Configuration](#project-configuration)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

## 🎯 Choosing Your Configuration Mode

**TL;DR: How do you launch VS Code?**

| **You launch VS Code from...**    | **Use this mode**            | **Command**                    | **Why**                                                       |
| --------------------------------- | ---------------------------- | ------------------------------ | ------------------------------------------------------------- |
| 🖱️ **Dock / Finder / Start Menu** | **Absolute Paths** (default) | `goenv vscode init`            | Works even when VS Code doesn't inherit shell environment     |
| 💻 **Terminal** (`code .`)        | **Environment Variables**    | `goenv vscode init --env-vars` | Automatically updates when you change Go versions             |
| 🤷 **Not sure / Both**            | **Absolute Paths** (default) | `goenv vscode init`            | Most reliable; use `goenv vscode sync` when changing versions |

### Understanding the Two Modes

#### Absolute Paths Mode (Default - Recommended)

```json
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.23.2",
  "go.gopath": "${env:HOME}/go/1.23.2"
}
```

**✅ Pros:**

- Works when VS Code is opened from GUI (Dock, Finder, Start Menu)
- Reliable - doesn't depend on shell environment
- Team-friendly - uses `${env:HOME}` for portability

**❌ Cons:**

- Must run `goenv vscode sync` after changing Go versions
- Settings contain specific version number

**When to use:** You open VS Code from the GUI, or you're not sure

#### Environment Variables Mode

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}"
}
```

**✅ Pros:**

- Automatically tracks current Go version
- No need to sync after `goenv use` commands
- Cleaner settings files

**❌ Cons:**

- **ONLY works if you launch VS Code from a terminal** with goenv initialized
- **Reload Window does NOT refresh environment variables**
- Must quit VS Code and restart from terminal to pick up changes

**When to use:** You always launch VS Code from terminal (`code .`)

### ⚠️ Critical: Reload Window vs Restart

**Common mistake:** Using "Developer: Reload Window" after changing Go versions

| **Action**                             | **Refreshes Environment?** | **When to Use**                   |
| -------------------------------------- | -------------------------- | --------------------------------- |
| ⌘+Shift+P → "Developer: Reload Window" | ❌ **NO**                  | Testing extensions, UI updates    |
| Quit VS Code → `code .` from terminal  | ✅ **YES**                 | After `goenv use` (env vars mode) |
| `goenv vscode sync` → Reload Window    | ✅ **YES**                 | After `goenv use` (absolute mode) |

**Environment Variables Mode requires full restart:**

```bash
# Wrong - Reload Window does NOT work for env vars!
goenv use 1.24.0
# ⌘+Shift+P → "Developer: Reload Window"  ❌ Still uses old version

# Correct - Must restart from terminal
goenv use 1.24.0
# Quit VS Code completely
code .  ✅ Now uses new version
```

**Absolute Paths Mode uses sync command:**

```bash
# When using absolute paths
goenv use 1.24.0
goenv vscode sync  # Updates settings.json with new version
# ⌘+Shift+P → "Developer: Reload Window"  ✅ Works!
```

### 📋 Quick Decision Flowchart

```
How do you typically open VS Code?
│
├─ From GUI (Dock/Finder/Start Menu)
│  └─> Use ABSOLUTE PATHS mode (default)
│      Command: goenv vscode init
│      After version change: goenv vscode sync
│
├─ Always from terminal (code .)
│  └─> Use ENVIRONMENT VARIABLES mode
│      Command: goenv vscode init --env-vars
│      After version change: Quit VS Code, reopen from terminal
│
└─ Sometimes GUI, sometimes terminal
   └─> Use ABSOLUTE PATHS mode (default)
       Command: goenv vscode init
       Works in both scenarios with goenv vscode sync
```

## Quick Start

### Option 1: Complete Setup - All-in-One Command (NEW! ⚡)

The absolute fastest way for new users - does everything in one command:

```bash
# Navigate to your project
cd ~/projects/myapp

# Complete setup: init + sync + doctor
goenv vscode setup
```

This single command automatically:

- ✅ Creates `.vscode/settings.json` with goenv configuration
- ✅ Generates `.vscode/extensions.json` (recommends Go extension)
- ✅ Syncs settings with current Go version
- ✅ Runs diagnostics to verify everything works
- ✅ Shows clear error messages if anything needs fixing

**Perfect for:**

- 🆕 First-time goenv + VS Code users
- 🚀 Quick project onboarding
- 🔍 Troubleshooting when things aren't working
- 📦 CI/CD workspace preparation

**Advanced options:**

```bash
# Use advanced template with gopls settings
goenv vscode setup --template advanced

# Force overwrite existing settings
goenv vscode setup --force

# Use environment variables mode (for terminal-only users)
goenv vscode setup --env-vars

# Dry run (see what would be done)
goenv vscode setup --dry-run
```

**What it does behind the scenes:**

```bash
# Equivalent to running these three commands:
goenv vscode init      # Create configuration files
goenv vscode sync      # Update with current Go version
goenv doctor           # Verify installation
```

### Option 2: Automatic Setup with `goenv use` (Easiest - NEW! ⚡)

Set project Go version AND configure VS Code in one command:

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

### 🎒 Portability Knobs (Advanced)

For maximum portability across different machines or team collaboration, goenv provides two special flags:

#### `--workspace-paths`: Workspace-Relative Paths

```bash
goenv vscode init --workspace-paths
```

**What it does:** Converts absolute paths to `${workspaceFolder}`-relative syntax where possible.

**Example output:**

```json
{
  "go.goroot": "${workspaceFolder}/../.goenv/versions/1.23.2",
  "go.gopath": "${workspaceFolder}/../go/1.23.2"
}
```

**When to use:**

- ✅ Sharing projects where goenv is in a predictable relative location
- ✅ Monorepos where `.goenv` is checked into the repository
- ✅ Docker/container setups with mounted volumes at specific paths

**When NOT to use:**

- ❌ Standard goenv installation in `$HOME/.goenv` (absolute paths are clearer)
- ❌ Team members have different directory structures
- ❌ Using environment variables mode (this flag only affects absolute path mode)

#### `--versioned-tools`: Per-Version Tools Directory

```bash
goenv vscode init --versioned-tools
```

**What it does:** Sets `go.toolsGopath` to use version-specific tools directory instead of shared tools.

**Standard behavior (default):**

```json
{
  "go.toolsGopath": "${env:HOME}/go/tools" // Shared across Go versions
}
```

**With `--versioned-tools`:**

```json
{
  "go.toolsGopath": "${env:HOME}/go/1.23.2/tools" // Isolated per version
}
```

**When to use:**

- ✅ Testing tool compatibility with different Go versions
- ✅ Projects requiring specific tool versions tied to Go version
- ✅ Avoiding tool conflicts between Go 1.x and 1.y

**When NOT to use:**

- ❌ Normal development (shared tools work fine and save disk space)
- ❌ CI/CD (tools are typically installed fresh anyway)
- ❌ You want faster tool updates across all projects

**Combining both flags:**

```bash
goenv vscode init --workspace-paths --versioned-tools
```

This creates maximum isolation: workspace-relative paths AND version-specific tools.

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

## Platform-Specific Behavior

VS Code integration with goenv works across all major platforms, but there are important differences in how paths, environment variables, and shell integration work on each operating system.

### Path Separators and Environment Variables

| Aspect               | Linux/macOS                            | Windows                                              |
| -------------------- | -------------------------------------- | ---------------------------------------------------- |
| **Path separator**   | Forward slash (`/`)                    | Backslash (`\`) or forward slash (both work)         |
| **Home variable**    | `${env:HOME}`                          | `${env:USERPROFILE}` (or `${env:HOME}` in Git Bash)  |
| **Example goroot**   | `/home/user/.goenv/versions/1.23.2/go` | `C:\Users\user\.goenv\versions\1.23.2\go`            |
| **Example gopath**   | `/home/user/go/1.23.2`                 | `C:\Users\user\go\1.23.2`                            |
| **Workspace folder** | `${workspaceFolder}/bin`               | `${workspaceFolder}\bin` or `${workspaceFolder}/bin` |

### Recommended Settings by Platform

#### Linux/macOS Settings

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

**Or with absolute paths:**

```json
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.23.2/go",
  "go.gopath": "${env:HOME}/go/1.23.2",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

#### Windows Settings (PowerShell)

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:USERPROFILE}\\go\\tools"
}
```

**Or with absolute paths:**

```json
{
  "go.goroot": "${env:USERPROFILE}\\.goenv\\versions\\1.23.2\\go",
  "go.gopath": "${env:USERPROFILE}\\go\\1.23.2",
  "go.toolsGopath": "${env:USERPROFILE}\\go\\tools"
}
```

**Windows with Git Bash:**

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

### Shell Integration by Platform

| Platform    | Default Shell | VS Code Terminal          | goenv init Location                                         |
| ----------- | ------------- | ------------------------- | ----------------------------------------------------------- |
| **Linux**   | bash          | bash, zsh, fish           | `~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`       |
| **macOS**   | zsh (10.15+)  | zsh, bash, fish           | `~/.zshrc`, `~/.bash_profile`, `~/.config/fish/config.fish` |
| **Windows** | PowerShell    | PowerShell, cmd, Git Bash | PowerShell profile, Git Bash `~/.bashrc`                    |

**Linux Example (`~/.bashrc`):**

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**macOS Example (`~/.zshrc`):**

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**Windows PowerShell Example:**

```powershell
# PowerShell profile: $PROFILE
$env:GOENV_ROOT = "$env:USERPROFILE\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
Invoke-Expression (goenv init - | Out-String)
```

**Windows Git Bash Example (`~/.bashrc`):**

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### How VS Code Launches Differ by Platform

| Platform    | GUI Launch                   | Terminal Launch              | Environment Inheritance                                                                  |
| ----------- | ---------------------------- | ---------------------------- | ---------------------------------------------------------------------------------------- |
| **Linux**   | Desktop launcher, app menu   | `code .` from terminal       | GUI launch: Limited (systemd user environment)<br>Terminal: Full shell environment       |
| **macOS**   | Dock, Finder, Spotlight      | `code .` from terminal       | GUI launch: `~/.MacOSX/environment.plist` or launchd<br>Terminal: Full shell environment |
| **Windows** | Start Menu, desktop shortcut | `code .` from PowerShell/cmd | GUI launch: System environment variables only<br>Terminal: Full shell environment        |

**Implications:**

- **Linux/macOS GUI launches:** May not have `GOROOT`/`GOPATH` unless you configure system environment (not recommended)
- **Windows GUI launches:** Only sees system environment variables, not PowerShell profile variables
- **All platforms:** Terminal launch (`code .`) is most reliable for inheriting goenv environment

### Platform-Specific Tips

#### Linux

**Desktop launchers:** If you launch VS Code from app menus/launchers, create a wrapper script:

```bash
# ~/.local/bin/code-goenv
#!/bin/bash
eval "$(goenv init -)"
exec code "$@"
```

Then use the wrapper: `code-goenv .` or update your `.desktop` file to use it.

**systemd user environment (advanced):**

```bash
# Add to ~/.config/environment.d/goenv.conf
GOENV_ROOT=/home/user/.goenv
PATH=/home/user/.goenv/shims:/home/user/.goenv/bin:/usr/local/bin:/usr/bin:/bin
```

#### macOS

**Dock/Finder launches:** Use absolute path mode or configure launchd environment:

```bash
# Set environment for GUI apps
launchctl setenv GOENV_ROOT "$HOME/.goenv"
launchctl setenv PATH "$HOME/.goenv/shims:$PATH"
```

**Recommended:** Use `goenv vscode init` (absolute paths) and `goenv vscode sync` when changing versions.

**Apple Silicon (ARM64) considerations:**

- Go binaries may be Intel (x86_64) or ARM64 (arm64)
- VS Code runs natively on ARM64
- Both work, but ARM64 Go binaries are faster
- goenv handles architecture automatically

#### Windows

**PowerShell vs Git Bash:**

| Shell      | Path Style     | HOME Variable      | Best For                   |
| ---------- | -------------- | ------------------ | -------------------------- |
| PowerShell | `C:\Users\...` | `$env:USERPROFILE` | Native Windows development |
| Git Bash   | `/c/Users/...` | `$HOME`            | Unix-like environment      |

**Recommendation:** Choose one shell and stick with it. Don't mix PowerShell and Git Bash settings.

**Windows-specific VS Code settings:**

```json
{
  "terminal.integrated.defaultProfile.windows": "PowerShell",
  "go.goroot": "${env:USERPROFILE}\\.goenv\\versions\\1.23.2\\go",
  "go.gopath": "${env:USERPROFILE}\\go\\1.23.2"
}
```

**WSL (Windows Subsystem for Linux):**

If using WSL2, treat it as Linux:

```bash
# Inside WSL2
export GOENV_ROOT="$HOME/.goenv"
eval "$(goenv init -)"
```

VS Code's WSL extension handles the environment automatically when you use "Remote-WSL: New Window".

### Cross-Platform Configuration (for Teams)

If your team uses multiple platforms, use environment variable references for maximum portability:

```json
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

**Why this works:**

- `${env:HOME}` resolves to correct value on all platforms
- `${env:GOROOT}` and `${env:GOPATH}` are set by goenv on all platforms
- Path separators are handled by Go tooling

**What team members must do:**

1. Install goenv on their platform
2. Add goenv init to their shell profile
3. Launch VS Code from terminal (`code .`)
4. Reload when changing Go versions (quit VS Code, restart from terminal)

**Alternative for teams:** Use absolute paths with `${env:HOME}` or `${env:USERPROFILE}`:

```json
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.23.2/go",
  "go.gopath": "${env:HOME}/go/1.23.2",
  "go.toolsGopath": "${env:HOME}/go/tools"
}
```

Then document the `goenv vscode sync` command for when versions change.

### Platform Support Matrix

| Feature                  | Linux | macOS | Windows | Notes                                                 |
| ------------------------ | ----- | ----- | ------- | ----------------------------------------------------- |
| `goenv vscode init`      | ✅    | ✅    | ✅      | Creates settings.json with platform-appropriate paths |
| `goenv vscode sync`      | ✅    | ✅    | ✅      | Updates settings when Go version changes              |
| `goenv vscode doctor`    | ✅    | ✅    | ✅      | Checks platform-specific settings                     |
| Environment variables    | ✅    | ✅    | ✅      | Requires shell initialization (terminal launch)       |
| Absolute paths           | ✅    | ✅    | ✅      | Works with GUI launches                               |
| Workspace-relative paths | ✅    | ✅    | ✅      | Useful for monorepos and containers                   |

**For more platform details, see:**

- **[Platform Support Matrix](../PLATFORM_SUPPORT.md)** - Comprehensive OS/architecture compatibility
- **[Installation Guide](INSTALL.md)** - Platform-specific installation instructions
- **[How It Works](HOW_IT_WORKS.md)** - Understanding goenv's cross-platform architecture

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

## VS Code Commands

goenv includes dedicated commands for managing VS Code integration:

### `goenv vscode init`

Initialize VS Code workspace configuration.

```bash
goenv vscode init                    # Basic template
goenv vscode init --template advanced  # Advanced settings
goenv vscode init --template monorepo  # Monorepo settings
goenv vscode init --force            # Overwrite existing
goenv vscode init --dry-run          # Preview changes
```

### `goenv vscode sync`

Re-sync settings after changing Go version.

```bash
goenv local 1.24.0
goenv vscode sync              # Update paths to new version
goenv vscode sync --dry-run    # Preview changes
```

### `goenv vscode status`

Check integration status.

```bash
goenv vscode status            # Human-readable
goenv vscode status --json     # Machine-readable for CI
```

### `goenv vscode doctor`

Run comprehensive health checks.

```bash
goenv vscode doctor            # Check everything
goenv vscode doctor --json     # JSON output for automation
```

**What it checks:**

- VS Code settings file exists
- Go version is configured
- Go installation is present
- gopls is available
- Tools directory is writable
- Workspace structure (go.mod/go.work)
- Settings match current version
- Go extension is recommended

### `goenv vscode revert`

Restore settings from backup.

```bash
goenv vscode revert    # Rollback to previous settings
```

> **Note:** For complete command reference, see [VSCODE_QUICK_REFERENCE.md](../reference/VSCODE_QUICK_REFERENCE.md)

## Setup Methods

### 📊 Quick Decision Guide: Which Setup Method Should I Use?

Choose your setup method based on how you launch VS Code:

| **Launch Method**                                | **Recommended Setup**          | **Configuration Mode** | **When to Refresh**                            |
| ------------------------------------------------ | ------------------------------ | ---------------------- | ---------------------------------------------- |
| 🖥️ **Terminal** (`code .`)                       | Method 1 (Terminal Launch)     | Environment variables  | Quit VS Code, restart from terminal            |
| 🖱️ **GUI** (Dock/Finder/Start Menu)              | Method 2 with absolute paths   | Hardcoded paths        | Update settings.json when changing Go version  |
| 👥 **Team project** (version controlled)         | Method 2 with env vars         | Environment variables  | Team members restart VS Code from terminal     |
| 🔀 **Mixed** (sometimes terminal, sometimes GUI) | Method 2 with absolute paths   | Hardcoded paths        | Run `goenv vscode sync` when changing versions |
| 🚀 **Quick project setup**                       | `goenv use <version> --vscode` | Environment variables  | Quit VS Code, restart from terminal            |

### 🔄 Reload vs. Restart Decision Matrix

**Common confusion:** "Developer: Reload Window" is NOT the same as restarting VS Code!

| **Action**                              | **Reloads UI** | **Refreshes Env Vars** | **When to Use**                                |
| --------------------------------------- | -------------- | ---------------------- | ---------------------------------------------- |
| ⌘+Shift+P → "Developer: Reload Window"  | ✅             | ❌                     | Testing extension updates, UI fixes            |
| Quit VS Code + Relaunch from terminal   | ✅             | ✅                     | **After changing Go versions** (env var mode)  |
| Quit VS Code + Relaunch from GUI        | ✅             | ❌                     | Only works with absolute path mode             |
| `goenv vscode sync` then Reload Window  | ✅             | ❌                     | **After changing Go versions** (absolute mode) |
| Open new integrated terminal in VS Code | ❌             | ✅ (new terminal only) | Quick check, but extension still uses old env  |

### 💡 Troubleshooting Quick Reference

| **Symptom**                                                  | **Likely Cause**                        | **Solution**                                                 |
| ------------------------------------------------------------ | --------------------------------------- | ------------------------------------------------------------ |
| 🔴 Wrong Go version after `goenv use`                        | Used "Reload Window" instead of restart | Quit VS Code completely, relaunch from terminal              |
| 🔴 `go version` in terminal is correct, but VS Code is wrong | VS Code launched from GUI, not terminal | Either: (1) Restart from terminal, or (2) Use absolute paths |
| 🔴 `$GOROOT` is empty in integrated terminal                 | `goenv init` not in shell config        | Add `eval "$(goenv init -)"` to ~/.bashrc/~/.zshrc           |
| 🟡 Settings don't update when changing versions              | Using absolute paths, not env vars      | Run `goenv vscode sync` after changing versions              |
| 🟡 Team members see different Go versions                    | Using absolute paths in version control | Switch to env vars mode: `"go.goroot": "${env:GOROOT}"`      |

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

### 🔍 Quick Troubleshooting Matrix

Use this matrix to diagnose and fix common VS Code + goenv issues:

| **Symptom**                                            | **Root Cause**                          | **Mode** | **Quick Fix**                                          | **Prevention**                                  |
| ------------------------------------------------------ | --------------------------------------- | -------- | ------------------------------------------------------ | ----------------------------------------------- |
| 🔴 **Wrong Go version in VS Code after `goenv use`**   | Used "Reload Window" instead of restart | Env Vars | Quit VS Code → relaunch from terminal (`code .`)       | Use absolute paths mode or always restart fully |
| 🔴 **Terminal shows correct version, VS Code doesn't** | VS Code opened from GUI, not terminal   | Env Vars | Switch to absolute mode: `goenv vscode init` (default) | Launch VS Code from terminal only               |
| 🔴 **`$GOROOT` empty in integrated terminal**          | `goenv init` not in shell profile       | Env Vars | Add `eval "$(goenv init -)"` to `~/.bashrc`/`~/.zshrc` | Check shell config with new terminal            |
| 🔴 **gopls not found / Go tools fail**                 | Tools installed for wrong Go version    | Both     | Run `goenv tools update` or `goenv rehash`             | Update tools after version changes              |
| 🟡 **Settings don't update when changing versions**    | Using absolute paths                    | Absolute | Run `goenv vscode sync` after `goenv use`              | Use env vars mode or remember to sync           |
| 🟡 **Team members see different Go versions**          | Absolute paths in version control       | Absolute | Switch to env vars: `goenv vscode init --env-vars`     | Use env vars for shared projects                |
| 🟡 **VS Code can't find Go at all**                    | No settings configured                  | None     | Run `goenv vscode init`                                | Configure on project setup                      |
| 🟢 **Everything works from terminal but not GUI**      | Expected behavior with env vars         | Env Vars | This is normal! Use absolute mode for GUI launches     | Understand mode differences                     |

### 🚦 Mode-Specific Workflows

**If using Absolute Paths Mode (default):**

```bash
# After changing Go version:
goenv use 1.24.0
goenv vscode sync              # Updates paths in settings.json
# ⌘+Shift+P → "Developer: Reload Window"  ✅ Works!
```

**If using Environment Variables Mode (`--env-vars`):**

```bash
# After changing Go version:
goenv use 1.24.0
# Quit VS Code completely (⌘+Q on macOS)
code .                         # Restart from terminal  ✅ Works!
# Note: "Reload Window" will NOT work  ❌
```

### 🎯 Decision: Which Mode Am I Using?

**Check your `.vscode/settings.json`:**

```json
// Absolute Paths Mode:
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.23.2",  // ← Contains version number
  "go.gopath": "${env:HOME}/go/1.23.2"
}

// Environment Variables Mode:
{
  "go.goroot": "${env:GOROOT}",  // ← References env var
  "go.gopath": "${env:GOPATH}"
}
```

**Or run:**

```bash
goenv vscode status
# Shows: "Mode: Absolute paths" or "Mode: Environment variables"
```

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

**⚠️ Critical:** "Developer: Reload Window" only reloads the UI - it does NOT refresh environment variables. You must completely quit and relaunch VS Code from your terminal.

**💡 Prefer GUI launches?** If you regularly open VS Code via Finder/Dock/Spotlight/Start Menu instead of the terminal, use **absolute path mode** (the default) to avoid environment variable dependency:

```bash
# Initialize VS Code with absolute paths (DEFAULT - no environment dependency)
goenv vscode init

# Or use advanced template with absolute paths
goenv vscode init --template advanced

# Example of what this creates in .vscode/settings.json:
{
  "go.goroot": "/Users/you/.goenv/versions/1.22.0/go",
  "go.gopath": "/Users/you/go/1.22.0"
}
```

**For terminal-only workflows,** you can use environment variables instead:

```bash
# Use environment variables (requires terminal launch)
goenv vscode init --env-vars

# This creates:
{
  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}"
}
```

**Absolute path mode trade-offs:**

- ✅ Works from GUI launches (Dock, Finder, Start Menu)
- ✅ No dependency on terminal environment
- ✅ Consistent regardless of how VS Code is opened
- ❌ Must run `goenv vscode sync` when changing Go versions
- ❌ Paths are user-specific (not great for team sharing)

**When to use absolute paths:**

- You primarily launch VS Code from GUI (not terminal)
- You rarely change Go versions
- You're on a single-user machine

**When to use environment variables:**

- You launch VS Code from terminal (`code .`)
- You frequently switch Go versions
- You're sharing settings with a team (via git)

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

### 1. Choose the Right Configuration Mode for Your Workflow

**🎯 The Golden Rule:**

> **If you launch VS Code from the Dock/Finder/Start Menu → Use Absolute Paths (default)**  
> **If you always launch from terminal (`code .`) → Use Environment Variables**

#### For Solo Projects (Use Absolute Paths - Default)

```bash
# Just run this - works from anywhere
goenv vscode init
```

**Why:** Most reliable for solo developers who might launch VS Code from the GUI.

#### For Team Projects (Use Environment Variables)

```bash
# Use env vars for team consistency
goenv vscode init --env-vars
```

**Why:** Team members can have different GOENV_ROOT locations. Environment variables work across different setups.

**Add to README.md:**

```markdown
## VS Code Setup

This project uses goenv with environment variable mode.

**Required:**

1. Install goenv and initialize: `eval "$(goenv init -)"`
2. Launch VS Code from terminal: `code .`
3. After changing Go versions: Quit VS Code and relaunch from terminal

**Important:** Do NOT use "Developer: Reload Window" - it won't refresh environment variables.
Always quit and restart VS Code from your terminal.
```

### 2. Always Use .go-version Files

Commit `.go-version` to your repository:

```bash
goenv use 1.22.0
git add .go-version
git commit -m "Set Go version to 1.22.0"
```

This ensures all team members use the same Go version.

### 3. Include .vscode/settings.json (With Important Notes)

Make VS Code integration seamless for your team:

```bash
git add .vscode/settings.json
git commit -m "Add VS Code Go configuration"
```

**Recommended settings.json (Environment Variables Mode - for teams):**

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

**⚠️ Important for Team Projects:**

Add this comment to the top of your `.vscode/settings.json`:

```json
{
  // This project uses goenv with environment variables.
  // IMPORTANT: Launch VS Code from terminal (code .) to inherit environment.
  // After changing Go versions, quit VS Code completely and restart from terminal.
  // "Developer: Reload Window" will NOT work!

  "go.goroot": "${env:GOROOT}",
  "go.gopath": "${env:GOPATH}"
}
```

**Alternative for Mixed Launch Methods (Absolute Paths):**

If team members might launch from GUI or terminal:

```json
{
  // Using absolute paths mode - works from GUI or terminal.
  // After changing Go versions, run: goenv vscode sync

  "go.goroot": "${env:HOME}/.goenv/versions/1.23.2",
  "go.gopath": "${env:HOME}/go/1.23.2"
}
```

**⚠️ Note:** With absolute paths, remember to add `.vscode/settings.json` to `.gitignore` if paths are user-specific, OR use `${env:HOME}` to make them portable.

### 4. Document Setup in README

Add comprehensive setup instructions to your project's README:

**For Environment Variables Mode (recommended for teams):**

````markdown
## Development Setup

This project uses goenv to manage Go versions.

### First Time Setup

1. **Install goenv:**

   ```bash
   # macOS/Linux
   curl -fsSL https://github.com/go-nv/goenv/raw/master/install.sh | bash

   # Or see: https://github.com/go-nv/goenv#installation
   ```
````

2. **Initialize goenv in your shell:**

   ```bash
   # Add to ~/.bashrc, ~/.zshrc, or equivalent
   eval "$(goenv init -)"

   # Restart shell or run: exec $SHELL
   ```

3. **Install Go version:**

   ```bash
   cd /path/to/project
   goenv install  # Reads .go-version file
   ```

4. **Launch VS Code from terminal:**
   ```bash
   code .
   ```

### Changing Go Versions

After running `goenv use <version>`:

**IMPORTANT:** You MUST restart VS Code properly:

1. Quit VS Code completely (⌘+Q on macOS, not just close window)
2. Relaunch from terminal: `code .`

**Common Mistake:** Using "Developer: Reload Window" (⌘+Shift+P) does NOT refresh environment variables!

### Troubleshooting

- **VS Code shows wrong Go version:** Quit VS Code and restart from terminal
- **`$GOROOT` is empty:** Ensure `eval "$(goenv init -)"` is in your shell config
- **gopls not found:** Run `goenv tools update` after changing versions

````

**For Absolute Paths Mode (solo projects):**

```markdown
## Development Setup

This project uses goenv with absolute path configuration.

### First Time Setup

1. Install goenv (see main docs)
2. Install Go: `goenv install`
3. Initialize VS Code: `goenv vscode init`
4. Open VS Code any way you like (GUI or terminal)

### Changing Go Versions

After running `goenv use <version>`:

```bash
goenv vscode sync  # Updates paths in .vscode/settings.json
````

Then reload VS Code: ⌘+Shift+P → "Developer: Reload Window"

````

### 5. Use Shell Initialization

Ensure goenv is initialized in your shell profile:

**Bash** (`~/.bashrc`):

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
````

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

- [ ] Install goenv: `brew install goenv` (or see [install docs](./INSTALL.md))
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
