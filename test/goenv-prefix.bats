#!/usr/bin/env bats

load test_helper

@test "prefix" {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
  echo "1.2.3" > .go-version
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-prefix
  assert_success "${GOENV_ROOT}/versions/1.2.3"
}

@test "prefix for invalid version" {
  GOENV_VERSION="1.2.3" run goenv-prefix
  assert_failure "goenv: version \`1.2.3' not installed"
}

@test "prefix for system" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"
  GOENV_VERSION="system" run goenv-prefix
  assert_success "$GOENV_TEST_DIR"
}

@test "prefix for invalid system" {
  PATH="$(path_without go)" run goenv-prefix system
  assert_failure "goenv: system version not found in PATH"
}
