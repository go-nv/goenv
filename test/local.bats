#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
}

@test "no version" {
  assert [ ! -e "${PWD}/.go-version" ]
  run goenv-local
  assert_failure "goenv: no local version configured for this directory"
}

@test "local version" {
  echo "1.2.3" > .go-version
  run goenv-local
  assert_success "1.2.3"
}

@test "discovers version file in parent directory" {
  echo "1.2.3" > .go-version
  mkdir -p "subdir" && cd "subdir"
  run goenv-local
  assert_success "1.2.3"
}

@test "ignores GOENV_DIR" {
  echo "1.2.3" > .go-version
  mkdir -p "$HOME"
  echo "3.4-home" > "${HOME}/.go-version"
  GOENV_DIR="$HOME" run goenv-local
  assert_success "1.2.3"
}

@test "sets local version" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.3
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.3" ]
}

@test "changes local version" {
  echo "1.0-pre" > .go-version
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local
  assert_success "1.0-pre"
  run goenv-local 1.2.3
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.3" ]
}

@test "unsets local version" {
  touch .go-version
  run goenv-local --unset
  assert_success ""
  assert [ ! -e .go-version ]
}
