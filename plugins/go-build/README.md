# go-build

go-build is a [goenv](https://github.com/syndbg/goenv) plugin that
provides a `goenv install` command to compile and install different versions
of Go on UNIX-like systems.

You can also use go-build without goenv in environments where you need
precise control over Go version installation.

See the [list of releases](https://github.com/syndbg/goenv/releases)
for changes in each version.


## Installation

### Installing as an goenv plugin (recommended)

You need nothing to do since go-build is bundled with goenv by
default.

### Installing as a standalone program (advanced)

Installing go-build as a standalone program will give you access to the
`go-build` command for precise control over Go version installation. If you
have goenv installed, you will also be able to use the `goenv install` command.

    git clone git://github.com/syndbg/goenv.git
    cd goenv/plugins/go-build
    ./install.sh

This will install go-build into `/usr/local`. If you do not have write
permission to `/usr/local`, you will need to run `sudo ./install.sh` instead.
You can install to a different prefix by setting the `PREFIX` environment
variable.

To update go-build after it has been installed, run `git pull` in your cloned
copy of the repository, then re-run the install script.

### (TODO) Installing with Homebrew (for OS X users)

Mac OS X users can install go-build with the [Homebrew](http://brew.sh)
package manager. This will give you access to the `go-build` command. If you
have goenv installed, you will also be able to use the `goenv install` command.

*This is the recommended method of installation if you installed goenv with
Homebrew.*

    brew install goenv

Or, if you would like to install the latest development release:

    brew install --HEAD goenv

## Usage

Before you begin, you should ensure that your build environment has the proper
system dependencies for compiling the wanted Go Version (see our [recommendations](https://github.com/syndbg/goenv/wiki#suggested-build-environment)).

### Using `goenv install` with goenv

To install a Go version for use with goenv, run `goenv install` with
exact name of the version you want to install. For example,

    goenv install 1.11.1

Golang versions will be installed into a directory of the same name under
`~/.goenv/versions`.

To see a list of all available Golang versions, run `goenv install --list`. You
may also tab-complete available Golang versions if your goenv installation is
properly configured.

### Using `go-build` standalone

If you have installed go-build as a standalone program, you can use the
`gp-build` command to compile and install Golang versions into specific
locations.

Run the `go-build` command with the exact name of the version you want to
install and the full path where you want to install it. For example,

    go-build 1.6.2 ~/local/goenv-1.6.2

To see a list of all available Golang versions, run `go-build --definitions`.

Pass the `-v` or `--verbose` flag to `go-build` as the first argument to see
what's happening under the hood.

### Custom definitions

Both `goenv install` and `go-build` accept a path to a custom definition file
in place of a version name. Custom definitions let you develop and install
versions of Golang that are not yet supported by go-build.

See the [go-build built-in definitions](https://github.com/syndbg/goenv/tree/master/plugins/go-build/share/go-build) as a starting point for
custom definition files.

[definitions]: https://github.com/syndbg/goenv/tree/master/plugins/go-build/share/go-build

### Special environment variables

You can set certain environment variables to control the build process.

* `TMPDIR` sets the location where go-build stores temporary files.
* `GO_BUILD_BUILD_PATH` sets the location in which sources are downloaded and
  built. By default, this is a subdirectory of `TMPDIR`.
* `GO_BUILD_CACHE_PATH`, if set, specifies a directory to use for caching
  downloaded package files.
* `GO_BUILD_MIRROR_URL` overrides the default mirror URL root to one of your
  choosing.
* `GO_BUILD_SKIP_MIRROR`, if set, forces go-build to download packages from
  their original source URLs instead of using a mirror.
* `GO_BUILD_ROOT` overrides the default location from where build definitions
  in `share/go-build/` are looked up.
* `GO_BUILD_DEFINITIONS` can be a list of colon-separated paths that get
  additionally searched when looking up build definitions.
* `CC` sets the path to the C compiler.
* `GO_CFLAGS` lets you pass additional options to the default `CFLAGS`. Use
  this to override, for instance, the `-O3` option.
* `CONFIGURE_OPTS` lets you pass additional options to `./configure`.
* `MAKE` lets you override the command to use for `make`. Useful for specifying
  GNU make (`gmake`) on some systems.
* `MAKE_OPTS` (or `MAKEOPTS`) lets you pass additional options to `make`.
* `MAKE_INSTALL_OPTS` lets you pass additional options to `make install`.
* `GO_CONFIGURE_OPTS` and `GO_MAKE_OPTS` and `GO_MAKE_INSTALL_OPTS` allow
  you to specify configure and make options for buildling CPython. These variables
  will be passed to Golang only, not any dependent packages (e.g. libyaml).

### Applying patches to Golang before compiling

Both `goenv install` and `go-build` support the `--patch` (`-p`) flag that
signals that a patch from stdin should be applied to Golang source code before
the `./configure` and compilation steps.

Example usage:

```sh
# applying a single patch
$ goenv install --patch 1.11.1 < /path/to/golang.patch

# applying a patch from HTTP
$ goenv install --patch 1.11.1 < <(curl -sSL http://git.io/golang.patch)

# applying multiple patches
$ cat fix1.patch fix2.patch | goenv install --patch 1.11.1
```

### Checksum verification

If you have the `shasum`, `openssl`, or `sha256sum` tool installed, go-build will
automatically verify the SHA2 checksum of each downloaded package before
installing it.

Checksums are optional and specified as anchors on the package URL in each
definition. (All bundled definitions include checksums.)

### Package download mirrors

go-build will first attempt to download package files from a mirror hosted on
GitHub Pages. If a package is not available on the mirror, if the mirror
is down, or if the download is corrupt, go-build will fall back to the
official URL specified in the defintion file.

You can point go-build to another mirror by specifying the
`GO_BUILD_MIRROR_URL` environment variable--useful if you'd like to run your
own local mirror, for example. Package mirror URLs are constructed by joining
this variable with the SHA2 checksum of the package file.

If you don't have an SHA2 program installed, go-build will skip the download
mirror and use official URLs instead. You can force go-build to bypass the
mirror by setting the `GO_BUILD_SKIP_MIRROR` environment variable.

### Package download caching

You can instruct go-build to keep a local cache of downloaded package files
by setting the `GO_BUILD_CACHE_PATH` environment variable. When set, package
files will be kept in this directory after the first successful download and
reused by subsequent invocations of `go-build` and `goenv install`.

The `goenv install` command defaults this path to `~/.goenv/cache`, so in most
cases you can enable download caching simply by creating that directory.

### Keeping the build directory after installation

Both `go-build` and `goenv install` accept the `-k` or `--keep` flag, which
tells go-build to keep the downloaded source after installation. This can be
useful if you need to use `gdb` and `memprof` with Golang.

Source code will be kept in a parallel directory tree `~/.goenv/sources` when
using `--keep` with the `goenv install` command. You should specify the
location of the source code with the `GOLANG_BUILD_BUILD_PATH` environment
variable when using `--keep` with `go-build`.

## Getting Help

If you can't find an answer on the, open an issue on the [issue
tracker](https://github.com/syndbg/goenv/issues). Be sure to include
the full build log for build failures.
