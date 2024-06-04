#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage global
  assert_success_out <<OUT
Usage: goenv global [<version>]
OUT
}

@test "global has completion support" {
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  run goenv-global --complete
  assert_success_out <<OUT
latest
system
1.10.9
1.9.10
OUT
}

@test "defaults to 'system'" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.1"
  run goenv-global
  assert_success "system"
}

@test "returns contents of GOENV_ROOT/version" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.1"
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/version"
  run goenv-global
  assert_success "1.2.3"
}

@test "returns contents of GOENV_ROOT/global" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.1"
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/global"
  run goenv-global
  assert_success "1.2.3"
}

@test "returns contents of GOENV_ROOT/default" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.1"
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/default"
  run goenv-global
  assert_success "1.2.3"
}

@test "writes specified version to GOENV_ROOT/version if version is installed" {
  mkdir -p "$GOENV_ROOT/versions/1.2.3"
  run goenv-global 1.2.3
  assert_success ""
  run goenv-global
  assert_success "1.2.3"
}

@test "sets properly sorted latest global version when 'latest' version is given and any version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.9.9"
  run goenv-global latest
  assert_success ""
  run goenv-global
  assert_success "1.10.10"
}

@test "sets latest global version when major version is given and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-global 1
  assert_success ""
  run goenv-global
  assert_success "1.2.10"
}

@test "fails setting latest global version when major or minor single number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.10"
  run goenv-global 9
  assert_failure "goenv: version '9' not installed"
}

@test "sets latest global version when minor version is given as single number and any matching major.minor version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/4.5.2"
  run goenv-global 2
  assert_success ""
  run goenv-global
  assert_success "1.2.10"
}

@test "sets latest global version when minor version is given as major.minor number and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.2.2"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/2.1.2"
  run goenv-global 1.2
  assert_success ""
  run goenv-global
  assert_success "1.2.10"
}

@test "fails setting latest global version when major.minor number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.9"
  run goenv-global 1.9
  assert_failure "goenv: version '1.9' not installed"
}

@test "fails writing specified version to GOENV_ROOT/version if version is not installed" {
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-global system
  assert_failure "goenv: system version not found in PATH"

  run goenv-global 1.2.3
  assert_failure "goenv: version '1.2.3' not installed"

  run goenv-installed 1.2.3
  assert_failure "goenv: version '1.2.3' not installed"

  run goenv-global
  assert_success "system"
}

@test "reads version from GOENV_ROOT/{version,global,default} in the order they're specified" {
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/version"
  echo "1.2.4" > "$GOENV_ROOT/global"
  echo "1.2.5" > "$GOENV_ROOT/default"

  run goenv-global
  assert_success "1.2.3"

  rm "$GOENV_ROOT/version"
  run goenv-global
  assert_success "1.2.4"

  rm "$GOENV_ROOT/global"
  run goenv-global
  assert_success "1.2.5"
}
