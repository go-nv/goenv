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
- **Diagnostic tool** (`goenv doctor`) for troubleshooting installation issues
- **Self-update capability** (`goenv update`) for both git and binary installations
- **Default tools** (`goenv default-tools`) - auto-install common tools with new Go versions
- **Update tools** (`goenv update-tools`) - keep all installed Go tools current
- **Migrate tools** (`goenv migrate-tools`) - copy tools when upgrading Go versions
- **Version aliases** (`goenv alias`) - create convenient shorthand names for versions

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

# Install a Go version
goenv install 1.22.0

# Set it as global
goenv global 1.22.0

# Verify
go version
```

See [Installation Guide](docs/user-guide/INSTALL.md) for more details.

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

- **[Installation Guide](./docs/user-guide/INSTALL.md)** - Get started with goenv
- **[How It Works](./docs/user-guide/HOW_IT_WORKS.md)** - Understanding goenv's internals
- **[VS Code Integration](./docs/user-guide/VSCODE_INTEGRATION.md)** - Setting up VS Code with goenv
- **[Command Reference](./docs/reference/COMMANDS.md)** - Complete CLI documentation
- **[Environment Variables](./docs/reference/ENVIRONMENT_VARIABLES.md)** - Configuration options
- **[Hooks System](./docs/HOOKS.md)** - Automate actions with declarative hooks
- **[Smart Caching](./docs/advanced/SMART_CACHING.md)** - Intelligent version caching
- **[Contributing](./docs/CONTRIBUTING.md)** - How to contribute
- **[Code of Conduct](./docs/CODE_OF_CONDUCT.md)** - Community guidelines
- **[Changelog](./docs/CHANGELOG.md)** - Version history
