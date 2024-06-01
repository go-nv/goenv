#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "${GOENV_TEST_DIR}/myproject"
  cd "${GOENV_TEST_DIR}/myproject"
}

@test "has usage instructions" {
  run goenv-help --usage version-file-read
  assert_success <<OUT
Usage: goenv version-file-read <file>
OUT
}

@test "fails without arguments" {
  run goenv-version-file-read
  assert_failure ""
}

@test "fails for file specified in arguments that does not exist" {
  assert [ ! -f ${GOENV_TEST_DIR}/non-existent ]

  run goenv-version-file-read "non-existent"
  assert_failure ""
}

@test "fails for file specified in arguments that exists but is blank" {
  echo >my-version

  run goenv-version-file-read my-version
  assert_failure ""
}

@test "reads go.mod file specified in arguments that exists and is not blank" {
  cat >go.mod <<IN

module github.com/go-nv/goenv

go 1.11

require (
	github.com/foo/bar v0.0.0-20220101000000-0123456789abcdef // indirect
)

IN

  run goenv-version-file-read go.mod
  assert_success "1.11"
}

@test "reads go.mod file with toolchain specified in arguments that exists and is not blank" {
  cat >go.mod <<IN

module github.com/go-nv/goenv

go 1.11

toolchain go1.11.4

require (
	github.com/foo/bar v0.0.0-20220101000000-0123456789abcdef // indirect
)

IN

  run goenv-version-file-read go.mod
  assert_success "1.11.4"
}

@test "reads version file specified in arguments that exists and is not blank" {
  echo "1.11.1" >my-version

  run goenv-version-file-read my-version
  assert_success "1.11.1"
}

@test "reads version file without leading and trailing spaces, specified in arguments that exists and is not blank" {
  echo "         1.11.1   " >my-version

  run goenv-version-file-read my-version
  assert_success "1.11.1"
}

@test "reads version file without additional newlines, specified in arguments that exists and is not blank" {
  cat >my-version <<IN

1.11.1



IN

  run goenv-version-file-read my-version
  assert_success "1.11.1"
}

@test "reads version file that's not ending with newline, specified in arguments that exists and is not blank" {
  echo -n "1.11.1" >my-version

  run goenv-version-file-read my-version
  assert_success "1.11.1"
}

@test "reads version file that ends with carriage return, specified in arguments that exists and is not blank" {
  echo $'1.11.1\r' >my-version

  run goenv-version-file-read my-version
  assert_success "1.11.1"
}
