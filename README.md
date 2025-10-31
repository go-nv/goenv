# Go Version Management: goenv

[![PR Checks Status](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml/badge.svg)](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml)
[![Latest Release](https://img.shields.io/github/v/release/go-nv/goenv.svg)](https://github.com/go-nv/goenv/releases/latest)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/go-nv/goenv/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?&logo=go&logoColor=white)](https://go.dev/)
[![Bash](https://img.shields.io/badge/Bash-4EAA25?logo=gnubash&logoColor=fff)](https://github.com/go-nv/goenv)
[![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black)](https://github.com/go-nv/goenv)
[![macOS](https://img.shields.io/badge/macOS-000000?logo=macos&logoColor=F0F0F0)](https://github.com/go-nv/goenv)

goenv aims to be as simple as possible and follow the already established
successful version management model of [pyenv](https://github.com/pyenv/pyenv) and [rbenv](https://github.com/rbenv/rbenv).

**🎉 Now 100% Go-based with dynamic version fetching!** No more static version files or manual updates needed.

This project was originally cloned from [pyenv](https://github.com/pyenv/pyenv), modified for Go, and has now been completely rewritten in Go for better performance and maintainability.

[![asciicast](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj.svg)](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj)

### goenv _does..._

- Let you **change the global Go version** on a per-user basis.
- Provide support for **per-project Go versions**.
- **Smart version discovery** - automatically detects versions from `.go-version` or `go.mod`
- **Go.mod toolchain support** - respects Go 1.21+ toolchain directives with smart precedence
- Allow you to **override the Go version** with an environment
  variable.
- Search commands from **multiple versions of Go at a time**.
- Provide **tab completion** for bash, zsh, fish, and PowerShell.
- **Automatically rehash** after `go install` - tools available immediately (can be disabled with `--no-rehash` flag)
- **Enhanced diagnostics** (`goenv doctor`) with interactive fix mode and 18 issue types
- **Quick status check** (`goenv status`) for installation health overview
- **Version usage analysis** (`goenv versions --used`) - scan projects to see which versions are in use
- **Version lifecycle tracking** (`goenv info`) with EOL detection and upgrade recommendations
- **Version comparison** (`goenv compare`) for side-by-side analysis
- **First-time setup wizard** (`goenv setup`) for automatic shell and IDE configuration
- **Interactive beginner guide** (`goenv get-started`) for new users
- **Command discovery** (`goenv explore`) to find commands by intent or category
- **Self-update capability** (`goenv update`) for both git and binary installations
- **Tool management** (`goenv tools`) - comprehensive tool management across Go versions:
  - Install/uninstall tools across all versions with `--all` flag
  - Check tool consistency with `goenv tools status`
  - Find outdated tools with `goenv tools outdated`
  - Auto-install common tools with `goenv tools default`
  - Sync tools between versions with `goenv tools sync`
- **Version aliases** (`goenv alias`) - create convenient shorthand names for versions
- **VS Code integration** (`goenv vscode`) - sync Go settings with workspace-relative paths and security validation

### goenv compared to others:

- https://github.com/crsmithdev/goenv depends on Go,
- https://github.com/moovweb/gvm is a different approach to the problem that's modeled after `nvm`.
  `goenv` is more simplified.

**New in 2.x**: This version is a complete rewrite in Go, offering:

- **Dynamic version fetching** - Always up-to-date without manual updates
- **Offline support** - Works without internet via intelligent caching
- **Better performance** - Native Go binary vs bash scripts
- **Cross-platform** - Single binary for all supported platforms
- **Auto-rehash** - Installed tools work immediately without manual intervention (configurable for CI/CD)

---

### Hints

#### AWS CodeBuild

The following snippet can be inserted in your buildspec.yml (or buildspec definition) for AWS CodeBuild. It's recommended to do this during the `pre_build` phase.

**Side Note:** if you use the below steps, please unset your golang version in the buildspec and run the installer manually.

```yaml
- (cd /root/.goenv && git pull)
```

---

## 🚀 Quick Start

### Option 1: Binary Installation (Recommended - No Go Required!)

The fastest way to get started. Download a pre-built binary - no Go installation needed!

```bash
# Automatic install (Linux/macOS)
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Or download manually from releases:
# https://github.com/go-nv/goenv/releases/latest
```

```powershell
# Automatic install (Windows)
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex
```

Then add to your shell config:

```bash
# Bash (~/.bash_profile or ~/.bashrc)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Option 2: Package Manager (macOS)

**Homebrew**:

```bash
brew install goenv
```

Then add to your shell config (see Option 1 above for shell configuration).

### Option 3: Git Clone (Requires Go to Build)

For contributors or those who want the latest development version:

```bash
# 1. Clone goenv
git clone https://github.com/go-nv/goenv.git ~/.goenv

# 2. Build (requires Go)
cd ~/.goenv && make build

# 3. Add to your shell config (~/.bashrc, ~/.zshrc, etc.)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"

# 4. Restart your shell
exec $SHELL
```

### Next Steps (All Options)

```bash
# Enable tab completion (optional but recommended)
goenv completion --install

# Restart your shell
exec $SHELL

# Install and set a Go version (one command!)
goenv use 1.22.0 --global

# Or install first, then set
goenv install 1.22.0

# 💡 Shorthand: goenv 1.22.0 is an alias for goenv use 1.22.0
goenv 1.22.0  # Same as: goenv use 1.22.0

# Verify
go version

# Install tools (automatically isolated per version)
go install golang.org/x/tools/cmd/goimports@latest
goenv rehash  # Makes tools available as shims
```

**💡 Tools installed with `go install` are isolated per Go version:**

- Tools install to `$HOME/go/{version}/bin/` (e.g., `~/go/1.22.0/bin/goimports`)
- Running `goenv rehash` creates shims that automatically use the right version
- Switch Go versions → tools switch too (no conflicts between versions)
- See [GOPATH Integration](docs/advanced/GOPATH_INTEGRATION.md) for complete details

See [Installation Guide](docs/user-guide/INSTALL.md) for platform-specific setup.

---

## 🪝 Hooks System

goenv includes a powerful hooks system that lets you automate actions at key points in the goenv lifecycle. Hooks are **declarative**, **safe**, and **cross-platform**.

### Quick Example

```bash
# Generate configuration template
goenv hooks init

# Edit ~/.goenv/hooks.yaml
# Set enabled: true and acknowledged_risks: true

# Example: Log installations and send Slack notifications
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Installed Go {version} at {timestamp}"

    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK
        body: '{"text": "✅ Go {version} installed"}'
```

### Available Actions

- **`log_to_file`** - Write audit logs and track installations
- **`http_webhook`** - Send notifications to Slack, Discord, or custom APIs
- **`notify_desktop`** - Display native desktop notifications
- **`check_disk_space`** - Verify sufficient space before operations
- **`set_env`** - Set environment variables dynamically

### Hook Points

Execute actions at 8 different lifecycle points:

- `pre_install` / `post_install` - Before/after installing Go versions
- `pre_uninstall` / `post_uninstall` - Before/after removing Go versions
- `pre_exec` / `post_exec` - Before/after executing Go commands
- `pre_rehash` / `post_rehash` - Before/after regenerating shims

### Commands

```bash
goenv hooks init       # Generate configuration template
goenv hooks list       # Show available actions and hook points
goenv hooks validate   # Check configuration for errors
goenv hooks test       # Dry-run hooks without executing
```

**[📖 Complete Hooks Documentation](./docs/HOOKS.md)** - Examples, use cases, and detailed guides

---

## 📖 Documentation

**[📚 Complete Documentation](./docs/)** - Comprehensive guides and references

### Quick Links

#### Getting Started
- **[Installation Guide](./docs/user-guide/INSTALL.md)** - Get started with goenv ⭐ **NEW: Windows FAQ**
- **[Quick Reference](./docs/QUICK_REFERENCE.md)** - One-page cheat sheet ⭐ **NEW**
- **[FAQ](./docs/FAQ.md)** - Frequently asked questions ⭐ **NEW**
- **[What's New in Docs](./docs/WHATS_NEW_DOCUMENTATION.md)** - Recent documentation improvements ⭐ **NEW**
- **[How It Works](./docs/user-guide/HOW_IT_WORKS.md)** - Understanding goenv's internals
- **[VS Code Integration](./docs/user-guide/VSCODE_INTEGRATION.md)** - Setting up VS Code with goenv

#### Reference
- **[Command Reference](./docs/reference/COMMANDS.md)** - Complete CLI documentation ⭐ **NEW: --force guidance, vscode setup**
- **[Environment Variables](./docs/reference/ENVIRONMENT_VARIABLES.md)** - Configuration options
- **[Platform Support Matrix](./docs/PLATFORM_SUPPORT.md)** - OS and architecture compatibility ⭐ **NEW**
- **[Modern Commands Guide](./docs/MODERN_COMMANDS.md)** - use/current/list vs legacy commands ⭐ **NEW**
- **[JSON Output Guide](./docs/JSON_OUTPUT_GUIDE.md)** - Automation and CI/CD integration ⭐ **NEW**

#### Advanced Topics
- **[CI/CD Integration](./docs/CI_CD_GUIDE.md)** - Best practices for pipelines ⭐ **NEW: Offline mode, Windows examples**
- **[Hooks System Quick Start](./docs/HOOKS_QUICKSTART.md)** - 5-minute hooks setup ⭐ **NEW**
- **[Hooks System (Full)](./docs/HOOKS.md)** - Complete hooks documentation ⭐ **NEW: Security best practices**
- **[Compliance Use Cases](./docs/COMPLIANCE_USE_CASES.md)** - SOC 2, ISO 27001, SBOM ⭐ **NEW**
- **[Smart Caching](./docs/advanced/SMART_CACHING.md)** - Intelligent version caching ⭐ **NEW: Network reliability**
- **[What NOT to Sync](./docs/advanced/WHAT_NOT_TO_SYNC.md)** - Sharing goenv across machines
- **[Cache Troubleshooting](./docs/CACHE_TROUBLESHOOTING.md)** - Cache issues and migration ⭐ **NEW**
- **[System Go Coexistence](./docs/SYSTEM_GO_COEXISTENCE.md)** - Using with system-installed Go ⭐ **NEW**

#### For Contributors
- **[Contributing](./docs/CONTRIBUTING.md)** - How to contribute (code & documentation)
- **[Documentation Review Checklist](./docs/DOCUMENTATION_REVIEW_CHECKLIST.md)** - Doc quality checklist ⭐ **NEW**
- **[Testing Roadmap](./docs/TESTING_ROADMAP.md)** - Test coverage gaps and priorities 🆕

#### Other
- **[Code of Conduct](./docs/CODE_OF_CONDUCT.md)** - Community guidelines
- **[Changelog](./docs/CHANGELOG.md)** - Version history
