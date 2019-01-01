#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
}

@test "has usage instructions" {
  run goenv-help --usage local
  assert_success <<'OUT'
Usage: goenv local <version>
       goenv local --unset
OUT
}

@test "has completion support" {
  run goenv-local --complete
  assert_success <<OUT
--unset
system
OUT
}

@test "fails when no arguments are given and there's no '.go-version' file in current dir" {
  assert [ ! -e "${PWD}/.go-version" ]
  run goenv-local
  assert_failure "goenv: no local version configured for this directory"
}

@test "prints local version when no arguments are given and there's a '.go-version' file in current dir" {
  echo "1.2.3" > .go-version
  run goenv-local
  assert_success "1.2.3"
}

@test "removes '.go-version' in current dir when '--unset' argument is given and there's a '.go-version' file in current dir" {
  touch .go-version
  run goenv-local --unset
  assert_success ""
  assert [ ! -e .go-version ]
}

@test "succeeds and does nothing when '--unset' argument is given and there's no '.go-version' file in current dir" {
  run goenv-local --unset
  assert_success ""
  assert [ ! -e .go-version ]
}

@test "prints local version '.go-version' file in parent directory when no arguments are given" {
  echo "1.2.3" > .go-version
  mkdir -p "subdir"
  cd "subdir"

  run goenv-local
  assert_success "1.2.3"
}

@test "reads local version '.go-version' file in current dir with priority over specified GOENV_DIR with '.go-version' file when no arguments are given" {
  echo "1.2.3" > .go-version
  mkdir -p "$HOME"

  echo "1.4-home" > "${HOME}/.go-version"
  GOENV_DIR="$HOME" run goenv-local

  assert_success "1.2.3"
}

@test "sets local version when version argument is given and matches at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.3
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.3" ]
}

@test "fails setting local version when version argument is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.4
  assert_failure "goenv: version '1.2.4' not installed"
}

@test "changes local existing version with version argument when  version argument is given and does match at 'GOENV_ROOT/versions/<version>'" {
  echo "1.0-pre" > .go-version
  run goenv-local
  assert_success "1.0-pre"

  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.3
  assert_success ""

  assert [ "$(cat .go-version)" = "1.2.3" ]
}
