# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument. 

All subcommands are:

* [`goenv commands`](#goenv-commands)
* [`goenv completions`](#goenv-completions)
* [`goenv exec`](#goenv-exec)
* [`goenv global`](#goenv-global)
* [`goenv help`](#goenv-help)
* [`goenv hooks`](#goenv-hooks)
* [`goenv init`](#goenv-init)
* [`goenv install`](#goenv-install)
* [`goenv local`](#goenv-local)
* [`goenv prefix`](#goenv-prefix)
* [`goenv rehash`](#goenv-rehash)
* [`goenv root`](#goenv-root)
* [`goenv shell`](#goenv-shell)
* [`goenv shims`](#goenv-shims)
* [`goenv uninstall`](#goenv-uninstall)
* [`goenv version`](#goenv-version)
* [`goenv --version`](#goenv---version)
* [`goenv version-file`](#goenv-version-file)
* [`goenv version-file-read`](#goenv-version-file-read)
* [`goenv version-file-write`](#goenv-version-file-write)
* [`goenv version-name`](#goenv-version-name)
* [`goenv version-origin`](#goenv-version-origin)
* [`goenv versions`](#goenv-versions)
* [`goenv whence`](#goenv-whence)
* [`goenv which`](#goenv-which)

## `goenv commands`

Lists all available goenv commands.

## `goenv completions`

Provides auto-completion for itself and other commands by calling them with `--complete`.

## `goenv exec`

Run an executable with the selected Go version.

Assuming there's an already installed golang by e.g `goenv install 1.11.1` and 
  selected by e.g `goenv global 1.11.1`,

```shell
> goenv exec go run main.go
```

## `goenv global`

Sets the global version of Go to be used in all shells by writing
the version name to the `~/.goenv/version` file. This version can be
overridden by an application-specific `.go-version` file, or by
setting the `GOENV_VERSION` environment variable.

```shell
> goenv global 1.5.4

# Showcase
> goenv versions
  system
  * 1.5.4 (set by /Users/syndbg/.goenv/version)

> goenv version
1.5.4 (set by /Users/syndbg/.goenv/version)

> go version
go version go1.5.4 darwin/amd64
```

The special version name `system` tells goenv to use the system Go
(detected by searching your `$PATH`).

When run without a version number, `goenv global` reports the
currently configured global version.

## `goenv help`

Parses and displays help contents from a command's source file.

A command is considered documented if it starts with a comment block
that has a `Summary:` or `Usage:` section. Usage instructions can
span multiple lines as long as subsequent lines are indented.
The remainder of the comment block is displayed as extended
documentation.


```shell
> goenv help help
```

```shell
> goenv help install
```

## `goenv hooks`

List hook scripts for a given goenv command

```shell
> goenv hooks uninstall
```

## `goenv init`

Configure the shell environment for goenv. Must have if you want to integrate `goenv` with your shell.

The following displays how to integrate `goenv` with your user's shell:

```shell
> goenv init
```

Usually it boils down to adding to your `.bashrc` or `.zshrc` the following:

```
eval "$(goenv init -)"
```

## `goenv install`

Install a Go version (using `go-build`). It's required that the version is a known installable definition by `go-build`.

```shell
> goenv install 1.11.1

```

## `goenv local`

Sets a local application-specific Go version by writing the version
name to a `.go-version` file in the current directory. This version
overrides the global version, and can be overridden itself by setting
the `GOENV_VERSION` environment variable or with the `goenv shell`
command.

```shell
> goenv local 1.6.1
```

When run without a version number, `goenv local` reports the currently
configured local version. You can also unset the local version:


```shell
> goenv local --unset
```

Previous versions of goenv stored local version specifications in a
file named `.goenv-version`. For backwards compatibility, goenv will
read a local version specified in an `.goenv-version` file, but a
`.go-version` file in the same directory will take precedence.

### `goenv local` (advanced)

You can specify local Go version. 

```shell
> goenv local 1.5.4

# Showcase
> goenv versions
  system
  * 1.5.4 (set by /Users/syndbg/path/to/project/.go-version)

> goenv version
1.5.4 (set by /Users/syndbg/path/to/project/.go-version)

> go version

go version go1.5.4 darwin/amd64
```

## `goenv prefix`

Displays the directory where a Go version is installed. If no
version is given, `goenv prefix' displays the location of the
currently selected version.

```shell
> goenv prefix
/home/syndbg/.goenv/versions/1.11.1
```

## `goenv rehash`

Installs shims for all Go binaries known to goenv (i.e.,
`~/.goenv/versions/*/bin/*`).
Run this command after you install a new
version of Go, or install a package that provides binaries.

```shell
> goenv rehash
```

## `goenv root`

Display the root directory where versions and shims are kept

```shell
> goenv root
/home/syndbg/.goenv
```

## `goenv shell`

Sets a shell-specific Go version by setting the `GOENV_VERSION`
environment variable in your shell. This version overrides
application-specific versions and the global version.

```shell
> goenv shell 1.5.4
```

When run without a version number, `goenv shell` reports the current
value of `GOENV_VERSION`. You can also unset the shell version:

```shell
> goenv shell --unset
```

Note that you'll need goenv's shell integration enabled (refer to [Installation](./INSTALL.md]) in order to use this command. If you
prefer not to use shell integration, you may simply set the
`GOENV_VERSION` variable yourself:

```shell
> export GOENV_VERSION=1.5.4
```

## `goenv shims`

List existing goenv shims

```shell
> goenv shims
/home/syndbg/.goenv/shims/go
/home/syndbg/.goenv/shims/godoc
/home/syndbg/.goenv/shims/gofmt
```

## `goenv uninstall`

Uninstalls the specified version if it exists, otherwise - error.

```shell
> goenv uninstall 1.6.3
```

## `goenv version`

Displays the currently active Go version, along with information on
how it was set.

```shell
> goenv version
1.11.1 (set by /home/syndbg/work/syndbg/goenv/.go-version)
```

## `goenv --version`

Show version of `goenv` in format of `goenv <version>`.

## `goenv version-file`

Detect the file that sets the current goenv version


```shell
> goenv version-file
/home/syndbg/work/syndbg/goenv/.go-version
```

## `goenv version-file-read`

Reads specified version file if it exists

```shell
> goenv version-file-read ./go-version 
1.11.1
```

## `goenv version-file-write`

Writes specified version(s) to the specified file if the version(s) exist

```shell
> goenv version-file-write ./go-version 1.11.1
```

## `goenv version-name`

Shows the current Go version

```shell
> goenv version-name
1.11.1
```

## `goenv version-origin`

Explain how the current Go version is set.

```shell
> goenv version-origin
/home/syndbg/.goenv/version)
```

## `goenv versions`

Lists all Go versions known to goenv, and shows an asterisk next to
the currently active version.

```shell
> goenv versions
  1.4.0
  1.4.1
  1.4.2
  1.4.3
  1.5.0
  1.5.1
  1.5.2
  1.5.3
  1.5.4
  1.6.0
* 1.6.1 (set by /home/syndbg/.goenv/version)
  1.6.2
```

## `goenv whence`

Lists all Go versions with the given command installed.

```shell
> goenv whence go
1.3.0
1.3.1
1.3.2
1.3.3
1.4.0
1.4.1
1.4.2
1.4.3
1.5.0
1.5.1
1.5.2
1.5.3
1.5.4
1.6.0
1.6.1
1.6.2
```

## `goenv which`

Displays the full path to the executable that goenv will invoke when
you run the given command.

```shell
> goenv which gofmt
/home/syndbg/.goenv/versions/1.6.1/bin/gofmt
```
