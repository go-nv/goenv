#!/usr/bin/env bats

load test_helper

teardown() {
  rm -rf "${GOENV_ROOT}/versions/" || true
  rm -f .go-version || true
}

@test "goenv has completion support" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  run goenv --complete
  assert_success <<OUT
1.10.9
1.9.10
commands
completions
exec
global
help
hooks
init
install
installed
latest
local
prefix
rehash
root
shell
shims
system
uninstall
version
version-file
version-file-read
version-file-write
version-name
version-origin
versions
whence
which
OUT
}

@test "fails and prints help when no command argument is given" {
  run goenv
  assert_failure <<OUT
$(goenv---version)
Usage: goenv <command> [<args>]

Some useful goenv commands are:
   commands    List all available commands of goenv
   local       Set or show the local application-specific Go version
   global      Set or show the global Go version
   shell       Set or show the shell-specific Go version
   install     Install a Go version using go-build
   uninstall   Uninstall a specific Go version
   rehash      Rehash goenv shims (run this after installing executables)
   version     Show the current Go version and its origin
   versions    List all Go versions available to goenv
   which       Display the full path to an executable
   whence      List all Go versions that contain the given executable

See 'goenv help <command>' for information on a specific command.
For full documentation, see: https://github.com/go-nv/goenv#readme
OUT
}

@test "Runs install when no command argument is given and GOENV_AUTO_INSTALL is set to 1" {
  export GOENV_AUTO_INSTALL=1

  echo "Path is $PATH"

  run goenv

  unset GOENV_AUTO_INSTALL
  
  assert_failure <<OUT
Usage: goenv install [-f] [-kvpq] <version>|latest|unstable
       goenv install [-f] [-kvpq] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  -q/--quiet         Disable Progress Bar
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/go-nv/goenv#readme
OUT
}

@test "fails when invalid command argument is given" {
  run goenv does-not-exist
  assert_failure "goenv: no such command 'does-not-exist'"
}

@test "uses '\$HOME/.goenv' as default 'GOENV_ROOT' when 'GOENV_ROOT' environment variable is blank" {
  GOENV_ROOT="" HOME=/home/mislav run goenv root

  assert_success "/home/mislav/.goenv"
}

@test "uses provided 'GOENV_ROOT' as default 'GOENV_ROOT' when 'GOENV_ROOT' environment variable is provided" {
  GOENV_ROOT=/opt/goenv run goenv root

  assert_success "/opt/goenv"
}

@test "uses 'PWD' as default 'GOENV_DIR' when no 'GOENV_DIR' is specified" {
  run goenv echo GOENV_DIR
  assert_success "$(pwd)"
}

@test "uses provided 'GOENV_DIR' as default 'GOENV_DIR' when 'GOENV_DIR' environment variable is provided" {
  dir="${BATS_TMPDIR}/myproject"
  mkdir -p "$dir"
  GOENV_DIR="$dir" run goenv echo GOENV_DIR
  assert_success "$dir"
}

@test "fails when provided 'GOENV_DIR' environment variable cannot be changed current dir into" {
  dir="${BATS_TMPDIR}/does-not-exist"
  assert [ ! -d "$dir" ]

  GOENV_DIR="$dir" run goenv echo GOENV_DIR

  assert_failure "goenv: cannot change working directory to '$dir'"
}

@test "adds its own 'GOENV_ROOT/libexec' to PATH" {
  run goenv echo "PATH"
  assert_success "${BATS_TEST_DIRNAME%/*}/libexec:${BATS_TEST_DIRNAME%/*}/plugins/go-build/bin:$PATH"
}

@test "adds plugin bin dirs 'GOENV_ROOT/{libexec,plugins}/<plugin>/<binary>' to PATH" {
  mkdir -p "$GOENV_ROOT"/plugins/go-build/bin
  mkdir -p "$GOENV_ROOT"/plugins/goenv-each/bin
  run goenv echo -F: "PATH"

  assert_success
  assert_line 0 "${BATS_TEST_DIRNAME%/*}/libexec"
  assert_line 1 "${GOENV_ROOT}/plugins/goenv-each/bin"
  assert_line 2 "${GOENV_ROOT}/plugins/go-build/bin"
  assert_line 3 "${BATS_TEST_DIRNAME%/*}/plugins/go-build/bin"
}

@test "'GOENV_HOOK_PATH' uses already defined 'GOENV_HOOK_PATH' in environment variable" {
  GOENV_HOOK_PATH=/my/hook/path:/other/hooks run goenv echo -F: "GOENV_HOOK_PATH"
  assert_success
  assert_line 0 "/my/hook/path"
  assert_line 1 "/other/hooks"
  assert_line 2 "${GOENV_ROOT}/goenv.d"
}

@test "'GOENV_HOOK_PATH' includes goenv built-in plugins paths" {
  unset GOENV_HOOK_PATH
  run goenv echo "GOENV_HOOK_PATH"
  assert_success "${GOENV_ROOT}/goenv.d:${BATS_TEST_DIRNAME%/*}/goenv.d:/usr/local/etc/goenv.d:/etc/goenv.d:/usr/lib/goenv/hooks"
}

@test "prints error when called with 'shell' subcommand, but $(GOENV_SHELL) environment variable is not present" {
  unset GOENV_SHELL
  run goenv shell
  assert_failure <<OUT
eval "$(goenv init -)" has not been executed.
Please read the installation instructions in the README.md at github.com/go-nv/goenv
or run 'goenv help init' for more information
OUT
}

@test "goenv sets properly sorted latest local version when 'latest' version is given to goenv and any version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.9.9"
  run goenv-local latest
  assert_success ""
  assert [ "$(cat .go-version)" = "1.10.10" ]
}

@test "goenv sets latest local version when major version is given and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-local 1
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.10" ]
}

@test "goenv fails setting latest local version when major or minor single number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.10"
  run goenv-local 9
  assert_failure "goenv: version '9' not installed"
}

@test "goenv sets latest local version when minor version is given as single number and any matching major.minor version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/4.5.2"
  run goenv-local 2
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.10" ]
}

@test "goenv sets latest local version when minor version is given as major.minor number and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.2.2"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/2.1.2"
  run goenv-local 1.2
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.10" ]
}

@test "goenv fails setting latest local version when major.minor number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.9"
  run goenv-local 1.9
  assert_failure "goenv: version '1.9' not installed"
}

