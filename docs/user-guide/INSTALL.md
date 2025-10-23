# Installation

## Quick Install (Recommended - No Go Required!)

**The easiest way to install goenv is using pre-built binaries.** This method doesn't require Go to be installed on your system.

### Automatic Installation Script

```bash
# Linux/macOS
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
```

```powershell
# Windows (PowerShell)
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex
```

### Manual Binary Installation

1. **Download the latest binary for your platform:**

   Visit [goenv releases](https://github.com/go-nv/goenv/releases/latest) and download the appropriate archive:

   - **Linux (x64)**: `goenv_*_linux_amd64.tar.gz`
   - **Linux (ARM64)**: `goenv_*_linux_arm64.tar.gz`
   - **macOS (Intel)**: `goenv_*_darwin_amd64.tar.gz`
   - **macOS (Apple Silicon)**: `goenv_*_darwin_arm64.tar.gz`
   - **Windows (x64)**: `goenv_*_windows_amd64.zip`
   - **FreeBSD (x64)**: `goenv_*_freebsd_amd64.tar.gz`

2. **Extract and install:**

   ```bash
   # Linux/macOS
   tar -xzf goenv_*_*.tar.gz
   mkdir -p ~/.goenv/bin
   mv goenv ~/.goenv/bin/
   chmod +x ~/.goenv/bin/goenv
   ```

   ```powershell
   # Windows
   Expand-Archive goenv_*_windows_amd64.zip
   mkdir $HOME\.goenv\bin -Force
   mv goenv.exe $HOME\.goenv\bin\
   ```

3. **Add to your shell** (see "Shell Setup" section below)

---

## Basic GitHub Checkout

This will get you going with the latest version of goenv and make it
easy to fork and contribute any changes back upstream.

**Note:** This method requires Go to be installed to build goenv. If you don't have Go installed, use the **Quick Install** method above.

1.  **Check out goenv where you want it installed.**
    A good place to choose is `$HOME/.goenv` (but you can install it somewhere else).

        git clone https://github.com/go-nv/goenv.git ~/.goenv

2.  **Build goenv** (requires Go to be installed):

    cd ~/.goenv
    make build

3.  **Define environment variable `GOENV_ROOT`** to point to the path where
    goenv repo is cloned and add `$GOENV_ROOT/bin` to your `$PATH` for access
    to the `goenv` command-line utility.

        echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bash_profile
        echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` file instead of `~/.bash_profile`.

    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

4.  **Add `goenv init` to your shell** to enable shims, management of `GOPATH` and `GOROOT` and auto-completion.
    Please make sure `eval "$(goenv init -)"` is placed toward the end of the shell
    configuration file since it manipulates `PATH` during the initialization.

        echo 'eval "$(goenv init -)"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` or `~/.zshrc` file instead of `~/.bash_profile`.

    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

    **General warning**: There are some systems where the `BASH_ENV` variable is configured
    to point to `.bashrc`. On such systems you should almost certainly put the abovementioned line
    `eval "$(goenv init -)` into `.bash_profile`, and **not** into `.bashrc`. Otherwise you
    may observe strange behaviour, such as `goenv` getting into an infinite loop.
    See pyenv's issue [#264](https://github.com/pyenv/pyenv/issues/264) for details.

5.  **Restart your shell so the path changes take effect.**
    You can now begin using goenv.

        exec $SHELL

6.  **(Optional) Enable tab completion for faster command-line usage.**
    goenv includes built-in shell completion scripts that enable tab completion for commands, flags, and Go versions.

    **Quick Install (Recommended):**

    ```bash
    goenv completion --install
    ```

    This will:

    - Auto-detect your shell (bash/zsh/fish/powershell)
    - Install the completion script to the appropriate location
    - Display activation instructions

    **Manual Install:**

    ```bash
    # Output the completion script and add to your shell config manually
    goenv completion >> ~/.bashrc  # or ~/.zshrc, ~/.config/fish/config.fish, etc.
    ```

    **Restart your shell** to activate completions:

    ```bash
    exec $SHELL
    ```

    After setup, you can use tab completion:

    - `goenv <TAB>` - See all available commands
    - `goenv install <TAB>` - See available Go versions
    - `goenv use <TAB>` - See installed versions

7.  **Install Go versions into `$GOENV_ROOT/versions`.**
    For example, to download and install Go 1.12.0, run:

        goenv install 1.12.0

    **NOTE:** It downloads and places the prebuilt Go binaries provided by Google.

8.  **Set global Go version.**
    For example, to set the version to Go 1.12.0, run:

        goenv use 1.12.0 --global

---

## Shell Setup

After installing goenv (via binary or git checkout), add the following to your shell configuration:

### Bash (~/.bash_profile or ~/.bashrc)

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Zsh (~/.zshrc or ~/.zshenv)

```zsh
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Fish (~/.config/fish/config.fish)

```fish
set -Ux GOENV_ROOT $HOME/.goenv
set -U fish_user_paths $GOENV_ROOT/bin $fish_user_paths
status --is-interactive; and goenv init - | source
```

### PowerShell ($PROFILE)

```powershell
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
& goenv init - | Invoke-Expression
```

Then restart your shell:

```bash
exec $SHELL
```

An example `.zshrc` that is properly configured may look like

```shell
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

## via ZPlug plugin manager for Zsh

Add the following line to your `.zshrc`:

`zplug "RiverGlide/zsh-goenv", from:gitlab`
Then install the plugin

```zsh
  source ~/.zshrc
  zplug install
```

The ZPlug plugin will install and initialise `goenv` and add `goenv` and `goenv-install` to your `PATH`

## Homebrew on macOS

**Recommended for macOS users who use Homebrew.**

You can install goenv using the [Homebrew](http://brew.sh) package manager for macOS (and Linux).

```bash
brew update
brew install goenv
```

**Advantages**:

- Automatic dependency handling
- Easy updates with `brew upgrade goenv`
- Integration with system package manager

**After installation**, you'll need to add the following to your shell profile (as shown in Homebrew caveats):

```bash
# Add to ~/.zshrc or ~/.bash_profile
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**To see installation details again**:

```bash
brew info goenv
```

**To upgrade in the future**:

```bash
brew upgrade goenv
```

**To uninstall**:

```bash
brew uninstall goenv
```

---

## Upgrading

If you've installed goenv using the instructions above, you can
upgrade your installation at any time using git.

To upgrade to the latest development version of goenv, use `git pull`:

    cd ~/.goenv && git fetch --all && git pull

To upgrade to a specific release of goenv, check out the corresponding tag:

    cd ~/.goenv
    git fetch --all
    git tag
    v20160417
    git checkout v20160417

## Uninstalling goenv

The simplicity of goenv makes it easy to temporarily disable it, or
uninstall from the system.

1. To **disable** goenv managing your Go versions, simply remove the
   `goenv init` line from your shell startup configuration. This will
   remove goenv shims directory from PATH, and future invocations like
   `goenv` will execute the system Go version, as before goenv.

`goenv` will still be accessible on the command line, but your Go
apps won't be affected by version switching.

2.  To completely **uninstall** goenv, perform step (1) and then remove
    its root directory. This will **delete all Go versions** that were
    installed under `` `goenv root`/versions/ `` directory:

         rm -rf `goenv root`

    If you've installed goenv using a package manager, as a final step
    perform the goenv package removal. For instance, for Homebrew:

         brew uninstall goenv


## Uninstalling Go Versions

As time goes on, you will accumulate Go versions in your
`~/.goenv/versions` directory.

To remove old Go versions, `goenv uninstall` command to automate
the removal process.

Alternatively, simply `rm -rf` the directory of the version you want
to remove. You can find the directory of a particular Go version
with the `goenv prefix` command, e.g. `goenv prefix 1.6.2`.
