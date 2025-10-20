# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument.

All subcommands are:

- [Command Reference](#command-reference)
  - [`goenv alias`](#goenv-alias)
  - [`goenv commands`](#goenv-commands)
  - [`goenv completions`](#goenv-completions)
  - [`goenv default-tools`](#goenv-default-tools)
  - [`goenv doctor`](#goenv-doctor)
  - [`goenv exec`](#goenv-exec)
  - [`goenv global`](#goenv-global)
  - [`goenv help`](#goenv-help)
  - [`goenv init`](#goenv-init)
  - [`goenv install`](#goenv-install)
    - [Options](#options)
    - [Auto-Rehash](#auto-rehash)
  - [`goenv installed`](#goenv-installed)
  - [`goenv local`](#goenv-local)
    - [`goenv local` (advanced)](#goenv-local-advanced)
  - [`goenv migrate-tools`](#goenv-migrate-tools)
    - [Options](#options-1)
  - [`goenv prefix`](#goenv-prefix)
  - [`goenv refresh`](#goenv-refresh)
  - [`goenv rehash`](#goenv-rehash)
  - [`goenv root`](#goenv-root)
  - [`goenv shell`](#goenv-shell)
  - [`goenv shims`](#goenv-shims)
  - [`goenv unalias`](#goenv-unalias)
  - [`goenv uninstall`](#goenv-uninstall)
  - [`goenv update`](#goenv-update)
    - [Options](#options-2)
    - [Installation Methods](#installation-methods)
  - [`goenv update-tools`](#goenv-update-tools)
    - [Options](#options-3)
  - [`goenv version`](#goenv-version)
  - [`goenv --version`](#goenv---version)
  - [`goenv version-file`](#goenv-version-file)
  - [`goenv version-file-read`](#goenv-version-file-read)
  - [`goenv version-file-write`](#goenv-version-file-write)
  - [`goenv version-name`](#goenv-version-name)
  - [`goenv version-origin`](#goenv-version-origin)
  - [`goenv versions`](#goenv-versions)
    - [Options](#options-4)
  - [`goenv whence`](#goenv-whence)
  - [`goenv which`](#goenv-which)

## `goenv alias`

Create and manage version aliases. Aliases provide convenient shorthand names for Go versions, making it easier to reference commonly used versions.

**Usage:**

```shell
# List all aliases
> goenv alias

# Show specific alias
> goenv alias stable
1.23.0

# Create or update an alias
> goenv alias stable 1.23.0
> goenv alias dev latest
> goenv alias lts 1.22.5
```

**Using aliases:**

Once created, aliases can be used anywhere a version number is expected:

```shell
# Set global version using alias
> goenv global stable

# Set local version using alias
> goenv local dev

# Aliases are resolved to their target versions
> goenv version
1.23.0 (set by /home/go-nv/.goenv/version)
```

**Alias features:**

- Aliases are stored in `~/.goenv/aliases` and persist across sessions
- Alias names cannot conflict with reserved keywords (`system`, `latest`)
- Aliases must contain only alphanumeric characters, hyphens, and underscores
- Aliases are automatically resolved when setting versions with `global` or `local`
- You can create aliases that point to special versions like `latest` or `system`

**Common use cases:**

```shell
# Track LTS versions
> goenv alias lts 1.22.5
> goenv global lts

# Maintain development and stable versions
> goenv alias stable 1.23.0
> goenv alias dev 1.24rc1

# Quick version names
> goenv alias prod 1.23.2
> goenv alias staging 1.23.0
```

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

## `goenv default-tools`

Manages the list of tools automatically installed with each new Go version.

Default tools are specified in `~/.goenv/default-tools.yaml` and are automatically installed after each `goenv install` command completes successfully.

**Subcommands:**

```shell
# List configured default tools
> goenv default-tools list

# Initialize default tools configuration with sensible defaults
> goenv default-tools init

# Enable automatic tool installation
> goenv default-tools enable

# Disable automatic tool installation
> goenv default-tools disable

# Install default tools for a specific Go version
> goenv default-tools install 1.25.2
```

**Common use cases:**

- Auto-install gopls, golangci-lint, staticcheck, delve
- Ensure consistent development environment across Go versions
- Reduce manual setup after installing new Go versions

**Configuration file example (`~/.goenv/default-tools.yaml`):**

```yaml
enabled: true
tools:
  - golang.org/x/tools/gopls@latest
  - github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  - honnef.co/go/tools/cmd/staticcheck@latest
  - github.com/go-delve/delve/cmd/dlv@latest
```

## `goenv doctor`

Diagnose goenv installation and configuration issues.

```shell
> goenv doctor
🔍 Checking goenv installation...

✓ goenv binary
  Location: /Users/user/.goenv/bin/goenv
  
✓ Shell configuration
  Found eval in ~/.zshrc
  
✓ PATH setup
  Shims directory is in PATH
  
✓ Shims directory
  Location: /Users/user/.goenv/shims
  Shim count: 12
  
✓ Go versions
  3 version(s) installed
  Current: 1.22.5
  
✓ Configuration complete
```

This command verifies:
- goenv binary and paths
- Shell configuration (init integration)
- PATH setup (shims directory)
- Shims directory existence
- Installed Go versions
- Common configuration problems

Use this command to troubleshoot issues with goenv.

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

### Options

- `-f, --force` - Install even if the version appears to be installed already
- `-s, --skip-existing` - Skip if the version appears to be installed already
- `-l, --list` - List all available versions
- `-k, --keep` - Keep source tree after installation
- `-v, --verbose` - Verbose mode: print detailed installation info
- `-q, --quiet` - Quiet mode: disable progress bar
- `-4, --ipv4` - Resolve names to IPv4 addresses only
- `-6, --ipv6` - Resolve names to IPv6 addresses only
- `-g, --debug` - Enable debug output
- `--no-rehash` - Skip automatic rehash after installation (advanced)

### Auto-Rehash

By default, goenv automatically rehashes shims after installation so that installed binaries are immediately available. For batch installations or CI/CD pipelines, you can disable this:

```shell
# Disable for single install
> goenv install 1.21.0 --no-rehash

# Batch install with single rehash at end
> goenv install 1.21.0 --no-rehash
> goenv install 1.22.0 --no-rehash
> goenv rehash

# Or use environment variable
> export GOENV_NO_AUTO_REHASH=1
> goenv install 1.21.0
> goenv install 1.22.0
> goenv rehash
```

## `goenv installed`

Display an installed Go version, searching for shortcuts if necessary.

**Usage:**

```shell
# Show the latest installed version
> goenv installed latest
1.23.2

# Check if a specific version is installed
> goenv installed 1.22.5
1.22.5

# With no arguments, shows current version
> goenv installed
1.22.5
```

This command is useful for scripting and automation to verify version installations.

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

## `goenv migrate-tools`

Migrate installed Go tools from one Go version to another.

**Usage:**

```shell
# Migrate all tools from 1.24.1 to 1.25.2
> goenv migrate-tools 1.24.1 1.25.2

# Preview migration without actually installing
> goenv migrate-tools 1.24.1 1.25.2 --dry-run

# Migrate only specific tools
> goenv migrate-tools 1.24.1 1.25.2 --select gopls,delve

# Migrate all except specific tools
> goenv migrate-tools 1.24.1 1.25.2 --exclude staticcheck
```

### Options

- `--dry-run` - Show what would be migrated without actually migrating
- `--select <tools>` - Comma-separated list of tools to migrate (e.g., gopls,delve)
- `--exclude <tools>` - Comma-separated list of tools to exclude from migration

This command is useful when upgrading Go versions and wanting to maintain your tool environment. It discovers all tools in the source version and reinstalls them in the target version.

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

## `goenv unalias`

Remove a version alias.

**Usage:**

```shell
# Remove an alias
> goenv unalias stable

# List remaining aliases
> goenv alias
dev -> 1.24rc1
lts -> 1.22.5
```

This command removes the specified alias from the `~/.goenv/aliases` file. The target version remains installed and can still be used directly by its version number.

**Example workflow:**

```shell
# Create an alias
> goenv alias temp 1.23.0

# Use it
> goenv local temp

# Remove it when no longer needed
> goenv unalias temp

# Version still works directly
> goenv local 1.23.0
```

## `goenv uninstall`

Uninstalls the specified version if it exists, otherwise - error.

```shell
> goenv uninstall 1.6.3
```

## `goenv update`

Update goenv to the latest version.

**Usage:**

```shell
# Update goenv
> goenv update
🔄 Checking for goenv updates...

✓ Updated to version 2.2.0

# Check for updates without installing
> goenv update --check
Current version: 2.1.0
Latest version: 2.2.0
Update available!

# Force update even if already up-to-date
> goenv update --force
```

### Options

- `--check` - Check for updates without installing
- `--force` - Force update even if already up-to-date

### Installation Methods

**Git-based installations (recommended):**
- Runs `git pull` in GOENV_ROOT directory
- Shows changes and new version

**Binary installations:**
- Downloads latest release from GitHub
- Replaces current binary
- Requires write permission to binary location

## `goenv update-tools`

Update installed Go tools to their latest versions.

**Usage:**

```shell
# Update all tools for current Go version
> goenv update-tools

# Check for updates without installing
> goenv update-tools --check

# Update only a specific tool
> goenv update-tools --tool gopls

# Show what would be updated (dry run)
> goenv update-tools --dry-run

# Update to specific version (default: latest)
> goenv update-tools --version v1.2.3
```

### Options

- `--check` - Check for updates without installing
- `--tool <name>` - Update only the specified tool
- `--dry-run` - Show what would be updated without actually updating
- `--version <version>` - Target version (default: latest)

This command updates tools installed with `go install` in your current Go version's GOPATH.

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

# Display bare version numbers only (without current marker)
> goenv versions --bare
1.4.0
1.4.1
1.4.2
1.5.0
1.5.1
1.6.0
1.6.1
1.6.2

# Skip aliases in output
> goenv versions --skip-aliases
  1.4.0
  1.5.0
* 1.6.1 (set by /home/go-nv/.goenv/version)
```

### Options

- `--bare` - Display bare version numbers only (no current marker, one per line)
- `--skip-aliases` - Skip aliases in the output

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
