# Command Reference

Like `git`, the `goenv` command delegates to subcommands based on its
first argument.

## 🚀 Modern Unified Commands (Recommended)

**New in v3.0**: Simplified, intuitive commands for common operations:

- **[`goenv use`](#goenv-use)** - Set Go version (replaces `local`/`global`)
- **[`goenv current`](#goenv-current)** - Show active version (replaces `version`)
- **[`goenv list`](#goenv-list)** - List versions (replaces `versions`/`installed`)

These unified commands provide a cleaner, more consistent interface. The legacy commands (`local`, `global`, `versions`) still work for backward compatibility.

## 📋 All Subcommands

- [Command Reference](#command-reference)
  - [🚀 Modern Unified Commands (Recommended)](#-modern-unified-commands-recommended)
  - [📋 All Subcommands](#-all-subcommands)
  - [`goenv use`](#goenv-use)
  - [`goenv current`](#goenv-current)
  - [`goenv list`](#goenv-list)
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
  - [`goenv sync-tools`](#goenv-sync-tools)
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
  - [`goenv vscode`](#goenv-vscode)
    - [`goenv vscode init`](#goenv-vscode-init)
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
  - [Version Discovery & Precedence](#version-discovery--precedence)
    - [Version Sources](#version-sources)
    - [Smart Precedence Rules](#smart-precedence-rules)
    - [Examples](#examples)
    - [Best Practices](#best-practices)
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
> goenv use stable --global

# Set local version using alias
> goenv use dev

# Aliases are resolved to their target versions
> goenv current
1.23.0 (set by /home/go-nv/.goenv/version)
```

**Alias features:**

- Aliases are stored in `~/.goenv/aliases` and persist across sessions
- Alias names cannot conflict with reserved keywords (`system`, `latest`)
- Aliases must contain only alphanumeric characters, hyphens, and underscores
- Aliases are automatically resolved when setting versions with `use`
- You can create aliases that point to special versions like `latest` or `system`

**Common use cases:**

```shell
# Track LTS versions
> goenv alias lts 1.22.5
> goenv use lts --global

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

## `goenv use`

**Modern unified command** for setting Go versions. Replaces `goenv local` and `goenv global` with a cleaner interface.

**Usage:**

```shell
# Set local version (creates .go-version in current directory)
> goenv use 1.25.2

# Set global version (updates ~/.goenv/version)
> goenv use 1.25.2 --global

# Show current version
> goenv use
1.25.2

# Unset local version (removes .go-version)
> goenv use --unset
```

**Options:**

- `--global` - Set global version instead of local
- `--unset` - Remove local .go-version file
- `--vscode` - Also update VS Code settings.json
- `--sync` - Sync go.mod toolchain directive

**Examples:**

```shell
# Set local version for current project
cd my-project
goenv use 1.24.8

# Set global default version
goenv use 1.25.2 --global

# Set version and update VS Code
goenv use 1.24.0 --vscode

# Sync with go.mod toolchain
goenv use 1.24.0 --sync
```

**Why use `goenv use`?**

- **Consistent**: One command for both local and global
- **Intuitive**: `use` is clearer than `local`/`global`
- **Modern**: Follows conventions from other version managers
- **Shorter**: `goenv use X` vs `goenv local X`

**Backward compatibility**: The legacy `goenv local` and `goenv global` commands still work but are hidden from help output.

## `goenv current`

**Modern unified command** for showing the active Go version and its source. Replaces `goenv version`.

**Usage:**

```shell
# Show current version with source
> goenv current
1.25.2 (set by /Users/user/project/.go-version)

# Show just the version number
> goenv current --bare
1.25.2

# Show the file that set the version
> goenv current --origin
/Users/user/project/.go-version
```

**Options:**

- `--bare` - Output only the version number
- `--origin` - Output only the source file path

**Examples:**

```shell
# Quick version check
> goenv current
1.24.8 (set by /Users/user/.goenv/version)

# Use in scripts
VERSION=$(goenv current --bare)
echo "Building with Go $VERSION"

# Check where version is set
> goenv current --origin
/Users/user/my-project/.go-version
```

**Why use `goenv current`?**

- **Clear**: "current" is more intuitive than "version"
- **Consistent**: Matches `goenv use` terminology
- **Modern**: Common pattern in version managers

**Backward compatibility**: The legacy `goenv version` command still works but is hidden from help output.

## `goenv list`

**Modern unified command** for listing Go versions. Replaces `goenv versions` and `goenv installed`.

**Usage:**

```shell
# List installed versions (default)
> goenv list
  1.23.5
  1.24.8
* 1.25.2 (set by /Users/user/.goenv/version)

# List available remote versions
> goenv list --remote
  1.20.0
  1.20.1
  ...
  1.25.2
  1.26rc1

# List only stable versions
> goenv list --remote --stable
  1.23.5
  1.24.8
  1.25.2

# Output bare version numbers
> goenv list --bare
1.23.5
1.24.8
1.25.2
```

**Options:**

- `--remote` - List available versions from remote (not installed)
- `--stable` - Filter to stable releases only (no beta/rc)
- `--bare` - Output only version numbers (no markers or descriptions)
- `--skip-aliases` - Don't show version aliases

**Examples:**

```shell
# See what's installed
> goenv list
  1.24.8
* 1.25.2 (set by /Users/user/.go-version)

# Check available versions before installing
> goenv list --remote --stable | tail -5
1.24.6
1.24.7
1.24.8
1.25.1
1.25.2

# Use in scripts
for version in $(goenv list --bare); do
  echo "Testing with Go $version"
  goenv use $version
  go test ./...
done
```

**Why use `goenv list`?**

- **Intuitive**: "list" is clearer than "versions"
- **Unified**: One command for both installed and available
- **Modern**: Shorter and more familiar command name
- **Smart default**: Shows installed by default (most common use case)

**Backward compatibility**: The legacy `goenv versions` and `goenv installed` commands still work but are hidden from help output.

- Common configuration problems

Use this command to troubleshoot issues with goenv.

## `goenv exec`

Run an executable with the selected Go version.

Assuming there's an already installed golang by e.g `goenv install 1.11.1` and
selected by e.g `goenv use 1.11.1 --global`,

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

### Using a Custom Mirror

For corporate proxies, regional mirrors, or air-gapped environments, you can specify a custom download mirror:

```shell
# Use a custom mirror for one installation
> GO_BUILD_MIRROR_URL=https://mirrors.example.com/golang/ goenv install 1.24.3

# Set permanently for all installations
> export GO_BUILD_MIRROR_URL=https://mirrors.example.com/golang/
> goenv install 1.24.3
```

Common use cases:

- **Corporate environments**: Use internal mirrors behind firewalls
- **China regions**: Use faster regional mirrors
- **Air-gapped networks**: Point to internal package repository
- **Testing**: Use custom build servers

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

**NEW: tfswitch-like behavior** - `goenv` (with no arguments) now automatically detects `.go-version` files:

```shell
# When you run goenv with no arguments in a directory with .go-version
> goenv
Found .go-version: 1.23.2
✓ Go 1.23.2 is installed

# If the version isn't installed, offers to install it
> goenv
Found .go-version: 1.23.2
⚠️  Go 1.23.2 is not installed
Install now? (Y/n) y
Installing Go 1.23.2...
✓ Installed Go 1.23.2

# Use --help to bypass detection and show help
> goenv --help
```

**NEW: Sync flag** - Ensure the version from `.go-version` is installed:

```shell
> goenv local --sync
Found .go-version: 1.23.2
✓ Go 1.23.2 is installed

# Or if not installed, automatically installs
> goenv local --sync
Found .go-version: 1.23.2
⚠️  Go 1.23.2 is not installed
Installing Go 1.23.2...
✓ Installed Go 1.23.2
```

This makes it easy to ensure your project's Go version is ready to use, similar to how `tfswitch` works for Terraform versions.

**Automatically set up VS Code integration when setting a local version:**

```shell
> goenv local 1.22.0 --vscode

Initializing VS Code workspace...
✓ Created/updated .vscode/settings.json
✓ Created/updated .vscode/extensions.json
✨ VS Code workspace configured for goenv!
```

The `--vscode` flag creates `.vscode/settings.json` and `.vscode/extensions.json` files configured to use goenv. This makes it easy to set up new projects with proper IDE integration in one command.

Previous versions of goenv stored local version specifications in a
file named `.goenv-version`. For backwards compatibility, goenv will
read a local version specified in an `.goenv-version` file, but a
`.go-version` file in the same directory will take precedence.

**Note:** goenv automatically discovers versions from both `.go-version` and `go.mod` files with smart precedence rules. See the [Version Discovery](#version-discovery--precedence) section below for details.

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

## `goenv sync-tools`

Sync/replicate installed Go tools from one Go version to another.

This command discovers tools in the source version and reinstalls them (from source) in the target version. The source version remains unchanged - think of this as "syncing" or "replicating" your tool setup rather than "moving" tools.

**Smart defaults:**
- No args: Sync from version with most tools → current version
- One arg: Sync from that version → current version
- Two args: Sync from source → target (explicit control)

**Usage:**

```shell
# Auto-sync: finds best source, syncs to current version
> goenv sync-tools

# Sync from specific version to current version
> goenv sync-tools 1.24.1

# Explicit source and target
> goenv sync-tools 1.24.1 1.25.2

# Preview auto-sync
> goenv sync-tools --dry-run

# Sync only specific tools
> goenv sync-tools 1.24.1 --select gopls,delve

# Exclude certain tools
> goenv sync-tools 1.24.1 --exclude staticcheck
```

### Options

- `--dry-run` / `-n` - Show what would be synced without actually syncing
- `--select <tools>` - Comma-separated list of tools to sync (e.g., gopls,delve)
- `--exclude <tools>` - Comma-separated list of tools to exclude from sync

This command is useful when upgrading Go versions and wanting to maintain your tool environment across versions.

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

## `goenv vscode`

Manage Visual Studio Code integration with goenv.

```shell
> goenv vscode
Commands to configure and manage Visual Studio Code integration with goenv

Available Commands:
  init        Initialize VS Code workspace for goenv
```

### `goenv vscode init`

Initialize VS Code workspace with goenv configuration.

This command creates or updates `.vscode/settings.json` and `.vscode/extensions.json` in the current directory to ensure VS Code uses the correct Go version from goenv.

**Usage:**

```shell
> goenv vscode init
✓ Created/updated /path/to/project/.vscode/settings.json
✓ Created/updated /path/to/project/.vscode/extensions.json

✨ VS Code workspace configured for goenv!
```

**Flags:**

- `--force` - Overwrite existing settings instead of merging
- `--template <name>` - Use specific template (basic, advanced, monorepo)

**Templates:**

| Template   | Description                                               |
| ---------- | --------------------------------------------------------- |
| `basic`    | Go configuration with goenv env vars (default)            |
| `advanced` | Includes gopls settings, format on save, organize imports |
| `monorepo` | Configured for large repositories with multiple modules   |

**Examples:**

```shell
# Basic setup (default)
> goenv vscode init

# Advanced setup with gopls configuration
> goenv vscode init --template advanced

# Force overwrite existing settings
> goenv vscode init --force

# Monorepo setup
> goenv vscode init --template monorepo
```

**What it does:**

1. Creates `.vscode/settings.json` with:

   - `go.goroot: ${env:GOROOT}` - Uses goenv's GOROOT
   - `go.gopath: ${env:GOPATH}` - Uses goenv's GOPATH
   - `go.toolsGopath` - Shared location for Go tools
   - Optional gopls and editor settings (advanced template)

2. Creates `.vscode/extensions.json` with:

   - Recommendation for the official Go extension (`golang.go`)

3. Merges with existing settings (unless `--force` is used)

**Integration with goenv use:**

You can also automatically set up VS Code when setting a Go version:

```shell
> goenv use 1.22.0 --vscode

Initializing VS Code workspace...
✓ Created/updated .vscode/settings.json
✓ Created/updated .vscode/extensions.json
✨ VS Code workspace configured for goenv!
```

**Doctor integration:**

The `goenv doctor` command checks your VS Code integration:

```shell
> goenv doctor
...
✅ VS Code integration
   VS Code configured to use goenv environment variables
```

See the [VS Code Integration Guide](../user-guide/VSCODE_INTEGRATION.md) for detailed setup instructions and troubleshooting.

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

---

## Version Discovery & Precedence

goenv automatically discovers the required Go version from multiple sources. When you run `goenv` or other commands, it searches for version information in this order:

### Version Sources

1. **GOENV_VERSION environment variable** (highest priority)

   - Set explicitly in your current shell
   - Example: `GOENV_VERSION=1.24.1 goenv version`

2. **.go-version file** (project-specific)

   - Simple text file with version number
   - Created with `goenv use <version>`
   - Searched from current directory up to root

3. **go.mod file** (Go module projects)

   - Uses `toolchain` directive (preferred)
   - Falls back to `go` directive if no toolchain
   - Standard Go toolchain mechanism

4. **~/.goenv/version** (global fallback)
   - Your default Go version
   - Set with `goenv use <version> --global`

### Smart Precedence Rules

When **both** `.go-version` and `go.mod` exist in the same directory:

| Scenario                                              | Result          | Reason                                      |
| ----------------------------------------------------- | --------------- | ------------------------------------------- |
| `.go-version = 1.25.0`<br>`go.mod toolchain = 1.24.1` | Uses **1.25.0** | User's explicit choice to use newer version |
| `.go-version = 1.24.1`<br>`go.mod toolchain = 1.24.1` | Uses **1.24.1** | Versions match (prefers user's file)        |
| `.go-version = 1.23.0`<br>`go.mod toolchain = 1.24.1` | Uses **1.24.1** | go.mod toolchain is project requirement     |

**Key principle:** go.mod's `toolchain` directive specifies the **minimum required version**. If your `.go-version` is older, goenv will use the go.mod version to ensure builds work correctly.

#### Interactive Mode

When goenv detects that `.go-version` is older than go.mod's toolchain, it will prompt:

```
⚠️  Your .go-version (1.23.0) is older than go.mod's toolchain requirement (1.24.1)
   Using 1.24.1 as required by go.mod

Update .go-version to 1.24.1 to avoid this warning? (Y/n)
```

This helps keep your version files in sync.

### Examples

**Example 1: Only .go-version exists**

```shell
> cat .go-version
1.24.1

> goenv version
1.24.1 (set by /path/to/project/.go-version)
```

**Example 2: Only go.mod exists**

```shell
> cat go.mod
module myproject

go 1.22
toolchain go1.24.1

> goenv version
1.24.1 (set by /path/to/project/go.mod)
```

**Example 3: Both exist, .go-version is newer**

```shell
> cat .go-version
1.25.0

> cat go.mod
module myproject
go 1.22
toolchain go1.24.1

> goenv version
1.25.0 (set by /path/to/project/.go-version)
# Uses .go-version because 1.25.0 >= 1.24.1 (requirement satisfied)
```

**Example 4: Both exist, .go-version is older**

```shell
> cat .go-version
1.23.0

> cat go.mod
module myproject
go 1.22
toolchain go1.24.1

> goenv
Found go.mod: 1.24.1
⚠️  Your .go-version (1.23.0) is older than go.mod's toolchain requirement (1.24.1)
   Using 1.24.1 as required by go.mod

Update .go-version to 1.24.1 to avoid this warning? (Y/n)
```

### Best Practices

#### For Go Module Projects

1. **Rely on go.mod** - It's the standard way to specify Go version requirements
2. **Optional .go-version** - Only create if you need to pin a specific version
3. **Keep in sync** - If you create .go-version, keep it >= go.mod toolchain

```shell
# Let go.mod manage version
go mod edit -toolchain=go1.24.1

# OR pin explicitly with .go-version
goenv use 1.24.1  # Creates .go-version
```

#### For Scripts/Non-Module Projects

Use `.go-version` as your primary version specification:

```shell
goenv use 1.24.1  # Creates .go-version
```

#### For CI/CD

Set `GOENV_AUTO_INSTALL=1` to automatically install the discovered version:

```yaml
# GitHub Actions example
env:
  GOENV_AUTO_INSTALL: 1

steps:
  - run: goenv # Auto-discovers and installs version from .go-version or go.mod
```

### Troubleshooting

**Q: Why is goenv using a different version than my .go-version?**

A: Check if you have a go.mod with a newer toolchain requirement. Run `goenv doctor` to see what version is discovered and why.

**Q: How do I force goenv to use my .go-version?**

A: Make sure your .go-version is >= the go.mod toolchain requirement. Update it with:

```shell
goenv use $(grep toolchain go.mod | awk '{print $2}' | sed 's/go//')
```

**Q: Can I disable go.mod version checking?**

A: No, this is intentional. Go's toolchain directive is a hard requirement for builds to work correctly. If you want to use a specific version, set it in .go-version to be >= the toolchain requirement.
