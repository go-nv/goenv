#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage shims
  assert_success <<'OUT'
Usage: goenv shims [--short]
OUT
}

@test "has completion support" {
  run goenv-shims --complete
  assert_success <<OUT
--short
OUT
}

@test "prints empty output when no arguments are given and no shims are present in 'GOENV_ROOT/shims/*'" {
  run goenv-shims
  assert_success
  assert_output ''
}

@test "prints found shims paths in alphabetic order when no arguments are given and shims are present in 'GOENV_ROOT/shims/*'" {
  mkdir -p "${GOENV_ROOT}/shims"
  # NOTE: Order of creation is not alphabetic
  touch "${GOENV_ROOT}/shims/godoc"
  touch "${GOENV_ROOT}/shims/go"
  touch "${GOENV_ROOT}/shims/gofmt"

  run goenv-shims

  assert_output <<OUT
${GOENV_ROOT}/shims/go
${GOENV_ROOT}/shims/godoc
${GOENV_ROOT}/shims/gofmt
OUT

  assert_success
}

@test "prints found shims names only in alphabetic order when '--short' argument is given and shims are present in 'GOENV_ROOT/shims/*'" {
  mkdir -p "${GOENV_ROOT}/shims"
  # NOTE: Order of creation is not alphabetic
  touch "${GOENV_ROOT}/shims/godoc"
  touch "${GOENV_ROOT}/shims/go"
  touch "${GOENV_ROOT}/shims/gofmt"

  run goenv-shims --short

  assert_output <<OUT
go
godoc
gofmt
OUT

  assert_success
}
