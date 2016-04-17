#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "reports global file even if it doesn't exist" {
  assert [ ! -e "${GOENV_ROOT}/version" ]
  run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}

@test "detects global file" {
  mkdir -p "$GOENV_ROOT"
  touch "${GOENV_ROOT}/version"
  run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}

@test "detects GOENV_VERSION" {
  GOENV_VERSION=1 run goenv-version-origin
  assert_success "GOENV_VERSION environment variable"
}

@test "detects local file" {
  touch .go-version
  run goenv-version-origin
  assert_success "${PWD}/.go-version"
}

@test "reports from hook" {
  create_hook version-origin test.bash <<<"GOENV_VERSION_ORIGIN=plugin"

  GOENV_VERSION=1 run goenv-version-origin
  assert_success "plugin"
}

@test "carries original IFS within hooks" {
  create_hook version-origin hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=system
  IFS=$' \t\n' run goenv-version-origin env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "doesn't inherit GOENV_VERSION_ORIGIN from environment" {
  GOENV_VERSION_ORIGIN=ignored run goenv-version-origin
  assert_success "${GOENV_ROOT}/version"
}
