#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage shell
  assert_success
  assert_output <<'OUT'
Usage: goenv shell <version>
       goenv shell --unset
OUT
}

@test "has completion support" {
  run goenv-sh-shell --complete
  assert_success
  assert_output <<OUT
--unset
system
OUT
}

@test "fails when there's no go version specified in arguments and in 'GOENV_VERSION' environment variable" {
  GOENV_VERSION="" run goenv-sh-shell
  assert_failure "goenv: no shell-specific version configured"
}

@test "prints 'GOENV_VERSION' when there's no go version specified in arguments, but there's one in 'GOENV_VERSION' environment variable" {
  GOENV_VERSION="1.2.3" run goenv-sh-shell
  assert_success 'echo "$GOENV_VERSION"'
}

@test "prints unset variable when '--unset' is given in arguments and shell is 'bash'" {
  GOENV_SHELL=bash run goenv-sh-shell --unset
  assert_success 'unset GOENV_VERSION'
}

@test "prints unset variable when '--unset' is given in arguments and shell is 'zsh'" {
  GOENV_SHELL=zsh run goenv-sh-shell --unset
  assert_success 'unset GOENV_VERSION'
}

@test "prints unset variable when '--unset' is given in arguments and shell is 'ksh'" {
  GOENV_SHELL=ksh run goenv-sh-shell --unset
  assert_success 'unset GOENV_VERSION'
}

@test "prints unset variable when '--unset' is given in arguments and shell is 'fish'" {
  GOENV_SHELL=fish run goenv-sh-shell --unset
  assert_success 'set -e GOENV_VERSION'
}

@test "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'bash'" {
  mkdir -p ${GOENV_ROOT}/versions/1.2.3

  GOENV_SHELL=bash run goenv-sh-shell 1.2.3

  assert_output 'export GOENV_VERSION="1.2.3"'
  assert_success
}

@test "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'zsh'" {
  mkdir -p ${GOENV_ROOT}/versions/1.2.3

  GOENV_SHELL=zsh run goenv-sh-shell 1.2.3

  assert_output 'export GOENV_VERSION="1.2.3"'
  assert_success
}

@test "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'ksh'" {
  mkdir -p ${GOENV_ROOT}/versions/1.2.3

  GOENV_SHELL=ksh run goenv-sh-shell 1.2.3

  assert_output 'export GOENV_VERSION="1.2.3"'
  assert_success
}

@test "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'fish'" {
  mkdir -p ${GOENV_ROOT}/versions/1.2.3

  GOENV_SHELL=fish run goenv-sh-shell 1.2.3

  assert_output 'set -gx GOENV_VERSION "1.2.3"'
  assert_success
}

@test "fails changing 'GOENV_VERSION' environment variable to specified shell version argument if version does not exist in GOENV_ROOT/versions/<version>" {
  GOENV_SHELL=bash run goenv-sh-shell 1.2.3

  assert_output <<OUT
goenv: version '1.2.3' not installed
false
OUT

  assert_failure
}
