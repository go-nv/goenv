#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version-file-write
  assert_success <<'OUT'
Usage: goenv version-file-write <file> <version>
OUT
}

@test "prints usage instructions when 2 arguments aren't specified" {
  run goenv-version-file-write

  assert_failure "Usage: goenv version-file-write <file> <version>"

  run goenv-version-file-write "one"
  assert_failure "Usage: goenv version-file-write <file> <version>"
}

@test "fails when 2 arguments are specified, but version is non-existent" {
  assert [ ! -e ".go-version" ]

  run goenv-version-file-write ".go-version" "1.11.1"
  assert_failure "goenv: version '1.11.1' not installed"

  assert [ ! -e ".go-version" ]
}

@test "writes version to file when 2 arguments are specified and version is existent" {
  mkdir -p "${GOENV_ROOT}/versions/1.11.1"
  assert [ ! -e "my-version" ]

  run goenv-version-file-write "${PWD}/my-version" "1.11.1"

  assert_success ""
  assert [ "$(cat my-version)" = "1.11.1" ]
}
