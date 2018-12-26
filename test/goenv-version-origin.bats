#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version-origin
  assert_success <<'OUT'
Usage: goenv version-origin
OUT
}

@test "prints GOENV_ROOT/version file regardless if it exists when no other '.go-version' files exist and no 'GOENV_VERSION_ORIGIN' (from hooks) and no 'GOENV_VERSION' environment variables are provided" {
  assert [ ! -e "${GOENV_ROOT}/version" ]
  assert [ ! -d "${GOENV_ROOT}/versions" ]

  run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}

@test "prints GOENV_ROOT/version file if it exists and no 'GOENV_VERSION_ORIGIN' (from hooks) and no 'GOENV_VERSION' environment variables are provided" {
  mkdir -p "$GOENV_ROOT"
  touch "${GOENV_ROOT}/version"
  run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}

@test "prints 'GOENV_VERSION' environment variable if it's specified in favor of local '.go-version' file" {
  touch '.go-version'

  GOENV_VERSION=1 run goenv-version-origin
  assert_success "GOENV_VERSION environment variable"
}

@test "prints local file path if '.go-version' file exists and no 'GOENV_VERSION' environment variable is provided" {
  touch .go-version

  run goenv-version-origin
  assert_success "${PWD}/.go-version"
}

@test "prints 'GOENV_VERSION_ORIGIN' environment variable provided by hook in favor of 'GOENV_VERSION' environment variable specified and local '.go-version' file" {
  touch .go-version
  create_hook version-origin test.bash <<<"GOENV_VERSION_ORIGIN=plugin"

  GOENV_VERSION=1 run goenv-version-origin
  assert_success "plugin"
}

@test "carries original IFS within hooks" {
  create_hook version-origin hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  IFS=$' \t\n' run goenv-version-origin
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "prints default 'GOENV_ROOT/version' file and doesn't inherit specified 'GOENV_VERSION_ORIGIN' environment variable" {
  GOENV_VERSION_ORIGIN=ignored run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}
