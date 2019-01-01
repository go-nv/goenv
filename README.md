# Go Version Management: goenv

[![Build Status](https://travis-ci.org/syndbg/goenv.svg?branch=master)](https://travis-ci.org/syndbg/goenv)

goenv aims to be as simple as possible and follow the already established
successful version management model of [pyenv](https://github.com/yyuu/pyenv) and [rbenv](https://github.com/rbenv/rbenv).

This project was cloned from [pyenv](https://github.com/yyuu/pyenv) and modified for Go.

[![asciicast](https://asciinema.org/a/ci4otj2507p1w7h91c0s8bjcu.png)](https://asciinema.org/a/ci4otj2507p1w7h91c0s8bjcu)

### goenv _does..._

* Let you **change the global Go version** on a per-user basis.
* Provide support for **per-project Go versions**.
* Allow you to **override the Go version** with an environment
  variable.
* Search commands from **multiple versions of Go at a time**.

### goenv compared to others:

* https://github.com/pwoolcoc/goenv depends on Python,
* https://github.com/crsmithdev/goenv depends on Go,
* https://github.com/moovweb/gvm is a different approach of the problem that's modeled after `nvm`. `goenv` is more simplified.

----

## Links

* **[How It Works](./HOW_IT_WORKS.md)**
* **[Installation](./INSTALL.md)**
* **[Command Reference](./COMMANDS.md)**
* **[Environment variables](#environment-variables)**
* **[Development](#development)**

----

## Environment variables

You can affect how goenv operates with the following settings:

name | default | description
-----|---------|------------
`GOENV_VERSION` | | Specifies the Go version to be used.<br>Also see `goenv help shell`.
`GOENV_ROOT` | `~/.goenv` | Defines the directory under which Go versions and shims reside.<br> Current value shown by `goenv root`.
`GOENV_DEBUG` | | Outputs debug information.<br>Also as: `goenv --debug <subcommand>`
`GOENV_HOOK_PATH` | | Colon-separated list of paths searched for goenv hooks.
`GOENV_DIR` | `$PWD` | Directory to start searching for `.go-version` files.

## Development

The goenv source code is [hosted on
GitHub](https://github.com/syndbg/goenv).  It's clean, modular,
and easy to understand, even if you're not a shell hacker. (I hope)

Tests are executed using [Bats](https://github.com/bats-core/bats-core):

```
$ make test
```

Please feel free to submit pull requests and file bugs on the [issue
tracker](https://github.com/syndbg/goenv/issues).
