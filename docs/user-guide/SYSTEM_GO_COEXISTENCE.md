# System Go and goenv Coexistence

Complete guide to using goenv alongside a system-installed Go.

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Initial Setup](#initial-setup)
- [Using System Go](#using-system-go)
- [Switching Between System and goenv](#switching-between-system-and-goenv)
- [Common Scenarios](#common-scenarios)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

Many users have Go installed globally before discovering goenv. Good news: **goenv is designed to coexist with system Go installations.**

### What "System Go" Means

System Go refers to Go installed via:
- **Package managers**: `apt install golang`, `brew install go`, `choco install golang`
- **Official installers**: Downloaded from golang.org
- **Company-managed**: Enterprise deployment tools
- **OS-bundled**: Included with your Linux distribution

**Typical locations:**
- Linux: `/usr/local/go/`, `/usr/bin/go`
- macOS: `/usr/local/go/`, Homebrew's `/opt/homebrew/bin/go`
- Windows: `C:\Go\`, `C:\Program Files\Go\`

### Why You Might Keep System Go

**Valid reasons to keep both:**

1. **Legacy scripts** - System scripts depend on system Go
2. **Root-level tools** - Some system tools use system Go
3. **Fallback** - Safety net if goenv has issues
4. **Quick scripts** - One-off scripts without project setup
5. **CI/CD baseline** - System Go as CI default

**You don't need system Go** if you're starting fresh with goenv.

## How It Works

### PATH Precedence

When both are installed, PATH order determines priority:

```bash
# Correct order (goenv first)
PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:/usr/local/go/bin:/usr/bin:..."
     ↑ goenv managed              ↑ system Go

# When you type 'go', shell searches PATH left-to-right
# Finds goenv shim first → goenv controls which version
```

### The "system" Version

goenv treats system Go as special version `system`:

```bash
$ goenv list
  1.25.2
  1.24.8
* system (set by GOENV_VERSION environment variable)

# Explicitly use system Go
$ goenv use system

# Check where it points
$ goenv prefix system
/usr/local/go
```

### Environment Variable Handling

| Variable | goenv Version Active | system Active | No goenv Version Set |
|----------|----------------------|---------------|----------------------|
| `GOROOT` | `~/.goenv/versions/1.25.2/` | `/usr/local/go/` | Uses system default |
| `GOPATH` | `~/go/1.25.2/` | `~/go/` (system default) | `~/go/` |
| `PATH` | goenv shims first | goenv shims first (routes to system) | System Go (if before goenv) |

**Key insight:** Even when using `system`, goenv shims are involved (they just route to system Go).

## Initial Setup

### Step 1: Verify System Go

Before installing goenv:

```bash
# Check system Go
which go
# Output: /usr/local/go/bin/go (or similar)

go version
# Output: go version go1.24.3 linux/amd64

# Check GOROOT
go env GOROOT
# Output: /usr/local/go

# Note these paths - you'll verify goenv doesn't break them
```

### Step 2: Install goenv

```bash
# Linux/macOS
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Windows (PowerShell)
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex
```

### Step 3: Configure Shell (Critical!)

**Correct PATH order** (goenv before system):

```bash
# ~/.bashrc or ~/.zshrc
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:$PATH"  # ← goenv first!
eval "$(goenv init -)"

# Restart shell
exec $SHELL
```

**Windows (PowerShell):**

```powershell
# $PROFILE
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\shims;$env:GOENV_ROOT\bin;$env:PATH"
goenv init - powershell | Out-String | Invoke-Expression
```

### Step 4: Verify Coexistence

```bash
# Check PATH order
echo $PATH | tr ':' '\n' | head -10

# Should show:
# /Users/you/.goenv/shims       ← goenv first
# /Users/you/.goenv/bin
# /usr/local/go/bin              ← system go still present
# ...

# Check goenv sees system Go
goenv list
# Output should include:
#   system

# Verify system version detected
goenv versions system
# Output: system (set by /usr/local/go)
```

## Using System Go

### Explicit system Selection

```bash
# Use system Go for current directory
goenv use system

# Verify
goenv current
# Output: system (set by /path/to/.go-version)

which go
# Output: /Users/you/.goenv/shims/go (shim routing to system)

go version
# Output: go version go1.24.3 linux/amd64 (system version)

go env GOROOT
# Output: /usr/local/go
```

### System Go as Global Default

```bash
# Set system as global default
goenv use system --global

# All projects without .go-version use system Go
goenv current
# Output: system (set by /Users/you/.goenv/version)
```

### System Go for Specific Projects

```bash
cd ~/legacy-project
echo "system" > .go-version

# Now this project always uses system Go
goenv current
# Output: system (set by /Users/you/legacy-project/.go-version)
```

## Switching Between System and goenv

### Quick Switching

```bash
# Project A: Use goenv-managed Go
cd ~/project-a
goenv use 1.25.2
go version
# Output: go version go1.25.2 linux/amd64

# Project B: Use system Go
cd ~/project-b
goenv use system
go version
# Output: go version go1.24.3 linux/amd64 (system)

# No conflicts - automatic per-directory switching!
```

### Environment Variable Override

```bash
# Force specific version regardless of .go-version
GOENV_VERSION=1.25.2 go version

# Force system Go
GOENV_VERSION=system go version

# Temporary shell session
GOENV_VERSION=system bash
go version  # Uses system Go
exit        # Back to normal
```

### Bypassing goenv Completely

```bash
# Use system Go directly (bypass goenv)
/usr/local/go/bin/go version

# Or temporarily remove goenv from PATH
PATH="/usr/local/bin:/usr/bin" go version
```

## Common Scenarios

### Scenario 1: Development vs. Production

**Development:** Use goenv for multiple versions

```bash
# Dev project with latest
cd ~/dev/new-feature
goenv use 1.25.2

# Dev project with older version
cd ~/dev/legacy-support
goenv use 1.23.2
```

**Production:** Use system Go for stability

```bash
# Production deployment
cd /var/www/app
echo "system" > .go-version

# Ensures deployment uses vetted system Go
```

### Scenario 2: Personal vs. System Tools

**Personal development:**

```bash
# Your projects use goenv
cd ~/projects/myapp
goenv use 1.25.2
go install ./...  # Installs to ~/go/1.25.2/bin/
```

**System tools:**

```bash
# System scripts use system Go
sudo /usr/local/go/bin/go run /usr/local/bin/system-tool.go

# Or explicitly with system version
GOENV_VERSION=system go run system-script.go
```

### Scenario 3: CI/CD Flexibility

**Option A: Use system Go in CI**

```yaml
# .gitlab-ci.yml
build:
  script:
    - export GOENV_VERSION=system
    - go build
```

**Option B: Use goenv version in CI**

```yaml
build:
  script:
    - goenv install $(cat .go-version)
    - goenv use $(cat .go-version)
    - go build
```

**Option C: Hybrid approach**

```yaml
# Use system for cache key, goenv for actual build
cache:
  key: $CI_COMMIT_REF_SLUG-system
  paths:
    - .cache

build:
  script:
    - goenv use 1.25.2  # Explicit version for build
    - go build
```

### Scenario 4: Team Collaboration

**Team member 1:** Uses goenv

```bash
cd shared-project
goenv use 1.25.2
go build
```

**Team member 2:** Uses system Go

```bash
cd shared-project
go build  # Uses system Go 1.25.2
```

**Both work** if Go versions match. Document expected version:

```bash
# Document in README
echo "Requires Go 1.25.2 or later"
echo "1.25.2" > .go-version  # goenv users get automatic version
```

## Troubleshooting

### System Go Being Used Instead of goenv

**Problem:**
```bash
$ goenv use 1.25.2
$ go version
go version go1.24.3 linux/amd64  # ← Wrong! Should be 1.25.2
```

**Diagnosis:**
```bash
which go
# Shows: /usr/local/go/bin/go ← Wrong! Should be shim

echo $PATH | tr ':' '\n' | head -5
# If system Go appears before goenv → PATH order wrong
```

**Solution:**
```bash
# Fix PATH order in shell config
# Ensure goenv comes FIRST

# ~/.bashrc
export PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:$PATH"

# Restart shell
exec $SHELL

# Verify fix
which go
# Should show: /Users/you/.goenv/shims/go
```

### GOROOT/GOPATH Conflicts

**Problem:**
```bash
$ goenv use 1.25.2
$ go env GOROOT
/usr/local/go  # ← Wrong! Should be ~/.goenv/versions/1.25.2
```

**Diagnosis:**
```bash
# Check for hardcoded GOROOT
env | grep GOROOT
# If shows: GOROOT=/usr/local/go ← Problem

# Check who set it
grep -r "GOROOT" ~/.bashrc ~/.zshrc ~/.profile
```

**Solution:**
```bash
# Remove hardcoded GOROOT from shell config
# Let goenv manage it

# If in ~/.bashrc:
# export GOROOT=/usr/local/go  ← DELETE THIS

# Restart shell
exec $SHELL

# Verify fix
goenv use 1.25.2
go env GOROOT
# Should show: /Users/you/.goenv/versions/1.25.2
```

### Tools Installed in Wrong Location

**Problem:**
```bash
$ goenv use 1.25.2
$ go install example.com/tool@latest
$ tool
command not found  # ← Tool not in PATH
```

**Diagnosis:**
```bash
# Check where tool was installed
go env GOPATH
# Should show: /Users/you/go/1.25.2

# Check if binary exists
ls ~/go/1.25.2/bin/tool
# If exists: PATH issue
# If not exists: Installation issue

# Check if tool in system GOPATH
ls /usr/local/go/bin/tool  # Installed to wrong place
```

**Solution:**
```bash
# Ensure goenv shims in PATH
export PATH="$GOENV_ROOT/shims:$PATH"

# Regenerate shims
goenv rehash

# Verify GOPATH
goenv use 1.25.2
go env GOPATH
# Should show versioned path: ~/go/1.25.2

# Reinstall tool
go install example.com/tool@latest

# Check installation
which tool
# Should show: /Users/you/.goenv/shims/tool
```

### VS Code Using Wrong Go

**Problem:** VS Code uses system Go instead of goenv version

**Diagnosis:**
```bash
# Check VS Code settings
cat .vscode/settings.json

# If shows:
# "go.goroot": "/usr/local/go"  ← Hardcoded to system
```

**Solution:**
```bash
# Use goenv VS Code integration
goenv vscode setup

# Or manually update settings
cat > .vscode/settings.json <<EOF
{
  "go.goroot": "\${env:GOROOT}",
  "go.gopath": "\${env:GOPATH}"
}
EOF

# Reload VS Code
# Cmd+Shift+P → "Developer: Reload Window"
```

[VS Code Integration Guide](./user-guide/VSCODE_INTEGRATION.md)

### System Go Removed But goenv Broken

**Problem:** Removed system Go, now `goenv use system` fails

**Expected:** This is normal if you removed system Go

**Solution:**
```bash
# Option 1: Install Go via goenv instead
goenv install 1.25.2
goenv use 1.25.2 --global

# Option 2: Reinstall system Go if needed
# apt install golang
# brew install go

# Option 3: Update .go-version files using system
find . -name .go-version -exec grep -l "system" {} \;
# Replace "system" with actual version number
```

## Best Practices

### Decision Matrix: System Go vs goenv-managed

**Use system Go when:**

| Scenario | Why System Go | Example |
|----------|--------------|---------|
| **Legacy system scripts** | Already expect `/usr/local/go/bin/go` | Cron jobs, system monitoring |
| **Corporate mandate** | IT policy requires system Go | Enterprise environments |
| **CI/CD pre-installed** | Go already in CI image | Docker images with Go pre-installed |
| **OS integration** | OS packages depend on system Go | Some Linux distro tools |
| **Root-level tools** | System admin tools using Go | Infrastructure tools |
| **No version control needed** | One Go version is sufficient | Single-language company |

**Use goenv-managed Go when:**

| Scenario | Why goenv-managed | Example |
|----------|-------------------|---------|
| **Multiple projects** | Need different Go versions | Maintaining v1.21 and v1.25 projects |
| **Development** | Full version control | Active development work |
| **Testing compatibility** | Test against multiple versions | Library maintainers |
| **Team consistency** | `.go-version` ensures same version | Team collaboration |
| **Latest features** | Access to newest releases | Early adopter projects |
| **Tool isolation** | Keep tools separate per version | gopls, golangci-lint per project |
| **Reproducible builds** | Exact version in version control | Production deployments |

**Quick decision flowchart:**

```
Do you need multiple Go versions?
├─ Yes → Use goenv-managed
└─ No
   └─ Is this a development project?
      ├─ Yes → Use goenv-managed (better control)
      └─ No
         └─ Is Go required by system tools/scripts?
            ├─ Yes → Use system Go
            └─ No → Either works (goenv recommended)
```

### 1. Clear Documentation

Document which Go is expected:

```markdown
# README.md

## Go Version

This project uses Go 1.25.2.

- **With goenv:** `goenv use 1.25.2`
- **Without goenv:** Install Go 1.25.2 from golang.org
- **System Go:** `goenv use system` (if system Go is 1.25.2)
```

### 2. Explicit Version Selection

Don't rely on global defaults:

```bash
# ✅ Good - Explicit .go-version file
echo "1.25.2" > .go-version

# ❌ Avoid - Relying on global setting
# (Team members might have different globals)
```

### 3. PATH Hygiene

Keep PATH clean:

```bash
# ✅ Good - Clear order
export PATH="$GOENV_ROOT/shims:$GOENV_ROOT/bin:$PATH"

# ❌ Avoid - Multiple Go paths
export PATH="/usr/local/go/bin:$PATH"
export PATH="$GOENV_ROOT/shims:$PATH"  # ← System Go comes first!
```

### 4. Environment Variable Discipline

Let goenv manage Go-related variables:

```bash
# ✅ Good - Only set GOENV_ROOT
export GOENV_ROOT="$HOME/.goenv"

# ❌ Avoid - Conflicting variables
export GOROOT="/usr/local/go"  # ← Let goenv manage this
export GOPATH="$HOME/go"       # ← Let goenv manage this
```

### 5. Testing Both Paths

Test your project works with both:

```bash
# Test with goenv version
goenv use 1.25.2
go test ./...

# Test with system Go
goenv use system
go test ./...

# Ensures compatibility
```

### 6. CI/CD Clarity

Be explicit in CI:

```yaml
# ✅ Good - Explicit about which Go
script:
  - export GOENV_VERSION=system
  - go version  # Logs which version
  - go build

# Or
script:
  - goenv install 1.25.2
  - goenv use 1.25.2
  - go version
  - go build
```

### 7. Gradual Migration

Migrate incrementally:

```bash
# Week 1: Install goenv, keep system as default
goenv use system --global

# Week 2: Try goenv for new projects
cd new-project
goenv use 1.25.2

# Week 3: Migrate active projects
cd active-project
goenv use 1.25.2

# Week 4: System Go only for legacy
# Most projects use goenv now
```

### 8. Recommendation Summary

**For new users:**

1. **Keep system Go installed** - Useful as fallback, no need to remove
2. **Configure PATH correctly** - goenv shims must come first
3. **Use goenv for development** - Better control and isolation
4. **Use system Go for system tools** - If needed by OS/infrastructure
5. **Document your choice** - Make it clear in project README

**For teams:**

1. **Standardize on goenv** - Use `.go-version` files for consistency
2. **Allow system Go as fallback** - For emergency situations
3. **Document local requirements** - README should specify approach
4. **Test both paths** - Ensure project works with goenv and system Go
5. **CI/CD flexibility** - Support both installation methods

**For enterprises:**

1. **Respect corporate Go installations** - Use `goenv use system` when required
2. **Supplement with goenv** - For development flexibility
3. **Isolate by project** - Use `.go-version` files per project
4. **Security updates** - Monitor both system Go and goenv versions
5. **Audit trail** - Use `goenv doctor` to verify configuration

## When to Remove System Go

**Safe to remove if:**

- ✅ All projects use goenv
- ✅ No system scripts depend on system Go
- ✅ No company policy requires system Go
- ✅ You've tested everything works without it

**How to remove safely:**

```bash
# 1. Test everything works with goenv
for project in ~/projects/*/; do
  cd "$project"
  if [ -f .go-version ]; then
    goenv use $(cat .go-version)
    go build || echo "Failed: $project"
  fi
done

# 2. Remove system Go
# Linux
sudo apt remove golang
sudo rm -rf /usr/local/go

# macOS
brew uninstall go
sudo rm -rf /usr/local/go

# Windows
# Use "Add or Remove Programs"

# 3. Update goenv version files
# Replace "system" with actual versions
find ~ -name .go-version -exec sed -i 's/system/1.25.2/' {} \;

# 4. Set new global default
goenv use 1.25.2 --global
```

## Summary

**Key Takeaways:**

1. ✅ **goenv and system Go coexist** - By design, not accident
2. ✅ **PATH order matters** - goenv shims must come first
3. ✅ **"system" version** - Explicitly use system Go when needed
4. ✅ **Per-project control** - `.go-version` files override global
5. ✅ **Environment variables** - Let goenv manage GOROOT/GOPATH
6. ✅ **Both have use cases** - System for stability, goenv for flexibility

**Quick Check:**

```bash
# Verify healthy coexistence
echo $PATH | grep -o "[^:]*go[^:]*" | head -5
# Should show goenv paths before system paths

goenv list
# Should include "system" if system Go present

goenv use 1.25.2 && go env GOROOT
# Should show goenv path

goenv use system && go env GOROOT
# Should show system path
```

## See Also

- [FAQ - System Go Question](./FAQ.md#i-already-have-go-installed-globally-will-goenv-interfere)
- [How It Works](./user-guide/HOW_IT_WORKS.md)
- [GOPATH Integration](./advanced/GOPATH_INTEGRATION.md)
- [Platform Support](../reference/PLATFORM_SUPPORT.md)
- [Troubleshooting](../advanced/CACHE_TROUBLESHOOTING.md)
