# How It Works

At a high level, goenv intercepts Go commands using shim
executables injected into your `PATH`, determines which Go version
has been specified by your application, and passes your commands along
to the correct Go installation.

**Note:** goenv is now written in Go (migrated from shell scripts), providing better cross-platform support including native Windows compatibility!

## Understanding PATH

When you run all the variety of Go commands using `go`, your operating system
searches through a list of directories to find an executable file with
that name. This list of directories lives in an environment variable
called `PATH`, with each directory in the list separated by a platform-specific separator:

**Unix/Linux/macOS:**

```
/usr/local/bin:/usr/bin:/bin
```

(Directories separated by colons `:`)

**Windows:**

```
C:\Program Files\Go\bin;C:\Windows\System32;C:\Windows
```

(Directories separated by semicolons `;`)

Directories in `PATH` are searched from left to right, so a matching
executable in a directory at the beginning of the list takes
precedence over another one at the end. In the Unix example above, the
`/usr/local/bin` directory will be searched first, then `/usr/bin`,
then `/bin`.

## Understanding Shims

goenv works by inserting a directory of _shims_ at the front of your
`PATH`:

**Unix/Linux/macOS:**

```
~/.goenv/shims:/usr/local/bin:/usr/bin:/bin
```

**Windows:**

```
%USERPROFILE%\.goenv\shims;C:\Program Files\Go\bin;C:\Windows\System32
```

Through a process called _rehashing_, goenv maintains shims in that
directory to match every `go` command across every installed version
of Go.

### Shim Implementation

Shims are lightweight executables that simply pass your command along
to goenv:

- **Unix/Linux/macOS**: Bash scripts (`go`, `gofmt`, `godoc`, etc.)
- **Windows**: Batch files (`go.bat`, `gofmt.bat`, `godoc.bat`, etc.)

The goenv binary (written in Go) then determines which Go version to use and executes the appropriate command.

So with goenv installed, when you run `go` your operating system will do the following:

- Search your `PATH` for an executable file named `go` (or `go.bat` on Windows)
- Find the goenv shim named `go` at the beginning of your `PATH`
- Run the shim, which calls `goenv exec go <your-args>`
- goenv determines the version and runs the correct Go binary

## Choosing the Go Version

When you execute a shim, goenv determines which Go version to use by
reading it from the following sources, in this order:

1. The `GOENV_VERSION` environment variable (if specified). You can use
   the [`goenv shell`](https://github.com/go-nv/goenv/blob/master/COMMANDS.md#goenv-shell) command to set this environment
   variable in your current shell session.

2. The application-specific `.go-version` file in the current
   directory (if present). You can modify the current directory's
   `.go-version` file with the [`goenv use`](https://github.com/go-nv/goenv/blob/master/COMMANDS.md#goenv-use)
   command.

3. The first `.go-version` file found (if any) by searching each parent
   directory, until reaching the root of your filesystem.

4. The global `~/.goenv/version` file. You can modify this file using
   the [`goenv use --global`](https://github.com/go-nv/goenv/blob/master/COMMANDS.md#goenv-use) command. If the global version
   file is not present, goenv assumes you want to use the "system"
   Go. (In other words, whatever version would run if goenv isn't present in
   `PATH`.)

**NOTE:** The precedence system allows you to work on multiple projects with different Go versions simultaneously. Each shell session or directory can have its own active version (via `goenv shell` or `.go-version` files), but only **one Go version is active** in any given context. For example, you can have Go 1.22.0 active in one terminal and Go 1.23.2 active in another terminal or project directory—the versions don't conflict because each context resolves to a single version through the precedence chain above.

## Locating the Go Installation

Once goenv has determined which version of Go your application has
specified, it passes the command along to the corresponding Go
installation.

Each Go version is installed into its own directory under the goenv root:

**Unix/Linux/macOS:**

```
~/.goenv/versions/1.21.0/
~/.goenv/versions/1.22.0/
```

**Windows:**

```
%USERPROFILE%\.goenv\versions\1.21.0\
%USERPROFILE%\.goenv\versions\1.22.0\
```

Each version directory contains:

```
versions/1.21.0/
├── bin/           # Go binaries (go, gofmt, etc.)
├── src/           # Go source code
├── pkg/           # Compiled packages
└── ...
```

As far as goenv is concerned, version names are simply the directories in
the versions directory.

## GOPATH Integration: Where Tools Land

When you install tools with `go install`, goenv automatically manages version-specific GOPATH directories:

**Default structure:**

```
$HOME/go/
├── 1.21.0/
│   ├── bin/           # Tools installed with Go 1.21.0
│   │   ├── gopls
│   │   ├── goimports
│   │   └── ...
│   ├── pkg/
│   └── src/
├── 1.22.0/
│   ├── bin/           # Tools installed with Go 1.22.0
│   │   ├── gopls
│   │   └── ...
│   └── ...
└── 1.23.2/
    └── bin/           # Tools installed with Go 1.23.2
        └── ...
```

**How it works:**

1. **Installation:** When you run `go install`, the tool is placed in `$HOME/go/{version}/bin/`
2. **Shim creation:** Running `goenv rehash` (or `goenv use`) creates shims in `~/.goenv/shims/`
3. **Version routing:** Shims automatically route to the active Go version's GOPATH bin directory
4. **Isolation:** Tools from different Go versions never conflict

**Example workflow:**

```bash
# Using Go 1.22.0
goenv use 1.22.0
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# → Installs to: $HOME/go/1.22.0/bin/golangci-lint

goenv rehash
golangci-lint version  # ✅ Uses tool from Go 1.22.0's GOPATH

# Switch to Go 1.21.0
goenv use 1.21.0
golangci-lint version  # ⚠️  command not found (not installed for this version)

# Install for this version
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# → Installs to: $HOME/go/1.21.0/bin/golangci-lint

goenv rehash
golangci-lint version  # ✅ Now uses tool from Go 1.21.0's GOPATH
```

**Configuration:**

- `GOENV_GOPATH_PREFIX` - Change base directory (default: `$HOME/go`)
- `GOENV_DISABLE_GOPATH` - Set to `1` to disable version-specific GOPATH

See [GOPATH Integration Guide](../advanced/GOPATH_INTEGRATION.md) for complete details, configuration options, and advanced usage.

## Implementation

goenv is written in Go and consists of:

- **Main binary**: `goenv` (or `goenv.exe` on Windows)

  - Handles version resolution
  - Manages installations
  - Executes commands with the correct Go version

- **Shims**: Platform-specific lightweight executables

  - Unix: Bash scripts that call `goenv exec`
  - Windows: Batch files that call `goenv.exe exec`

- **Configuration files**:
  - `~/.goenv/version` - Global version setting
  - `.go-version` - Per-directory version setting
  - `GOENV_VERSION` - Environment variable override

For more details on the Go-based implementation and Windows support, see [WINDOWS_SUPPORT.md](../../WINDOWS_SUPPORT.md).
