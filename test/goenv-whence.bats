#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage whence
  assert_success
  assert_output <<'OUT'
Usage: goenv whence [--path] <command>
OUT
}

@test "has completion support" {
  run goenv-whence --complete
  assert_success <<OUT
--path
OUT
}

@test "fails and prints usage when no argument is given" {
  run goenv-whence

  assert_failure
  assert_output <<OUT
Usage: goenv whence [--path] <command>
OUT
}

@test "prints version when given executable argument is present in GOENV_ROOT/versions/<version>/bin/<binary>" {
  create_executable "1.6.0" "go"
  create_executable "1.6.1" "go"

  run goenv-whence go
  assert_success
  assert_output <<OUT
1.6.0
1.6.1
OUT
}

@test "prints version when '--path' argument is given and given executable argument is present in GOENV_ROOT/versions/<version>/bin/<binary>" {
  create_executable "1.6.0" "go"
  create_executable "1.6.1" "go"

  run goenv-whence --path go
  assert_success
  assert_output <<OUT
${GOENV_ROOT}/versions/1.6.0/bin/go
${GOENV_ROOT}/versions/1.6.1/bin/go
OUT
}

# NOTE: Notice that one of the binaries is not in `bin` folder
@test "does not print version when executable argument is present in GOENV_ROOT/versions/<version>/<binary>" {
  mkdir -p "${GOENV_ROOT}/versions/1.6.1/"
  touch "${GOENV_ROOT}/versions/1.6.1/notfound"

  create_executable "1.6.1" "go"

  run goenv-whence go
  assert_success
  assert_output <<OUT
1.6.1
OUT
}

@test "fails when given filename argument is present but not executable in GOENV_ROOT/versions/<version>/bin/<binary>" {
  create_executable "1.6.1" "go"

  chmod -x "${GOENV_ROOT}/versions/1.6.1/bin/go"

  run goenv-whence go
  assert_failure
  assert_output <<OUT
OUT
}

@test "fails when given filename argument is present but not executable in GOENV_ROOT/versions/<version>/bin/<binary>" {
  create_executable "1.6.1" "go"

  chmod -x "${GOENV_ROOT}/versions/1.6.1/bin/go"

  run goenv-whence go
  assert_failure
  assert_output <<OUT
OUT
}

