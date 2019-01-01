#!/usr/bin/env bats

load test_helper

@test "fails and prints help when no command argument is given" {
  run goenv
  assert_failure
  assert_output <<OUT
$(goenv---version)
Usage: goenv <command> [<args>]

Some useful goenv commands are:
   commands    List all available commands of goenv
   local       Set or show the local application-specific Go version
   global      Set or show the global Go version
   shell       Set or show the shell-specific Go version
   rehash      Rehash goenv shims (run this after installing executables)
   version     Show the current Go version and its origin
   versions    List all Go versions available to goenv
   which       Display the full path to an executable
   whence      List all Go versions that contain the given executable

See 'goenv help <command>' for information on a specific command.
For full documentation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails when invalid command argument is given" {
  run goenv does-not-exist
  assert_failure
  assert_output "goenv: no such command 'does-not-exist'"
}

@test "uses '\$HOME/.goenv' as default 'GOENV_ROOT' when 'GOENV_ROOT' environment variable is blank" {
  GOENV_ROOT="" HOME=/home/mislav run goenv root

  assert_success
  assert_output "/home/mislav/.goenv"
}

@test "uses provided 'GOENV_ROOT' as default 'GOENV_ROOT' when 'GOENV_ROOT' environment variable is provided" {
  GOENV_ROOT=/opt/goenv run goenv root

  assert_success
  assert_output "/opt/goenv"
}

@test "uses 'PWD' as default 'GOENV_DIR' when no 'GOENV_DIR' is specified" {
  run goenv echo GOENV_DIR
  assert_output "$(pwd)"
}

@test "uses provided 'GOENV_DIR' as default 'GOENV_DIR' when 'GOENV_DIR' environment variable is provided" {
  dir="${BATS_TMPDIR}/myproject"
  mkdir -p "$dir"
  GOENV_DIR="$dir" run goenv echo GOENV_DIR
  assert_output "$dir"
}

@test "fails when provided 'GOENV_DIR' environment variable cannot be changed current dir into" {
  dir="${BATS_TMPDIR}/does-not-exist"
  assert [ ! -d "$dir" ]

  GOENV_DIR="$dir" run goenv echo GOENV_DIR

  assert_failure
  assert_output "goenv: cannot change working directory to '$dir'"
}

@test "adds its own 'GOENV_ROOT/libexec' to PATH" {
  run goenv echo "PATH"
  assert_success "${BATS_TEST_DIRNAME%/*}/libexec:$PATH"
}

@test "adds plugin bin dirs 'GOENV_ROOT/{libexec,plugins}/<plugin>/<binary>' to PATH" {
  mkdir -p "$GOENV_ROOT"/plugins/go-build/bin
  mkdir -p "$GOENV_ROOT"/plugins/goenv-each/bin
  run goenv echo -F: "PATH"

  assert_success
  assert_line 0 "${BATS_TEST_DIRNAME%/*}/libexec"
  assert_line 1 "${GOENV_ROOT}/plugins/goenv-each/bin"
  assert_line 2 "${GOENV_ROOT}/plugins/go-build/bin"
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

@test "prints error when called with 'shell' subcommand, but `GOENV_SHELL` environment variable is not present" {
  unset GOENV_SHELL
  run goenv shell
  assert_output <<'OUT'
eval "$(goenv init -)" has not been executed.
Please read the installation instructions in the README.md at github.com/syndbg/goenv
or run 'goenv help init' for more information
OUT

  assert_failure
}
