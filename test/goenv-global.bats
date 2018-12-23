#!/usr/bin/env bats

load test_helper

@test "default" {
  run goenv-global
  assert_success
  assert_output "system"
}

@test "read GOENV_ROOT/version" {
  mkdir -p "$GOENV_ROOT"
  echo "1.2.3" > "$GOENV_ROOT/version"
  run goenv-global
  assert_success
  assert_output "1.2.3"
}

@test "set GOENV_ROOT/version" {
  mkdir -p "$GOENV_ROOT/versions/1.2.3"
  run goenv-global "1.2.3"
  assert_success
  run goenv-global
  assert_success "1.2.3"
}

@test "fail setting invalid GOENV_ROOT/version" {
  mkdir -p "$GOENV_ROOT"
  run goenv-global "1.2.3"
  assert_failure "goenv: version \`1.2.3' not installed"
}
