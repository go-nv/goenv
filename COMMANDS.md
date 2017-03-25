# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument. 

The most common subcommands are:

* [`goenv commands`](#goenv-commands)
* [`goenv local`](#goenv-local)
* [`goenv global`](#goenv-global)
* [`goenv shell`](#goenv-shell)
* [`goenv install`](#goenv-install)
* [`goenv uninstall`](#goenv-uninstall)
* [`goenv rehash`](#goenv-rehash)
* [`goenv version`](#goenv-version)
* [`goenv versions`](#goenv-versions)
* [`goenv which`](#goenv-which)
* [`goenv whence`](#goenv-whence)

## `goenv commands`

Lists all available goenv commands.

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

Note that you'll need goenv's shell integration enabled (step 3 of
the installation instructions) in order to use this command. If you
prefer not to use shell integration, you may simply set the
`GOENV_VERSION` variable yourself:

```shell
> export GOENV_VERSION=1.5.4

```

## `goenv install`

Install a Go version (using `go-build`).

```shell
> goenv install

Usage: goenv install [-f] [-kvp] <version>
        goenv install [-f] [-kvp] <definition-file>
        goenv install -l|--list

  -l/--list             List all available versions
  -f/--force            Install even if the version appears to be installed already
  -s/--skip-existing    Skip the installation if the version appears to be installed already

  go-build options:

  -k/--keep        Keep source tree in $GOENV_BUILD_ROOT after installation
                    (defaults to $GOENV_ROOT/sources)
  -v/--verbose     Verbose mode: print compilation status to stdout
  -p/--patch       Apply a patch from stdin before building
  -g/--debug       Build a debug version
```

## `goenv uninstall`

Uninstall a specific Go version.

```shell
> goenv uninstall
Usage: goenv uninstall [-f|--force] <version>

    -f  Attempt to remove the specified version without prompting
        for confirmation. If the version does not exist, do not
        display an error message.
```

## `goenv rehash`

Installs shims for all Go binaries known to goenv (i.e.,
`~/.goenv/versions/*/bin/*`).
Run this command after you install a new
version of Go, or install a package that provides binaries.

```shell
> goenv rehash
```

## `goenv version`

Displays the currently active Go version, along with information on
how it was set.

```shell
> goenv version
1.5.4 (set by /Users/syndbg/.goenv/version)
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
* 1.6.1 (set by /Users/syndbg/.goenv/version)
  1.6.2
```

## `goenv which`

Displays the full path to the executable that goenv will invoke when
you run the given command.

```shell
> goenv which gofmt
/home/syndbg/.goenv/versions/1.6.1/bin/gofmt
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
