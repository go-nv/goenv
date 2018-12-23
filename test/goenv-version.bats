#!/usr/bin/env bats

load test_helper

create_version() {
  mkdir -p "${GOENV_ROOT}/versions/$1"
}

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "no version selected" {
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-version
  assert_success "system (set by ${GOENV_ROOT}/version)"
}

@test "set by GOENV_VERSION" {
  create_version "3.3.3"
  GOENV_VERSION=3.3.3 run goenv-version
  assert_success "3.3.3 (set by GOENV_VERSION environment variable)"
}

@test "set by local file" {
  create_version "3.3.3"
  cat > ".go-version" <<<"3.3.3"
  run goenv-version
  assert_success "3.3.3 (set by ${PWD}/.go-version)"
}

@test "set by global file" {
  create_version "3.3.3"
  cat > "${GOENV_ROOT}/version" <<<"3.3.3"
  run goenv-version
  assert_success "3.3.3 (set by ${GOENV_ROOT}/version)"
}

@test "set by GOENV_VERSION, one missing" {
  create_version "3.3.3"
  GOENV_VERSION=3.3.3:1.2 run goenv-version
  assert_failure
  assert_output <<OUT
goenv: version \`1.2' is not installed (set by GOENV_VERSION environment variable)
3.3.3 (set by GOENV_VERSION environment variable)
OUT
}

@test "set by GOENV_VERSION, two missing" {
  create_version "3.3.3"
  GOENV_VERSION=3.4.2:3.3.3:1.2 run goenv-version
  assert_failure
  assert_output <<OUT
goenv: version \`3.4.2' is not installed (set by GOENV_VERSION environment variable)
goenv: version \`1.2' is not installed (set by GOENV_VERSION environment variable)
3.3.3 (set by GOENV_VERSION environment variable)
OUT
}

goenv-version-without-stderr() {
  goenv-version 2>/dev/null
}

@test "set by GOENV_VERSION, one missing (stderr filtered)" {
  create_version "3.3.3"
  GOENV_VERSION=3.4.2:3.3.3 run goenv-version-without-stderr
  assert_failure
  assert_output <<OUT
3.3.3 (set by GOENV_VERSION environment variable)
OUT
}
