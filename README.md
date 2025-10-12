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
- Allow you to **override the Go version** with an environment
  variable.
- Search commands from **multiple versions of Go at a time**.

### goenv compared to others:

- https://github.com/crsmithdev/goenv depends on Go,
- https://github.com/moovweb/gvm is a different approach to the problem that's modeled after `nvm`.
  `goenv` is more simplified.

**New in 2.x**: This version is a complete rewrite in Go, offering:

- **Dynamic version fetching** - Always up-to-date without manual updates
- **Offline support** - Works without internet via intelligent caching
- **Better performance** - Native Go binary vs bash scripts
- **Cross-platform** - Single binary for all supported platforms

---

### Hints

#### AWS CodeBuild

The following snippet can be inserted in your buildspec.yml (or buildspec definition) for AWS CodeBuild. It's recommended to do this during the `pre_build` phase.

**Side Note:** if you use the below steps, please unset your golang version in the buildspec and run the installer manually.

```yaml
- (cd /root/.goenv && git pull)
```

---

## 📖 Documentation

**[📚 Complete Documentation](./docs/)** - Comprehensive guides and references

### Quick Links

- **[Installation Guide](./docs/user-guide/INSTALL.md)** - Get started with goenv
- **[How It Works](./docs/user-guide/HOW_IT_WORKS.md)** - Understanding goenv's internals
- **[Command Reference](./docs/reference/COMMANDS.md)** - Complete CLI documentation
- **[Environment Variables](./docs/reference/ENVIRONMENT_VARIABLES.md)** - Configuration options
- **[Smart Caching](./docs/advanced/SMART_CACHING.md)** - Intelligent version caching
- **[Contributing](./docs/CONTRIBUTING.md)** - How to contribute
- **[Code of Conduct](./docs/CODE_OF_CONDUCT.md)** - Community guidelines
- **[Changelog](./docs/CHANGELOG.md)** - Version history
