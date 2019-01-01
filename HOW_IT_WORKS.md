# How It Works

At a high level, goenv intercepts Go commands using shim
executables injected into your `PATH`, determines which Go version
has been specified by your application, and passes your commands along
to the correct Go installation.

## Understanding PATH

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

## Understanding Shims

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

## Choosing the Go Version

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

## Locating the Go Installation

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

