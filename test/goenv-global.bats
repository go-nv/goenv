#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help global
  assert_success <<OUT
Usage: goenv global <version>
OUT
}

@test "defaults to 'system'" {
  run goenv-global
  assert_success
  assert_output "system"
}

@test "returns contents of GOENV_ROOT/version" {
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/version"
  run goenv-global
  assert_success
  assert_output "1.2.3"
}

@test "returns contents of GOENV_ROOT/global" {
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/global"
  run goenv-global
  assert_success
  assert_output "1.2.3"
}

@test "returns contents of GOENV_ROOT/default" {
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/default"
  run goenv-global
  assert_success
  assert_output "1.2.3"
}

@test "writes specified version to GOENV_ROOT/version if version is installed" {
  mkdir -p "$GOENV_ROOT/versions/1.2.3"
  run goenv-global "1.2.3"
  assert_success
  run goenv-global
  assert_success "1.2.3"
}

@test "fails writing specified version to GOENV_ROOT/version if version is not installed" {
  run goenv-global "1.2.3"
  assert_failure

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
