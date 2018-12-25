# Go Version Management: goenv

goenv aims to be as simple as possible and follow the already estabilished
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

## Table of Contents

* **[How It Works](#how-it-works)**
  * [Understanding PATH](#understanding-path)
  * [Understanding Shims](#understanding-shims)
  * [Choosing the Go Version](#choosing-the-go-version)
  * [Locating the Go Installation](#locating-the-go-installation)
* **[Installation](#installation)**
  * [Basic GitHub Checkout](#basic-github-checkout)
    * [Upgrading](#upgrading)
    * [Advanced Configuration](#advanced-configuration)
    * [Uninstalling Go Versions](#uninstalling-go-versions)
* **[Command Reference](#command-reference)**
* **[Development](#development)**
  * [Version History](#version-history)
  * [License](#license)

----


## How It Works

At a high level, goenv intercepts Go commands using shim
executables injected into your `PATH`, determines which Go version
has been specified by your application, and passes your commands along
to the correct Go installation.

### Understanding PATH

When you run all the variety of Go commands using  `go`, your operating system
searches through a list of directories to find an executable file with
that name. This list of directories lives in an environment variable
called `PATH`, with each directory in the list separated by a colon:

    /usr/local/bin:/usr/bin:/bin

Directories in `PATH` are searched from left to right, so a matching
executable in a directory at the beginning of the list takes
precedence over another one at the end. In this example, the
`/usr/local/bin` directory will be searched first, then `/usr/bin`,
then `/bin`.

### Understanding Shims

goenv works by inserting a directory of _shims_ at the front of your
`PATH`:

    ~/.goenv/shims:/usr/local/bin:/usr/bin:/bin

Through a process called _rehashing_, goenv maintains shims in that
directory to match every `go` command across every installed version
of Go.

Shims are lightweight executables that simply pass your command along
to goenv. So with goenv installed, when you run `go` your
operating system will do the following:

* Search your `PATH` for an executable file named `go`
* Find the goenv shim named `go` at the beginning of your `PATH`
* Run the shim named `go`, which in turn passes the command along to
  goenv

### Choosing the Go Version

When you execute a shim, goenv determines which Go version to use by
reading it from the following sources, in this order:

1. The `GOENV_VERSION` environment variable (if specified). You can use
   the [`goenv shell`](https://github.com/syndbg/goenv/blob/master/COMMANDS.md#goenv-shell) command to set this environment
   variable in your current shell session.

2. The application-specific `.go-version` file in the current
   directory (if present). You can modify the current directory's
   `.go-version` file with the [`goenv local`](https://github.com/syndbg/goenv/blob/master/COMMANDS.md#goenv-local)
   command.

3. The first `.go-version` file found (if any) by searching each parent
   directory, until reaching the root of your filesystem.

4. The global `~/.goenv/version` file. You can modify this file using
   the [`goenv global`](https://github.com/syndbg/goenv/blob/master/COMMANDS.md#goenv-global) command. If the global version
   file is not present, goenv assumes you want to use the "system"
   Go. (In other words, whatever version would run if goenv isn't present in
   `PATH`.)

**NOTE:** You can activate multiple versions at the same time, including multiple
versions of Go simultaneously or per project.

### Locating the Go Installation

Once goenv has determined which version of Go your application has
specified, it passes the command along to the corresponding Go
installation.

Each Go version is installed into its own directory under
`~/.goenv/versions`.

For example, you might have these versions installed:

* `~/.goenv/versions/1.6.1/`
* `~/.goenv/versions/1.6.2/`

As far as goenv is concerned, version names are simply the directories in
`~/.goenv/versions`.

## Installation

If you're on Mac OS X, consider [installing with Homebrew](#homebrew-on-mac-os-x).

### Basic GitHub Checkout

This will get you going with the latest version of goenv and make it
easy to fork and contribute any changes back upstream.

1. **Check out goenv where you want it installed.**
   A good place to choose is `$HOME/.goenv` (but you can install it somewhere else).

        $ git clone https://github.com/syndbg/goenv.git ~/.goenv


2. **Define environment variable `GOENV_ROOT`** to point to the path where
   goenv repo is cloned and add `$GOENV_ROOT/bin` to your `$PATH` for access
   to the `goenv` command-line utility.

        $ echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bash_profile
        $ echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` file instead of `~/.bash_profile`.
    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

3. **Add `goenv init` to your shell** to enable shims and autocompletion.
   Please make sure `eval "$(goenv init -)"` is placed toward the end of the shell
   configuration file since it manipulates `PATH` during the initialization.

        $ echo 'eval "$(goenv init -)"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` file instead of `~/.bash_profile`.
    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.
    
    **General warning**: There are some systems where the `BASH_ENV` variable is configured
    to point to `.bashrc`. On such systems you should almost certainly put the abovementioned line
    `eval "$(goenv init -)` into `.bash_profile`, and **not** into `.bashrc`. Otherwise you
    may observe strange behaviour, such as `goenv` getting into an infinite loop.
    See pyenv's issue [#264](https://github.com/yyuu/pyenv/issues/264) for details.

4. **Restart your shell so the path changes take effect.**
   You can now begin using goenv.

        $ exec $SHELL

5. **Install Go versions into `$GOENV_ROOT/versions`.**
   For example, to download and install Go 1.6.2, run:

        $ goenv install 1.6.2

   **NOTE:** It downloads and places the prebuilt Go binaries provided by Google.

#### Upgrading

If you've installed goenv using the instructions above, you can
upgrade your installation at any time using git.

To upgrade to the latest development version of goenv, use `git pull`:

    $ cd ~/.goenv
    $ git pull

To upgrade to a specific release of goenv, check out the corresponding tag:

    $ cd ~/.goenv
    $ git fetch
    $ git tag
    v20160417
    $ git checkout v20160417

### Uninstalling goenv

The simplicity of goenv makes it easy to temporarily disable it, or
uninstall from the system.

1. To **disable** goenv managing your Go versions, simply remove the
  `goenv init` line from your shell startup configuration. This will
  remove goenv shims directory from PATH, and future invocations like
  `goenv` will execute the system Go version, as before goenv.

  `goenv` will still be accessible on the command line, but your Go
  apps won't be affected by version switching.

2. To completely **uninstall** goenv, perform step (1) and then remove
   its root directory. This will **delete all Go versions** that were
   installed under `` `goenv root`/versions/ `` directory:

        rm -rf `goenv root`

   If you've installed goenv using a package manager, as a final step
   perform the goenv package removal. For instance, for Homebrew:

        brew uninstall goenv

## Command Reference

### Homebrew on Mac OS X

You can also install goenv using the [Homebrew](http://brew.sh)
package manager for Mac OS X.

    $ brew update
    $ brew install goenv

To upgrade goenv in the future, use `upgrade` instead of `install`.

After installation, you'll need to add `eval "$(goenv init -)"` to your profile (as stated in the caveats displayed by Homebrew — to display them again, use `brew info goenv`). You only need to add that to your profile once.

Then follow the rest of the post-installation steps under "Basic GitHub Checkout" above, starting with #4 ("restart your shell so the path changes take effect").

### Advanced Configuration

Skip this section unless you must know what every line in your shell
profile is doing.

`goenv init` is the only command that crosses the line of loading
extra commands into your shell. Coming from rvm, some of you might be
opposed to this idea. Here's what `goenv init` actually does:

1. **Sets up your shims path.** This is the only requirement for goenv to
   function properly. You can do this by hand by prepending
   `~/.goenv/shims` to your `$PATH`.

2. **Installs autocompletion.** This is entirely optional but pretty
   useful. Sourcing `~/.goenv/completions/goenv.bash` will set that
   up. There is also a `~/.goenv/completions/goenv.zsh` for Zsh
   users.

3. **Rehashes shims.** From time to time you'll need to rebuild your
   shim files. Doing this on init makes sure everything is up to
   date. You can always run `goenv rehash` manually.

4. **Installs the sh dispatcher.** This bit is also optional, but allows
   goenv and plugins to change variables in your current shell, making
   commands like `goenv shell` possible. The sh dispatcher doesn't do
   anything crazy like override `cd` or hack your shell prompt, but if
   for some reason you need `goenv` to be a real script rather than a
   shell function, you can safely skip it.

To see exactly what happens under the hood for yourself, run `goenv init -`.

### Uninstalling Go Versions

As time goes on, you will accumulate Go versions in your
`~/.goenv/versions` directory.

To remove old Go versions, `goenv uninstall` command to automate
the removal process.

Alternatively, simply `rm -rf` the directory of the version you want
to remove. You can find the directory of a particular Go version
with the `goenv prefix` command, e.g. `goenv prefix 1.6.2`.

----

## Command Reference

See [COMMANDS.md](COMMANDS.md).

----

## Environment variables

You can affect how goenv operates with the following settings:

name | default | description
-----|---------|------------
`GOENV_VERSION` | | Specifies the Go version to be used.<br>Also see [`goenv shell`](#goenv-shell)
`GOENV_ROOT` | `~/.goenv` | Defines the directory under which Go versions and shims reside.<br>Also see `goenv root`
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
