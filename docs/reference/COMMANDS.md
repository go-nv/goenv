# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument.

All subcommands are:

- [Command Reference](#command-reference)
  - [`goenv commands`](#goenv-commands)
  - [`goenv completions`](#goenv-completions)
  - [`goenv exec`](#goenv-exec)
  - [`goenv global`](#goenv-global)
  - [`goenv help`](#goenv-help)
  - [`goenv init`](#goenv-init)
  - [`goenv install`](#goenv-install)
  - [`goenv local`](#goenv-local)
    - [`goenv local` (advanced)](#goenv-local-advanced)
  - [`goenv prefix`](#goenv-prefix)
  - [`goenv refresh`](#goenv-refresh)
  - [`goenv rehash`](#goenv-rehash)
  - [`goenv root`](#goenv-root)
  - [`goenv shell`](#goenv-shell)
  - [`goenv shims`](#goenv-shims)
  - [`goenv uninstall`](#goenv-uninstall)
  - [`goenv version`](#goenv-version)
  - [`goenv --version`](#goenv---version)
  - [`goenv version-file`](#goenv-version-file)
  - [`goenv version-file-read`](#goenv-version-file-read)
  - [`goenv version-file-write`](#goenv-version-file-write)
  - [`goenv version-name`](#goenv-version-name)
  - [`goenv version-origin`](#goenv-version-origin)
  - [`goenv versions`](#goenv-versions)
  - [`goenv whence`](#goenv-whence)
  - [`goenv which`](#goenv-which)

## `goenv commands`

Lists all available goenv commands.

**Filtering commands:**

```shell
# List all commands
> goenv commands

# List only shell commands (sh-*)
> goenv commands --sh

# List only non-shell commands
> goenv commands --no-sh
```

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
  * 1.5.4 (set by /Users/go-nv/.goenv/version)

> goenv version
1.5.4 (set by /Users/go-nv/.goenv/version)

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

Install a Go version (using `go-build`). It's required that the version is a known installable definition by `go-build`. Alternatively, supply `latest` as an argument to install the latest version available to goenv.

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

**Version shorthand syntax:**

You can use a shorthand syntax to set the local version by specifying the version directly as the first argument:

```shell
# These are equivalent:
> goenv 1.21.0
> goenv local 1.21.0

# Works with various version formats:
> goenv 1.22.5        # Full version
> goenv latest        # Latest installed version
> goenv system        # System Go
```

This shorthand automatically routes to the `local` command, making it faster to switch versions in your current directory.

## `goenv prefix`

Displays the directory where a Go version is installed. If no
version is given, `goenv prefix' displays the location of the
currently selected version.

```shell
> goenv prefix
/home/go-nv/.goenv/versions/1.11.1
```

## `goenv refresh`

Clears all cached version data and forces a fresh fetch from the official Go API on the next version-related command.

```shell
> goenv refresh
✓ Cache cleared! Removed 2 cache file(s).
Next version fetch will retrieve fresh data from go.dev

# With verbose output
> goenv refresh --verbose
✓ Removed versions-cache.json
✓ Removed releases-cache.json
✓ Cache cleared! Removed 2 cache file(s).
Next version fetch will retrieve fresh data from go.dev
```

This removes:

- `versions-cache.json` - Version list cache
- `releases-cache.json` - Full release metadata cache

Use this command when:

- A new Go version was just released and you want it immediately
- You suspect cached data is stale or corrupted
- Testing version fetching behavior
- Troubleshooting version-related issues

The next time you run `goenv install --list` or similar commands, goenv will fetch fresh data from go.dev.

## `goenv rehash`

Installs shims for all Go binaries known to goenv. This includes:
- Go version binaries: `~/.goenv/versions/*/bin/*`
- GOPATH binaries: `$GOENV_GOPATH_PREFIX/<version>/bin/*` (unless `GOENV_DISABLE_GOPATH=1`)

Run this command after you install a new version of Go, or install a package that provides binaries.

```shell
> goenv rehash
```

**GOPATH binary support:**

By default, goenv also creates shims for binaries in your GOPATH. This allows tools installed with `go install` to work seamlessly with version switching:

```shell
# Install a tool with the current Go version
> go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Rehash to create shims
> goenv rehash

# The tool is now available via shim
> golangci-lint version
```

To disable GOPATH binary scanning:
```shell
export GOENV_DISABLE_GOPATH=1
```

To customize the GOPATH location:
```shell
export GOENV_GOPATH_PREFIX=/custom/path  # Default: $HOME/go
```

## `goenv root`

Display the root directory where versions and shims are kept

```shell
> goenv root
/home/go-nv/.goenv
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
/home/go-nv/.goenv/shims/go
/home/go-nv/.goenv/shims/godoc
/home/go-nv/.goenv/shims/gofmt
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
1.11.1 (set by /home/syndbg/work/go-nv/goenv/.go-version)
```

## `goenv --version`

Show version of `goenv` in format of `goenv <version>`.

## `goenv version-file`

Detect the file that sets the current goenv version

```shell
> goenv version-file
/home/syndbg/work/go-nv/goenv/.go-version
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
/home/go-nv/.goenv/version)
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
* 1.6.1 (set by /home/go-nv/.goenv/version)
  1.6.2
```

## `goenv whence`

Lists all Go versions with the given command installed. Searches both version bin directories and GOPATH bin directories (unless `GOENV_DISABLE_GOPATH=1`).

```shell
# List versions with the 'go' command
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

# List versions with a GOPATH-installed tool
> goenv whence golangci-lint
1.21.5
1.22.5
```

Use the `--path` flag to show full paths instead of version names:

```shell
> goenv whence --path golangci-lint
/home/go-nv/go/1.21.5/bin/golangci-lint
/home/go-nv/go/1.22.5/bin/golangci-lint
```

## `goenv which`

Displays the full path to the executable that goenv will invoke when
you run the given command. Searches both version bin directories and GOPATH bin directories (unless `GOENV_DISABLE_GOPATH=1`).

```shell
# Find a Go core binary
> goenv which gofmt
/home/go-nv/.goenv/versions/1.6.1/bin/gofmt

# Find a GOPATH-installed tool
> goenv which golangci-lint
/home/go-nv/go/1.22.5/bin/golangci-lint
```
