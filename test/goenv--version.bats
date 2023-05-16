#!/usr/bin/env bats

load test_helper

expected_version="goenv 2.0.7"

@test "default version is 'version' variable" {
  assert [ ! -e "$GOENV_ROOT" ]
  run goenv---version
  assert_success "${expected_version}"
}
