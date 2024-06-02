#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
}

@test "has usage instructions" {
  run goenv-help --usage local
  assert_success <<OUT
Usage: goenv local <version>
       goenv local --unset
OUT
}

@test "local has completion support" {
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  run goenv-installed --complete
  assert_success <<OUT
latest
system
1.10.9
1.9.10
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

@test "goenv local sets properly sorted latest version when 'latest' version is given and any version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.9.9"
  run goenv-local latest
  assert_success ""
  assert [ "$(cat .go-version)" = "1.10.10" ]
}

@test "goenv local sets latest version when major version is given and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-local 1
  assert_success ""
  run goenv-local
  assert_success "1.2.10"
}

@test "goenv local fails setting latest version when major or minor single number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.10"
  run goenv-local 9
  assert_failure "goenv: version '9' not installed"
}

@test "goenv local sets latest version when minor version is given as single number and any matching major.minor version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/4.5.2"
  run goenv-local 2
  assert_success ""
  run goenv-local
  assert_success "1.2.10"
}

@test "goenv local sets latest version when minor version is given as major.minor number and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.2"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/2.1.2"
  run goenv-local 1.2
  assert_success ""
  run goenv-local
  assert_success "1.2.10"
}

@test "goenv local fails setting latest version when major.minor number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.9"
  run goenv-local 1.9
  assert_failure "goenv: version '1.9' not installed"
}

@test "fails setting local version when version argument is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.4
  assert_failure "goenv: version '1.2.4' not installed"
}

@test "changes local existing version with version argument when version argument is given and does match at 'GOENV_ROOT/versions/<version>'" {
  echo "1.0-pre" > .go-version
  run goenv-local
  assert_success "1.0-pre"

  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local 1.2.3
  assert_success ""

  assert [ "$(cat .go-version)" = "1.2.3" ]
}
