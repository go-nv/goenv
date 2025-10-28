# Modern vs Legacy Commands

goenv provides modern, intuitive commands while maintaining backward compatibility with legacy commands from the bash implementation.

## Table of Contents

- [Quick Reference](#quick-reference)
- [Why Modern Commands?](#why-modern-commands)
- [Command Mapping](#command-mapping)
- [Migration Guide](#migration-guide)
- [Best Practices](#best-practices)

## Quick Reference

### Recommended Modern Commands

Use these commands for new scripts and workflows:

| Task | Modern Command | Why Use It |
|------|----------------|------------|
| **Set version** | `goenv use <version>` | Unified interface for local/global |
| **Check active version** | `goenv current` | Clear name, shows source |
| **List versions** | `goenv list` | Consistent with other tools |
| **Install version** | `goenv install <version>` | Already modern ✓ |
| **Uninstall version** | `goenv uninstall <version>` | Already modern ✓ |

### Legacy Commands (For Compatibility)

These work but are not recommended for new code:

| Legacy Command | Modern Equivalent | Status |
|----------------|-------------------|--------|
| `goenv local <version>` | `goenv use <version>` | Deprecated |
| `goenv global <version>` | `goenv use <version> --global` | Deprecated |
| `goenv versions` | `goenv list` | Deprecated |
| `goenv version` | `goenv current` | Deprecated |

## Why Modern Commands?

### 1. Consistent Interface

**Modern:**
```bash
goenv use 1.25.2              # Set local version
goenv use 1.25.2 --global     # Set global version
```

**Legacy (inconsistent):**
```bash
goenv local 1.25.2            # Set local version
goenv global 1.25.2           # Set global version (different command!)
```

The modern approach uses flags for behavior modification rather than separate commands, making the interface more predictable.

### 2. Clear Names

**Modern:**
```bash
goenv current    # "What version am I currently using?"
goenv list       # "What versions are available?"
```

**Legacy (ambiguous):**
```bash
goenv version    # Is this THE version, or showing versions?
goenv versions   # Similar to above, confusing singular vs plural
```

Modern commands use clearer, more descriptive names that indicate their purpose.

### 3. Rich Output

**Modern commands provide more context:**

```bash
$ goenv current
1.25.2 (set by /Users/user/project/.go-version)
```

**Legacy commands give less information:**

```bash
$ goenv version
1.25.2
```

### 4. Future-Proof

Modern commands are actively developed with new features:

```bash
# Modern command with new features
goenv list --json               # Structured output for automation
goenv list --remote --stable    # Remote version discovery
goenv use 1.25.2 --vscode      # Integrated VS Code setup

# Legacy commands have minimal updates
goenv versions                  # Basic text output only
```

## Command Mapping

### Setting Versions

#### Local Version (Project-Specific)

**Modern (Recommended):**
```bash
cd ~/my-project
goenv use 1.25.2
```

**Legacy (Still Works):**
```bash
cd ~/my-project
goenv local 1.25.2
```

**What happens:**
- Creates `.go-version` file with `1.25.2`
- Sets this project to use Go 1.25.2
- Other projects unaffected

#### Global Version (User Default)

**Modern (Recommended):**
```bash
goenv use 1.24.8 --global
```

**Legacy (Still Works):**
```bash
goenv global 1.24.8
```

**What happens:**
- Updates `~/.goenv/version` file
- Sets default Go version for all projects without `.go-version`
- Existing `.go-version` files take precedence

### Checking Versions

#### Current Active Version

**Modern (Recommended):**
```bash
$ goenv current
1.25.2 (set by /Users/user/project/.go-version)
```

**Legacy (Still Works):**
```bash
$ goenv version
1.25.2
```

**Modern advantages:**
- Shows WHERE the version was set (`.go-version`, `GOENV_VERSION`, etc.)
- Helps troubleshoot "why am I using this version?"
- More verbose output with `--bare` flag for scripting

#### Listing Installed Versions

**Modern (Recommended):**
```bash
$ goenv list
  1.23.2
  1.24.8
* 1.25.2 (set by /Users/user/project/.go-version)
  system

# JSON output for automation
$ goenv list --json
```

**Legacy (Still Works):**
```bash
$ goenv versions
  1.23.2
  1.24.8
* 1.25.2 (set by /Users/user/project/.go-version)
  system
```

**Modern advantages:**
- Shorter, clearer name
- JSON output support
- Combined with `--remote` for available versions

#### Listing Available Versions

**Modern (Recommended):**
```bash
$ goenv list --remote
Available versions:
  1.23.0
  1.23.1
  1.23.2
  ...

# Filter to stable only
$ goenv list --remote --stable

# JSON output
$ goenv list --remote --json
```

**Legacy (Still Works):**
```bash
$ goenv install --list
```

**Modern advantages:**
- Consistent with `goenv list` (same command, different flag)
- Supports JSON output
- Can combine filters (stable, latest, etc.)

## Migration Guide

### For Scripts

If you have existing scripts using legacy commands:

#### Option 1: Update to Modern Commands (Recommended)

```bash
# Before (legacy)
goenv local 1.25.2
goenv global 1.24.8
VERSION=$(goenv version | awk '{print $1}')

# After (modern)
goenv use 1.25.2
goenv use 1.24.8 --global
VERSION=$(goenv current --bare)
```

#### Option 2: Keep Legacy Commands (Backward Compatibility)

Legacy commands will continue to work indefinitely:

```bash
# These will keep working
goenv local 1.25.2
goenv global 1.24.8
goenv versions
goenv version
```

**Note:** Legacy commands may not receive new features, but they won't be removed.

### For Documentation

When writing new documentation:

```markdown
❌ Don't use:
goenv local 1.25.2
goenv global 1.24.8

✅ Do use:
goenv use 1.25.2
goenv use 1.24.8 --global
```

### For Team Onboarding

Teach new team members modern commands:

```bash
# Modern workflow
goenv install 1.25.2       # Install Go version
goenv use 1.25.2           # Use it in project
goenv current              # Verify active version
goenv list                 # See installed versions
goenv list --remote        # Browse available versions
```

This provides a consistent, easy-to-remember interface.

## Best Practices

### 1. Use Modern Commands in New Code

```bash
# ✅ Good - Modern commands
goenv use 1.25.2
goenv current

# ❌ Avoid - Legacy commands in new code
goenv local 1.25.2
goenv version
```

### 2. Update Scripts Gradually

Don't rush to update working scripts:

```bash
# Existing CI/CD scripts with legacy commands
# ✅ These still work - update when convenient
goenv local $(cat .go-version)
goenv global 1.24.8

# New scripts or major refactors
# ✅ Use modern commands
goenv use $(cat .go-version)
goenv use 1.24.8 --global
```

### 3. Document Which Commands You Use

In project README files:

```markdown
## Go Version Management

This project uses goenv to manage Go versions.

### Setup
\`\`\`bash
# Install goenv (if needed)
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Install project Go version
goenv install        # Uses .go-version file

# Verify
goenv current        # Should show 1.25.2
\`\`\`

Note: We use modern goenv commands (`use`, `current`, `list`).
Legacy commands (`local`, `global`, `versions`) also work.
```

### 4. Use `--bare` for Scripting

Modern commands support `--bare` for machine-readable output:

```bash
# Get just the version number (no decoration)
VERSION=$(goenv current --bare)
echo $VERSION  # Outputs: 1.25.2

# Works in scripts
if [ "$(goenv current --bare)" = "1.25.2" ]; then
  echo "Using correct Go version"
fi
```

### 5. Leverage JSON Output

Modern commands support `--json` for structured data:

```bash
# Get structured data
goenv list --json | jq -r '.[] | select(.active) | .version'

# Remote versions as JSON
goenv list --remote --json | jq -r '.[].version'
```

## Command Comparison Examples

### Example 1: Project Setup

**Modern Approach:**
```bash
cd ~/new-project
goenv install 1.25.2           # Install if needed
goenv use 1.25.2               # Set for project
goenv current                  # Verify (shows source)
code .                         # Open in VS Code
```

**Legacy Approach:**
```bash
cd ~/new-project
goenv install 1.25.2           # Install if needed
goenv local 1.25.2             # Set for project
goenv version                  # Verify (less info)
code .                         # Open in VS Code
```

### Example 2: Checking Current State

**Modern Approach:**
```bash
$ goenv current
1.25.2 (set by /Users/user/project/.go-version)

$ goenv list
  1.23.2
  1.24.8
* 1.25.2 (set by /Users/user/project/.go-version)
```

**Legacy Approach:**
```bash
$ goenv version
1.25.2

$ goenv versions
  1.23.2
  1.24.8
* 1.25.2 (set by /Users/user/project/.go-version)
```

### Example 3: CI/CD Pipeline

**Modern Approach:**
```yaml
# .github/workflows/build.yml
steps:
  - name: Install Go version from .go-version
    run: |
      VERSION=$(cat .go-version)
      goenv install "$VERSION"
      goenv use "$VERSION"

  - name: Verify Go version
    run: |
      goenv current
      go version
```

**Legacy Approach:**
```yaml
# .github/workflows/build.yml
steps:
  - name: Install Go version from .go-version
    run: |
      VERSION=$(cat .go-version)
      goenv install "$VERSION"
      goenv local "$VERSION"

  - name: Verify Go version
    run: |
      goenv version
      go version
```

## Feature Availability

### Modern Command Features

| Feature | `goenv use` | `goenv current` | `goenv list` | Legacy Equivalent |
|---------|-------------|-----------------|--------------|-------------------|
| Set local version | ✅ | ❌ | ❌ | `goenv local` |
| Set global version | ✅ `--global` | ❌ | ❌ | `goenv global` |
| Show version source | ❌ | ✅ | ❌ | Not available |
| JSON output | ❌ | ❌ | ✅ | Not available |
| List remote versions | ❌ | ❌ | ✅ `--remote` | `goenv install --list` |
| Filter stable only | ❌ | ❌ | ✅ `--stable` | Not available |
| Bare output | ❌ | ✅ `--bare` | ✅ `--bare` | Limited |
| VS Code integration | ✅ `--vscode` | ❌ | ❌ | Not available |

## Help Output

When running `goenv --help` or `goenv help`, modern commands are shown first:

```
Usage: goenv <command> [<args>]

Modern Commands (Recommended):
  use         Set Go version (local or global)
  current     Show active version and source
  list        List installed or remote versions
  install     Install Go version
  uninstall   Remove Go version

Legacy Commands (Backward Compatibility):
  local       Set local version (use 'goenv use' instead)
  global      Set global version (use 'goenv use --global' instead)
  version     Show version (use 'goenv current' instead)
  versions    List versions (use 'goenv list' instead)

Run 'goenv help <command>' for more information.
```

**Note:** This prioritization helps new users discover the recommended commands while keeping legacy commands accessible for backward compatibility.

## Transition Timeline

- **Now**: Both modern and legacy commands work
- **Recommended**: Start using modern commands in new code
- **Future**: Legacy commands will remain for backward compatibility
- **Never**: Legacy commands will not be removed (guaranteed compatibility)

## Getting Help

- **Modern command help**: `goenv help use`, `goenv help current`, `goenv help list`
- **Legacy command help**: `goenv help local`, `goenv help global`, `goenv help versions`
- **Full documentation**: [Command Reference](./reference/COMMANDS.md)

## Summary

**For New Users:**
- Learn modern commands: `use`, `current`, `list`
- Ignore legacy commands: `local`, `global`, `versions`, `version`

**For Existing Users:**
- Legacy commands still work (no rush to update)
- Modern commands offer more features
- Update scripts gradually when convenient

**For Teams:**
- Document which style you prefer
- Modern commands recommended for consistency
- Both styles can coexist in the same workflow
