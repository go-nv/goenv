# Go Version Management: goenv

[![PR Checks Status](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml/badge.svg)](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml)
[![Latest Release](https://img.shields.io/github/v/release/go-nv/goenv.svg)](https://github.com/go-nv/goenv/releases/latest)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/go-nv/goenv/blob/main/LICENSE)
[![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?&logo=go&logoColor=white)](https://go.dev/)
[![Bash](https://img.shields.io/badge/Bash-4EAA25?logo=gnubash&logoColor=fff)](https://github.com/go-nv/goenv)
[![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black)](https://github.com/go-nv/goenv)
[![macOS](https://img.shields.io/badge/macOS-000000?logo=macos&logoColor=F0F0F0)](https://github.com/go-nv/goenv)

> **📢 Version Note:** goenv v3 is now the current stable version. v2 remains available as `goenv@2` for legacy support. See [Version Support Policy](#version-support-policy) below.

goenv aims to be as simple as possible and follow the already established
successful version management model of [pyenv](https://github.com/pyenv/pyenv) and [rbenv](https://github.com/rbenv/rbenv).

New go versions are added automatically on a daily CRON schedule.

This project was cloned from [pyenv](https://github.com/pyenv/pyenv) and modified for Go.

[![asciicast](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj.svg)](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj)

### goenv _does..._

- Let you **change the global Go version** on a per-user basis.
- Provide support for **per-project Go versions**.
- Allow you to **override the Go version** with an environment
  variable.
- Search commands from **multiple versions of Go at a time**.

### goenv compared to others:

- https://github.com/crsmithdev/goenv depends on Go,
- https://github.com/moovweb/gvm is a different approach to the problem that's modeled after `nvm`.
  `goenv` is more simplified.

---

## Version Support Policy

### Current Version (v3.x)

goenv v3 is the current stable release and is recommended for all new installations. It features:
- Complete rewrite from shell scripts to Go-based CLI for improved performance and reliability
- Full backward compatibility with v2
- Active development and feature updates

**Installation:**
```bash
# Homebrew (macOS/Linux)
brew install goenv

# Manual installation
git clone https://github.com/go-nv/goenv.git ~/.goenv
```

### Legacy Version (v2.x)

goenv v2 is maintained for legacy support with a **minimum 2-year commitment** (until April 2028 or End of Support, whichever comes first). During this period, v2 will receive:
- Security patches and critical bug fixes
- Maintenance through the `master` branch
- Support for existing production deployments

**When to use v2:**
- AWS CodeBuild and CI systems with existing v2 dependencies
- Production systems requiring additional validation time before migrating to v3
- Docker containers that haven't yet been updated to v3

**Installation:**
```bash
# Homebrew - versioned formula
brew install goenv@2 && brew link goenv@2

# Manual installation - v2 branch
git clone -b master https://github.com/go-nv/goenv.git ~/.goenv
```

**Migration:** v3 is fully backward compatible with v2. Most users can migrate immediately without changes. See our [migration guide](./docs/MIGRATION.md) for details.

**End of Support Date:** v2 support will be discontinued on or before **April 9, 2028**.

---

### Hints

#### AWS CodeBuild

The following snippet can be inserted in your buildspec.yml (or buildspec definition) for AWS CodeBuild. It's recommended to do this during the `pre_build` phase.
    
**Side Note:** if you use the below steps, please unset your golang version in the buildspec and run the installer manually.

```yaml
- (cd /root/.goenv/plugins/go-build/../.. && git pull)
```

---

## Links

- **[How It Works](./HOW_IT_WORKS.md)**
- **[Installation](./INSTALL.md)**
- **[Command Reference](./COMMANDS.md)**
- **[Environment variables](./ENVIRONMENT_VARIABLES.md)**
- **[Contributing](./CONTRIBUTING.md)**
- **[Code-of-Conduct](./CODE_OF_CONDUCT.md)**
