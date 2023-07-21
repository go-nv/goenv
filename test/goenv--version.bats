#!/usr/bin/env bats

load test_helper

base_dir=$(echo $(dirname -- "$0") | sed -E 's/goenv(\/[0-9]+\.[0-9]+\.[0-9]+|\/goenv)?.+/goenv\/\1/i')
base_dir=$(echo $base_dir | sed -E 's/\/$//')

expected_version="goenv $(cat $base_dir/APP_VERSION)"

@test "default version is 'version' variable" {
  assert [ ! -e "$GOENV_ROOT" ]
  run goenv---version
  assert_success "${expected_version}"
}
