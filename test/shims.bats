#!/usr/bin/env bats

load test_helper

@test "no shims" {
  run goenv-shims
  assert_success
  assert [ -z "$output" ]
}

@test "shims" {
  mkdir -p "${GOENV_ROOT}/shims"
  touch "${GOENV_ROOT}/shims/python"
  touch "${GOENV_ROOT}/shims/irb"
  run goenv-shims
  assert_success
  assert_line "${GOENV_ROOT}/shims/python"
  assert_line "${GOENV_ROOT}/shims/irb"
}

@test "shims --short" {
  mkdir -p "${GOENV_ROOT}/shims"
  touch "${GOENV_ROOT}/shims/python"
  touch "${GOENV_ROOT}/shims/irb"
  run goenv-shims --short
  assert_success
  assert_line "irb"
  assert_line "python"
}
