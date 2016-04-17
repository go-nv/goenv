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
  run goenv-version-name
  assert_success "system"
}

@test "system version is not checked for existance" {
  GOENV_VERSION=system run goenv-version-name
  assert_success "system"
}

@test "GOENV_VERSION can be overridden by hook" {
  create_version "2.7.11"
  create_version "3.5.1"
  create_hook version-name test.bash <<<"GOENV_VERSION=3.5.1"

  GOENV_VERSION=2.7.11 run goenv-version-name
  assert_success "3.5.1"
}

@test "carries original IFS within hooks" {
  create_hook version-name hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=system
  IFS=$' \t\n' run goenv-version-name env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "GOENV_VERSION has precedence over local" {
  create_version "2.7.11"
  create_version "3.5.1"

  cat > ".go-version" <<<"2.7.11"
  run goenv-version-name
  assert_success "2.7.11"

  GOENV_VERSION=3.5.1 run goenv-version-name
  assert_success "3.5.1"
}

@test "local file has precedence over global" {
  create_version "2.7.11"
  create_version "3.5.1"

  cat > "${GOENV_ROOT}/version" <<<"2.7.11"
  run goenv-version-name
  assert_success "2.7.11"

  cat > ".go-version" <<<"3.5.1"
  run goenv-version-name
  assert_success "3.5.1"
}

@test "missing version" {
  GOENV_VERSION=1.2 run goenv-version-name
  assert_failure "goenv: version \`1.2' is not installed (set by GOENV_VERSION environment variable)"
}

@test "one missing version (second missing)" {
  create_version "3.5.1"
  GOENV_VERSION="3.5.1:1.2" run goenv-version-name
  assert_failure
  assert_output <<OUT
goenv: version \`1.2' is not installed (set by GOENV_VERSION environment variable)
3.5.1
OUT
}

@test "one missing version (first missing)" {
  create_version "3.5.1"
  GOENV_VERSION="1.2:3.5.1" run goenv-version-name
  assert_failure
  assert_output <<OUT
goenv: version \`1.2' is not installed (set by GOENV_VERSION environment variable)
3.5.1
OUT
}

goenv-version-name-without-stderr() {
  goenv-version-name 2>/dev/null
}

@test "one missing version (without stderr)" {
  create_version "3.5.1"
  GOENV_VERSION="1.2:3.5.1" run goenv-version-name-without-stderr
  assert_failure
  assert_output <<OUT
3.5.1
OUT
}

@test "version with prefix in name" {
  create_version "2.7.11"
  cat > ".go-version" <<<"go-2.7.11"
  run goenv-version-name
  assert_success
  assert_output "2.7.11"
}
