# Installation

## Basic GitHub Checkout

This will get you going with the latest version of goenv and make it
easy to fork and contribute any changes back upstream.

1. **Check out goenv where you want it installed.**
   A good place to choose is `$HOME/.goenv` (but you can install it somewhere else).

       git clone https://github.com/go-nv/goenv.git ~/.goenv

2. **Define environment variable `GOENV_ROOT`** to point to the path where
   goenv repo is cloned and add `$GOENV_ROOT/bin` to your `$PATH` for access
   to the `goenv` command-line utility.

       echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bash_profile
       echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` file instead of `~/.bash_profile`.

    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

3. **Add `goenv init` to your shell** to enable shims, management of `GOPATH` and `GOROOT` and auto-completion.
   Please make sure `eval "$(goenv init -)"` is placed toward the end of the shell
   configuration file since it manipulates `PATH` during the initialization.

       echo 'eval "$(goenv init -)"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` or `~/.zshrc` file instead of `~/.bash_profile`.
    
    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.
    
    **General warning**: There are some systems where the `BASH_ENV` variable is configured
    to point to `.bashrc`. On such systems you should almost certainly put the abovementioned line
    `eval "$(goenv init -)` into `.bash_profile`, and **not** into `.bashrc`. Otherwise you
    may observe strange behaviour, such as `goenv` getting into an infinite loop.
    See pyenv's issue [#264](https://github.com/yyuu/pyenv/issues/264) for details.
    
4. **If you want  `goenv` to manage `GOPATH` and `GOROOT` (recommended)**, 
  add `GOPATH` and `GOROOT` to your shell **after `eval "$(goenv init -)"`**.
  
       echo 'export PATH="$GOROOT/bin:$PATH"' >> ~/.bash_profile
       echo 'export PATH="$PATH:$GOPATH/bin"' >> ~/.bash_profile
        
    **Zsh note**: Modify your `~/.zshenv` or `~/.zshrc` file instead of `~/.bash_profile`.
    
    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

    **General warning**: There are some systems where the `BASH_ENV` variable is configured
    to point to `.bashrc`. On such systems you should almost certainly put the abovementioned line
    `eval "$(goenv init -)` into `.bash_profile`, and **not** into `.bashrc`. Otherwise you
    may observe strange behaviour, such as `goenv` getting into an infinite loop.
    See pyenv's issue [#264](https://github.com/yyuu/pyenv/issues/264) for details.

    **Security warning**: You likely want to keep $GOPATH/bin at the end
    of your $PATH as shown above, rather than at the beginning.  See
    [#99](https://github.com/go-nv/goenv/issues/99) for details and
    discussion.
  

5. **Restart your shell so the path changes take effect.**
   You can now begin using goenv.

       exec $SHELL

6. **Install Go versions into `$GOENV_ROOT/versions`.**
   For example, to download and install Go 1.12.0, run:

       goenv install 1.12.0

   **NOTE:** It downloads and places the prebuilt Go binaries provided by Google.

7. **Set goenv global version.**
   For example, to set the version to Go 1.12.0, run:

       goenv global 1.12.0
   
An example `.zshrc` that is properly configured may look like

```shell
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
export PATH="$GOROOT/bin:$PATH"
export PATH="$PATH:$GOPATH/bin"
```
   
## via ZPlug plugin manager for Zsh

Add the following line to your `.zshrc`:

```zplug "RiverGlide/zsh-goenv", from:gitlab```
Then install the plugin
~~~ zsh
  source ~/.zshrc
  zplug install
~~~
The ZPlug plugin will install and initialise `goenv` and add `goenv` and `goenv-install` to your `PATH`
   
## Homebrew on Mac OS X

You can also install goenv using the [Homebrew](http://brew.sh)
package manager for Mac OS X.

    brew update
    brew install goenv

To upgrade goenv in the future, use `upgrade` instead of `install`.

After installation, you'll need to add `eval "$(goenv init -)"` to your profile (as stated in the caveats displayed by Homebrew — to display them again, use `brew info goenv`). You only need to add that to your profile once.

Then follow the rest of the post-installation steps under "Basic GitHub Checkout" above, starting with #4 ("restart your shell so the path changes take effect").

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

2. To completely **uninstall** goenv, perform step (1) and then remove
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
