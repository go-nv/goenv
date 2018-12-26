#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

stub_system_go() {
  local stub="${GOENV_TEST_DIR}/bin/go"
  mkdir -p "$(dirname "$stub")"
  touch "$stub" && chmod +x "$stub"
}

@test "no versions installed" {
  stub_system_go
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-versions
  assert_success "* system (set by ${GOENV_ROOT}/version)"
}

@test "not even system go available" {
  PATH="$(path_without go)" run goenv-versions
  assert_failure
  assert_output "Warning: no Go detected on the system"
}

@test "bare output no versions installed" {
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-versions --bare
  assert_success ""
}

@test "single version installed" {
  stub_system_go
  create_version "3.3"
  run goenv-versions
  assert_success
  assert_output <<OUT
* system (set by ${GOENV_ROOT}/version)
  3.3
OUT
}

@test "single version bare" {
  create_version "3.3"
  run goenv-versions --bare
  assert_success "3.3"
}

@test "multiple versions" {
  stub_system_go
  create_version "2.7.6"
  create_version "3.3.3"
  create_version "3.4.0"
  run goenv-versions
  assert_success
  assert_output <<OUT
* system (set by ${GOENV_ROOT}/version)
  2.7.6
  3.3.3
  3.4.0
OUT
}

@test "indicates current version" {
  stub_system_go
  create_version "3.3.3"
  create_version "3.4.0"
  GOENV_VERSION=3.3.3 run goenv-versions
  assert_success
  assert_output <<OUT
  system
* 3.3.3 (set by GOENV_VERSION environment variable)
  3.4.0
OUT
}

@test "bare doesn't indicate current version" {
  create_version "3.3.3"
  create_version "3.4.0"
  GOENV_VERSION=3.3.3 run goenv-versions --bare
  assert_success
  assert_output <<OUT
3.3.3
3.4.0
OUT
}

@test "globally selected version" {
  stub_system_go
  create_version "3.3.3"
  create_version "3.4.0"
  cat > "${GOENV_ROOT}/version" <<<"3.3.3"
  run goenv-versions
  assert_success
  assert_output <<OUT
  system
* 3.3.3 (set by ${GOENV_ROOT}/version)
  3.4.0
OUT
}

@test "per-project version" {
  stub_system_go
  create_version "3.3.3"
  create_version "3.4.0"
  cat > ".go-version" <<<"3.3.3"
  run goenv-versions
  assert_success
  assert_output <<OUT
  system
* 3.3.3 (set by ${GOENV_TEST_DIR}/.go-version)
  3.4.0
OUT
}

@test "ignores non-directories under versions" {
  create_version "3.3"
  touch "${GOENV_ROOT}/versions/hello"

  run goenv-versions --bare
  assert_success "3.3"
}

@test "lists symlinks under versions" {
  create_version "2.7.8"
  ln -s "2.7.8" "${GOENV_ROOT}/versions/2.7"

  run goenv-versions --bare
  assert_success
  assert_output <<OUT
2.7
2.7.8
OUT
}

@test "doesn't list symlink aliases when --skip-aliases" {
  create_version "1.8.7"
  ln -s "1.8.7" "${GOENV_ROOT}/versions/1.8"
  mkdir moo
  ln -s "${PWD}/moo" "${GOENV_ROOT}/versions/1.9"

  run goenv-versions --bare --skip-aliases
  assert_success

  assert_output <<OUT
1.8.7
1.9
OUT
}
