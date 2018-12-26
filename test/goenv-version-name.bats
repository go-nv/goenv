#!/usr/bin/env bats

load test_helper

create_version() {
  mkdir -p "${GOENV_ROOT}/versions/$1"
}

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version-name
  assert_success <<'OUT'
Usage: goenv version-name
OUT
}

@test "prints 'system' when no version is present in GOENV_ROOT/versions" {
  assert [ ! -d "${GOENV_ROOT}/versions" ]

  run goenv-version-name
  assert_success "system"
}

@test "prints 'system' when 'GOENV_VERSION' environment variable is 'system' and does not check for existence" {
  GOENV_VERSION=system run goenv-version-name
  assert_success "system"
}

@test "hooks can override version of specified 'GOENV_VERSION' environment variable if versions exist" {
  create_version "1.11.1"
  create_version "1.10.3"
  create_hook version-name test.bash <<< "GOENV_VERSION=1.10.3"

  GOENV_VERSION=1.11.1 run goenv-version-name
  assert_success "1.10.3"
}

@test "carries original IFS within hooks when version exists or is 'system'" {
  create_hook version-name hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=system
  IFS=$' \t\n' run goenv-version-name env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "carries original IFS within hooks when version does not exist or is not 'system'" {
  create_hook version-name hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=1.11.1
  IFS=$' \t\n' run goenv-version-name env
  assert_failure
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "prints 'GOENV_VERSION' environment variable which has precedence over local '.go-version' file when versions exist" {
  create_version "1.10.3"
  create_version "1.11.1"

  echo "1.10.3" > '.go-version'
  run goenv-version-name

  # NOTE: Verify it's chosen when there's only a local '.go-version' file
  assert_success "1.10.3"

  GOENV_VERSION=1.11.1 run goenv-version-name
  assert_success "1.11.1"
}

@test "local '.go-version' file has precedence over 'GOENV_ROOT/version' file when versions exist" {
  create_version "1.10.3"
  create_version "1.11.1"

  echo "1.10.3" > "${GOENV_ROOT}/version"

  # NOTE: Verify it's chosen when there's only a global '.go-version' file
  run goenv-version-name
  assert_success "1.10.3"

  echo "1.11.1" > "${GOENV_ROOT}/version"

  run goenv-version-name
  assert_success "1.11.1"
}

@test "fails when version specified by 'GOENV_VERSION' environment variable is not existent" {
  GOENV_VERSION=1.2 run goenv-version-name
  assert_failure "goenv: version '1.2' is not installed (set by GOENV_VERSION environment variable)"
}

@test "fails when one of the versions separated by ':' and specified by 'GOENV_VERSION' environment variable is not existing" {
  create_version "1.11.1"

  # NOTE: Test with last version specified is missing
  GOENV_VERSION="1.11.1:1.10.3" run goenv-version-name

  assert_failure
  assert_output <<OUT
goenv: version '1.10.3' is not installed (set by GOENV_VERSION environment variable)
1.11.1
OUT

  # NOTE: Test with first version specified is missing
  GOENV_VERSION="1.10.3:1.11.1" run goenv-version-name

  assert_failure
  assert_output <<OUT
goenv: version '1.10.3' is not installed (set by GOENV_VERSION environment variable)
1.11.1
OUT
}

@test "prints version only while removing 'go' prefix in name when version exists" {
  create_version "1.10.3"
  echo "1.10.3" > '.go-version'
  run goenv-version-name

  assert_success
  assert_output "1.10.3"
}
