#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage root
  assert_success <<'OUT'
Usage: goenv root
OUT
}

@test "returns current GOENV_ROOT" {
  GOENV_ROOT=/tmp/whatiexpect run goenv-root

  assert_success
  assert_output '/tmp/whatiexpect'
}

