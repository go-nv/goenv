#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage exec
  assert_success "Usage: goenv exec <command> [arg1 arg2...]"
}

@test "fails with usage instructions when no command is specified" {
  run goenv-exec
  assert_failure
  assert_output <<OUT
Usage: goenv exec <command> [arg1 arg2...]
OUT
}

@test "fails with version that's not installed but specified by GOENV_VERSION" {
  export GOENV_VERSION="1.6.1"
  run goenv-exec go version
  assert_failure "goenv: version '1.6.1' is not installed (set by GOENV_VERSION environment variable)"
}

@test "fails with version that's not installed but specified by '.go-version' file" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  echo 1.6.1 > .go-version

  run goenv-exec go build
  assert_failure "goenv: version '1.6.1' is not installed (set by $PWD/.go-version)"
}

@test "succeeds with version that's installed and specified by GOENV_VERSION" {
  export GOENV_VERSION="1.6.1"
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  goenv-rehash
  run goenv-exec Zgo123unique

  assert_output ""
  assert_success
}

@test "succeeds with version that's installed and specified by '.go-version' file" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  echo 1.6.1 > .go-version
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  goenv-rehash
  run goenv-exec Zgo123unique
  assert_success
}

@test "completes with names of executables for version that's specified by GOENV_VERSION environment variable" {
  export GOENV_VERSION="1.6.1"
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  goenv-rehash
  run goenv-completions exec
  assert_success
  assert_output <<OUT
--help
Zgo123unique
OUT
}

@test "carries original IFS within hooks for version that's specified by GOENV_VERSION environment variable" {
  create_hook exec hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=system
  IFS=$' \t\n' run goenv-exec env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "forwards all arguments for command that's specified by GOENV_VERSION environment variable" {
  export GOENV_VERSION="1.6.1"
  create_executable "1.6.1" "go" <<SH
#!$BASH
echo \$0
for arg; do
  # hack to avoid bash builtin echo which can't output '-e'
  printf "  %s\\n" "\$arg"
done
SH

  run goenv-exec go run "/path to/go script.go" -- extra args
  assert_success
  assert_output <<OUT
${GOENV_ROOT}/versions/1.6.1/bin/go
  run
  /path to/go script.go
  --
  extra
  args
OUT
}
