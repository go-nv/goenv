#!/usr/bin/env bats

load test_helper

@test "no shell version" {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
  echo "1.2.3" > .go-version
  GOENV_VERSION="" run goenv-sh-shell
  assert_failure "goenv: no shell-specific version configured"
}

@test "shell version" {
  GOENV_SHELL=bash GOENV_VERSION="1.2.3" run goenv-sh-shell
  assert_success 'echo "$GOENV_VERSION"'
}

@test "shell version (fish)" {
  GOENV_SHELL=fish GOENV_VERSION="1.2.3" run goenv-sh-shell
  assert_success 'echo "$GOENV_VERSION"'
}

@test "shell unset" {
  GOENV_SHELL=bash run goenv-sh-shell --unset
  assert_success "unset GOENV_VERSION"
}

@test "shell unset (fish)" {
  GOENV_SHELL=fish run goenv-sh-shell --unset
  assert_success "set -e GOENV_VERSION"
}

@test "shell change invalid version" {
  run goenv-sh-shell 1.2.3
  assert_failure
  assert_output <<SH
goenv: version \`1.2.3' not installed
false
SH
}

@test "shell change version" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  GOENV_SHELL=bash run goenv-sh-shell 1.2.3
  assert_success 'export GOENV_VERSION="1.2.3"'
}

@test "shell change version (fish)" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  GOENV_SHELL=fish run goenv-sh-shell 1.2.3
  assert_success 'setenv GOENV_VERSION "1.2.3"'
}
