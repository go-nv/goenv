# Interactive Mode Guide

## Overview

goenv provides flexible interactivity to suit different use cases: from fully automated CI/CD pipelines to guided interactive workflows for new users.

## Three Levels of Interactivity

### 1. Non-Interactive Mode (Automation/CI)

**When to use**: CI/CD pipelines, automation scripts, batch operations

**How to enable**:
```bash
# Method 1: Use --yes flag
goenv install 1.23.0 --yes
goenv uninstall 1.22.0 --yes

# Method 2: Use environment variable
export GOENV_ASSUME_YES=1
goenv install 1.23.0

# Method 3: Quiet mode (suppresses all output)
goenv install 1.23.0 --quiet

# Automatic: CI environments are auto-detected
# (CI=true, GITHUB_ACTIONS=true, etc.)
```

**Behavior**:
- No prompts or confirmations
- Auto-confirms all operations
- Fails fast on errors
- Minimal or no output

**Examples**:
```bash
# CI pipeline
goenv install latest --yes
goenv global latest --yes

# Batch uninstall
for v in 1.20.0 1.21.0; do
  goenv uninstall $v --yes
done

# Silent background operation
goenv install 1.23.0 --quiet --yes
```

---

### 2. Minimal Interactive Mode (Default)

**When to use**: Normal daily usage, terminal sessions

**How to enable**: Default behavior (no flags needed)

**Behavior**:
- Prompts only for critical/destructive operations
- Confirms before deleting files
- Shows progress and results
- Balanced UX

**Examples**:
```bash
# Confirms before deleting
$ goenv uninstall 1.22.0
Really uninstall Go 1.22.0? [y/N] y
Uninstalled Go 1.22.0

# Shows progress
$ goenv install 1.23.0
Downloading go1.23.0.linux-amd64.tar.gz...
[=================>] 100%
Installed Go 1.23.0

# Offers retry on failure
$ goenv install 1.23.0
Error: download failed
Retry? [Y/n] y
Downloading...
```

---

### 3. Guided Interactive Mode (Learning/Exploration)

**When to use**: First-time users, learning, exploring features

**How to enable**:
```bash
goenv install --interactive
goenv doctor --interactive
goenv use --interactive
```

**Behavior**:
- Helpful prompts and explanations
- Suggests solutions and alternatives
- Step-by-step guidance
- Educational messages

**Examples**:
```bash
# Guided installation
$ goenv install --interactive

📦 Install Go Version

Which version would you like to install?
  1) latest (1.23.4)
  2) 1.22.9 (previous stable)
  3) Enter specific version
  
Select [1-3]: 1

Would you like to set this as your global default? [Y/n] y

✓ Installed Go 1.23.4
✓ Set as global default
✓ Rehashed shims

Next steps:
  • Run 'go version' to verify
  • Run 'goenv list' to see installed versions
  • Run 'goenv explain' to understand version selection
```

---

## Global Flags

### --interactive
Enable guided mode with helpful prompts and suggestions.

```bash
goenv install --interactive    # Shows version selection menu
goenv doctor --interactive     # Offers to fix issues automatically
goenv use --interactive        # Guides through version selection
```

### --yes / -y
Auto-confirm all prompts (non-interactive mode).

```bash
goenv uninstall 1.22.0 --yes   # No confirmation prompt
goenv install latest -y        # Short form
```

### --quiet / -q
Suppress all progress output (only show errors).

```bash
goenv install 1.23.0 --quiet   # Silent operation
goenv list -q                  # Minimal output
```

**Flag Combinations**:
```bash
# Silent automation
goenv install 1.23.0 --yes --quiet

# Guided but quiet progress
goenv install --interactive --quiet
```

---

## Environment Variables

### GOENV_ASSUME_YES
Auto-confirm all prompts globally.

```bash
# Enable for entire session
export GOENV_ASSUME_YES=1
goenv uninstall 1.22.0  # No prompt
goenv install 1.23.0    # No prompt
```

### CI Environment Detection
goenv automatically enables non-interactive mode when running in CI.

**Detected CI environments**:
- `CI=true`
- `GITHUB_ACTIONS=true`
- `GITLAB_CI=true`
- `CIRCLECI=true`
- `JENKINS_HOME` set
- `TRAVIS=true`

**In CI**:
```yaml
# No need for --yes flag
- run: goenv install 1.23.0
- run: goenv global 1.23.0
```

---

## Command-Specific Behavior

### install
```bash
# Default: shows progress
goenv install 1.23.0

# Guided: offers version selection
goenv install --interactive

# Non-interactive: silent, auto-retry
goenv install 1.23.0 --yes
```

### uninstall
```bash
# Default: confirms before deleting
goenv uninstall 1.22.0
# Prompt: "Really uninstall Go 1.22.0? [y/N]"

# Non-interactive: no confirmation
goenv uninstall 1.22.0 --yes
```

### use
```bash
# Default: installs if needed (with prompt)
goenv use 1.23.0
# Prompt: "Go 1.23.0 not installed. Install now? [Y/n]"

# Guided: helps choose version
goenv use --interactive
# Shows: installed versions, suggests latest, offers info

# Non-interactive: fails if not installed
goenv use 1.23.0 --yes
# Error: "Go 1.23.0 not installed"
```

### doctor
```bash
# Default: shows issues
goenv doctor

# Interactive: offers to fix
goenv doctor --interactive
# Prompt: "Fix this issue automatically? [Y/n]"

# Non-interactive: just report
goenv doctor --yes
```

---

## Best Practices

### For Daily Use
Use defaults (minimal interactive mode):
```bash
goenv install 1.23.0     # Shows progress, confirms if needed
goenv use 1.23.0         # Installs if needed
```

### For Scripts
Use --yes flag:
```bash
#!/bin/bash
goenv install 1.23.0 --yes
goenv global 1.23.0 --yes
```

### For CI/CD
Rely on auto-detection or set GOENV_ASSUME_YES:
```yaml
env:
  GOENV_ASSUME_YES: 1
steps:
  - run: goenv install 1.23.0
  - run: goenv global 1.23.0
```

### For Learning
Use --interactive flag:
```bash
goenv install --interactive  # Learn about versions
goenv doctor --interactive   # Learn about issues
goenv explain               # Understand version resolution
```

---

## Troubleshooting

### "Operation cancelled" in CI
**Problem**: Command prompts in CI even though it shouldn't

**Solution**:
```bash
# Use explicit flags
goenv install 1.23.0 --yes

# Or set environment
export GOENV_ASSUME_YES=1
```

### Want more verbosity in scripts
**Problem**: --quiet is too quiet, want some output

**Solution**: Use defaults without --quiet
```bash
goenv install 1.23.0 --yes  # Shows progress but auto-confirms
```

### Unexpected prompts in automation
**Problem**: Script hangs waiting for input

**Solution**: Always use --yes or GOENV_ASSUME_YES=1 in automation
```bash
export GOENV_ASSUME_YES=1
# All commands now non-interactive
```

---

## Examples by Use Case

### New User Learning
```bash
# Use guided mode
goenv install --interactive
goenv doctor --interactive
goenv explain
goenv status
```

### Daily Development
```bash
# Use defaults
goenv install 1.23.0
goenv use 1.23.0
goenv list
```

### Automation Script
```bash
#!/bin/bash
set -e
export GOENV_ASSUME_YES=1

goenv install 1.23.0
goenv global 1.23.0
goenv tools install golangci-lint
```

### CI Pipeline
```yaml
name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      # goenv auto-detects CI
      - run: goenv install 1.23.0
      - run: goenv global 1.23.0
      - run: go test ./...
```

### Batch Operations
```bash
# Uninstall old versions
for v in $(goenv list | grep 1.20); do
  goenv uninstall $v --yes
done

# Install multiple versions
for v in 1.22.0 1.23.0; do
  goenv install $v --yes --quiet
done
```

---

## Summary

| Mode | When | Flags | Behavior |
|------|------|-------|----------|
| **Non-Interactive** | CI, automation | --yes, --quiet | No prompts, auto-confirm |
| **Minimal** | Daily use | (default) | Critical prompts only |
| **Guided** | Learning | --interactive | Helpful, educational |

**Key Takeaway**: goenv adapts to your context - automated in CI, helpful in terminal, guided when learning.

---

## See Also

- `goenv explain` - Understand version resolution
- `goenv doctor --help` - See all diagnostic options
- `goenv --help` - Full command reference
- [Design Doc](../docs/design/INTERACTIVE_MODE_EXPANSION.md) - Technical details
