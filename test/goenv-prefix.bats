#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage prefix
  assert_success <<'OUT'
Usage: goenv prefix [<version>]
OUT
}

@test "has completion support" {
  run goenv-local --complete
  assert_success <<OUT
--unset
OUT
}

@test "returns GOENV_ROOT/versions/<version> that matches current dir's '.go-version' file's version when no arguments and '.go-version' file exists" {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
  echo "1.2.3" > .go-version
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"

  run goenv-prefix
  assert_success "${GOENV_ROOT}/versions/1.2.3"
}

@test "fails when no '.go-version' exists, version is system and there's no 'go' executable in PATH" {
  run goenv-prefix
  assert_failure "goenv: system version not found in PATH"
}

@test "returns dir containing 'go' when no '.go-version' exists, version is system and there's 'go' executable in PATH" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"

  GOENV_VERSION="system" run goenv-prefix
  assert_success "$GOENV_TEST_DIR"
}

@test "fails when '.go-version' exists but it does not match any 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
  echo "1.2.3" > .go-version

  run goenv-prefix
  assert_failure "goenv: version '1.2.3' is not installed (set by $GOENV_TEST_DIR/myproject/.go-version)"
}

@test "prints version prioritized from arguments rather than 'GOENV_VERSION' environment variable" {
  GOENV_VERSION="1.2.3" run goenv-prefix 1.2.4
  assert_failure "goenv: version '1.2.4' not installed"
}

@test "prints version from 'GOENV_VERSION' environment variable when no arguments are given" {
  GOENV_VERSION="1.2.3" run goenv-prefix
  assert_failure "goenv: version '1.2.3' not installed"
}
