#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version
  assert_success <<'OUT'
Usage: goenv version
OUT
}

@test "uses system one when no version arguments are given" {
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-version

  assert_success "system (set by ${GOENV_ROOT}/version)"
}

@test "uses version from 'GOENV_VERSION' environment variable when no version arguments are given and version is present in GOENV_ROOT/versions/<version>" {
  create_version "1.11.1"
  GOENV_VERSION=1.11.1 run goenv-version

  assert_success "1.11.1 (set by GOENV_VERSION environment variable)"
}

@test "uses version from '.go-version' local file when no version arguments are given and version is present in GOENV_ROOT/versions/<version>" {
  create_version "1.11.1"
  echo "1.11.1" > '.go-version'

  run goenv-version
  assert_success "1.11.1 (set by ${PWD}/.go-version)"
}

@test "uses version from 'GOENV_ROOT/version' file when no version arguments are given and version is present in GOENV_ROOT/versions/<version>" {
  create_version "1.11.1"
  echo "1.11.1" > "${GOENV_ROOT}/version"

  run goenv-version
  assert_success "1.11.1 (set by ${GOENV_ROOT}/version)"
}

@test "uses versions separated by ':' from 'GOENV_ROOT/version' file when no version arguments are given and versions are present in GOENV_ROOT/versions/<version>" {
  create_version "1.11.1"
  create_version "1.10.3"

  echo "1.11.1:1.10.3" > "${GOENV_ROOT}/version"

  run goenv-version
  assert_success
  assert_output <<OUT
1.11.1 (set by ${GOENV_ROOT}/version)
1.10.3 (set by ${GOENV_ROOT}/version)
OUT
}

@test "fails when versions separated by ':' from 'GOENV_VERSION' environment variable, no version arguments are given, but only one version is present in GOENV_ROOT/versions/<version>" {
  create_version "1.11.1"

  GOENV_VERSION=1.1:1.11.1:1.2 run goenv-version
  assert_failure
  assert_output <<OUT
goenv: version '1.1' is not installed (set by GOENV_VERSION environment variable)
goenv: version '1.2' is not installed (set by GOENV_VERSION environment variable)
1.11.1 (set by GOENV_VERSION environment variable)
OUT
}
