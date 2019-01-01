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

@test "has usage instructions" {
  run goenv-help --usage versions
  assert_success
  assert_output <<'OUT'
Usage: goenv versions [--bare] [--skip-aliases]
OUT
}

@test "has completion support" {
  run goenv-versions --complete
  assert_success
  assert_output <<OUT
--bare
--skip-aliases
OUT
}

@test "prints usage instructions when unknown arguments are given" {
  run goenv-versions magic and more
  assert_failure
  assert_output <<'OUT'
Usage: goenv versions [--bare] [--skip-aliases]
OUT
}

@test "prints 'system' when no versions are installed other than 'go' executable in 'PATH'" {
  stub_system_go
  assert [ ! -d "${GOENV_ROOT}/versions" ]

  run goenv-versions
  assert_success "* system (set by ${GOENV_ROOT}/version)"
}

@test "fails with warning when no versions are installed and no 'go' executable in 'PATH'" {
  PATH="$(path_without go)" run goenv-versions

  assert_failure
  assert_output "Warning: no Go detected on the system"
}

@test "prints empty output when '--bare' argument is given and no versions installed and no 'go' executable in 'PATH'" {
  assert [ ! -d "${GOENV_ROOT}/versions" ]

  run goenv-versions --bare
  assert_success ""
}

@test "prints empty output when '--bare' argument is given and no versions installed and 'go' executable in 'PATH'" {
  stub_system_go
  assert [ ! -d "${GOENV_ROOT}/versions" ]

  run goenv-versions --bare
  assert_success ""
}

@test "prints all available versions and showing 'system' as currently selected one when version is installed and 'go' executable in 'PATH'" {
  stub_system_go
  create_version "1.10.3"
  create_version "1.10.2"
  create_version "1.10.1"
  run goenv-versions

  assert_success
  assert_output <<OUT
* system (set by ${GOENV_ROOT}/version)
  1.10.1
  1.10.2
  1.10.3
OUT
}

@test "prints single version when no 'go' executable in 'PATH' and version is installed" {
  create_version "1.10.3"
  run goenv-versions

  assert_success
  assert_output <<OUT
  1.10.3
OUT
}

@test "prints single version when '--bare' argument is specified and version is installed" {
  create_version "1.10.3"
  run goenv-versions --bare

  assert_success "1.10.3"
}

@test "prints specified version by 'GOENV_VERSION' environment variable as selected one and all versions when all versions are available and 'go' executable in PATH" {
  stub_system_go
  create_version "1.10.1"
  create_version "1.11.1"

  GOENV_VERSION=1.10.1 run goenv-versions
  assert_success
  assert_output <<OUT
  system
* 1.10.1 (set by GOENV_VERSION environment variable)
  1.11.1
OUT
}

@test "prints all available versions and does not specify version by 'GOENV_VERSION' environment variable when '--bare' argument is specified and all versions are available" {
  create_version "1.10.3"
  create_version "1.11.1"
  GOENV_VERSION=1.10.3 run goenv-versions --bare
  assert_success
  assert_output <<OUT
1.10.3
1.11.1
OUT
}

@test "prints globally selected version by 'GOENV_ROOT/version' and all versions when all versions are installed and 'go' executable in PATH" {
  stub_system_go
  create_version "1.10.3"
  create_version "1.11.1"

  echo "1.11.1" > "${GOENV_ROOT}/version"
  run goenv-versions

  assert_success
  assert_output <<OUT
  system
  1.10.3
* 1.11.1 (set by ${GOENV_ROOT}/version)
OUT
}

@test "prints local '.go-version' file as selected one and all available versions when all versions are installed and 'go' executable in PATH" {
  stub_system_go
  create_version "1.6.1"
  create_version "1.8.4"

  echo "1.6.1" > '.go-version'

  run goenv-versions
  assert_success
  assert_output <<OUT
  system
* 1.6.1 (set by ${GOENV_TEST_DIR}/.go-version)
  1.8.4
OUT
}

@test "prints all available versions while ignoring non-directories in GOENV_ROOT/versions when all versions are installed" {
  create_version "1.8.4"
  touch "${GOENV_ROOT}/versions/hello"

  run goenv-versions
  assert_success <<OUT
  1.8.4
OUT
}

@test "prints symlinks in GOENV_ROOT/versions when all versions are installed" {
  create_version "1.8.3"
  ln -s "1.8.3" "${GOENV_ROOT}/versions/1.8.4"

  run goenv-versions
  assert_success
  assert_output <<OUT
  1.8.3
  1.8.4
OUT
}

@test "prints no symlinks in GOENV_ROOT/versions when '--skip-aliases' argument is specified and all versions are installed" {
  create_version "1.8.3"
  ln -s "1.8.3" "${GOENV_ROOT}/versions/1.8.4"

  run goenv-versions --skip-aliases
  assert_success
  assert_output <<OUT
  1.8.3
OUT
}
