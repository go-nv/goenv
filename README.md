# Go Version Management: goenv

[![PR Checks Status](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml/badge.svg)](https://github.com/go-nv/goenv/actions/workflows/pr_checks.yml)
[![Latest Release](https://img.shields.io/github/v/release/go-nv/goenv.svg)](https://github.com/go-nv/goenv/releases/latest)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/go-nv/goenv/blob/main/LICENSE)

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

### Hints

#### AWS CodeBuild

The following snippet can be inserted in your buildspec.yml (or buildspec definition) for AWS CodeBuild. It's recommended to do this during the `pre_build` phase.
    
**Side Note:** if you use the below steps, please unset your golang version in the buildspec and run the installer manually.

```yaml
- BUILD_DIR=$PWD
- cd /root/.goenv/plugins/go-build/../.. && git pull && cd -
- cd $BUILD_DIR
```

---

## Links

- **[How It Works](./HOW_IT_WORKS.md)**
- **[Installation](./INSTALL.md)**
- **[Command Reference](./COMMANDS.md)**
- **[Environment variables](./ENVIRONMENT_VARIABLES.md)**
- **[Contributing](./CONTRIBUTING.md)**
- **[Code-of-Conduct](./CODE_OF_CONDUCT.md)**
