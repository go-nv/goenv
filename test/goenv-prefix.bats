#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage prefix
  assert_success_out <<OUT
Usage: goenv prefix [<version>]
OUT
}

@test "has completion support" {
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  run goenv-prefix --complete
  assert_success_out <<OUT
latest
system
1.10.9
1.9.10
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

@test "returns GOENV_ROOT/versions/<version> when only major.minor version is given and it matches the latest patch version installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"

  run goenv-prefix 1.2
  assert_success_out <<OUT
${GOENV_ROOT}/versions/1.2.3
OUT
}

@test "fails when no '.go-version' exists, version is system and there's no 'go' executable in PATH" {
  run goenv-prefix
  assert_failure "goenv: system version not found in PATH"
}

@test "returns dir containing 'go' when no '.go-version' exists, GOENV_VERSION is system and there's 'go' executable in PATH" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"

  GOENV_VERSION="system" run goenv-prefix
  assert_success "$GOENV_TEST_DIR"
}

@test "system returns go if installed" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"

  run goenv-prefix system
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

@test "goenv prefix displays properly sorted latest version when 'latest' version is given and any version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.9.9"
  run goenv-prefix latest
  assert_success "${GOENV_ROOT}/versions/1.10.10"
}

@test "goenv prefix displays latest version when major version is given and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-prefix 1
  assert_success "${GOENV_ROOT}/versions/1.2.10"
}

@test "goenv local fails setting latest version when major or minor single number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.10"
  run goenv-prefix 9
  assert_failure "goenv: version '9' not installed"
}

@test "goenv prefix displays latest version when minor version is given as single number and any matching major.minor version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/4.5.2"
  run goenv-prefix 2
  assert_success "${GOENV_ROOT}/versions/1.2.10"
}

@test "goenv prefix displays latest version when minor version is given as major.minor number and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.2"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/2.1.2"
  run goenv-prefix 1.2
  assert_success "${GOENV_ROOT}/versions/1.2.10"
}

@test "goenv local fails setting latest version when major.minor number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.9"
  run goenv-prefix 1.9
  assert_failure "goenv: version '1.9' not installed"
}
