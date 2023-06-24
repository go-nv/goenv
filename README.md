# Go Version Management: goenv

[![PR Checks Status](https://github.com/syndbg/goenv/actions/workflows/pr_checks.yml/badge.svg)](https://github.com/syndbg/goenv/actions/workflows/pr_checks.yml)

goenv aims to be as simple as possible and follow the already established
successful version management model of [pyenv](https://github.com/yyuu/pyenv) and [rbenv](https://github.com/rbenv/rbenv).

This project was cloned from [pyenv](https://github.com/yyuu/pyenv) and modified for Go.

[![asciicast](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj.svg)](https://asciinema.org/a/17IT3YiQ56hiJsb2iHpGHlJqj)

### goenv _does..._

- Let you **change the global Go version** on a per-user basis.
- Provide support for **per-project Go versions**.
- Allow you to **override the Go version** with an environment
  variable.
- Search commands from **multiple versions of Go at a time**.

### goenv install
1.Install Goenv:
   
```ssh
git clone https://github.com/syndbg/goenv.git ~/.goenv
```

2.Configure environment variables:

Open a terminal and edit the ~/.bash_profile file (if you're using Bash) or ~/.zshrc file (if you're using Zsh):

```ssh
vim ~/.bash_profile  # or vim ~/.zshrc
```

Add the following lines at the end of the file:

```ssh
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

Save and close the file.
Execute the following command to reload the configuration:

```ssh
source ~/.bash_profile  # or source ~/.zshrc
```

3.Verify the installation:

Run the following command to verify that Goenv is properly installed:

```ssh
goenv -h
```

### goenv compared to others:

- https://github.com/crsmithdev/goenv depends on Go,
- https://github.com/moovweb/gvm is a different approach to the problem that's modeled after `nvm`.
  `goenv` is more simplified.

---

## Links

- **[How It Works](./HOW_IT_WORKS.md)**
- **[Installation](./INSTALL.md)**
- **[Command Reference](./COMMANDS.md)**
- **[Environment variables](./ENVIRONMENT_VARIABLES.md)**
- **[Contributing](./CONTRIBUTING.md)**
- **[Code-of-Conduct](./CODE_OF_CONDUCT.md)**
