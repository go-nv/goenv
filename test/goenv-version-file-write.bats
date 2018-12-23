#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "invocation without 2 arguments prints usage" {
  run goenv-version-file-write
  assert_failure "Usage: goenv version-file-write <file> <version>"
  run goenv-version-file-write "one" ""
  assert_failure
}

@test "setting nonexistent version fails" {
  assert [ ! -e ".go-version" ]
  run goenv-version-file-write ".go-version" "2.7.6"
  assert_failure "goenv: version \`2.7.6' not installed"
  assert [ ! -e ".go-version" ]
}

@test "writes value to arbitrary file" {
  mkdir -p "${GOENV_ROOT}/versions/2.7.6"
  assert [ ! -e "my-version" ]
  run goenv-version-file-write "${PWD}/my-version" "2.7.6"
  assert_success ""
  assert [ "$(cat my-version)" = "2.7.6" ]
}
