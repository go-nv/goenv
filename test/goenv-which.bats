#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage which
  assert_success <<'OUT'
Usage: goenv which <command>
OUT
}

@test "has completion support" {
  run goenv-which --complete
  assert_success <<OUT
OUT
}

@test "fails and prints usage when no command argument is given" {
  run goenv-which
  assert_failure
  assert_output <<OUT
Usage: goenv which <command>
OUT
}

@test "prints path to executable when 'GOENV_VERSION' environment variable is specified and executable argument is found in 'GOENV_ROOT/versions/<version>/bin/<executable>'" {
  create_executable "1.10.3" "gofmt"
  create_executable "1.11.1" "go"

  GOENV_VERSION=1.10.3 run goenv-which gofmt
  assert_success
  assert_output "${GOENV_ROOT}/versions/1.10.3/bin/gofmt"

  GOENV_VERSION=1.11.1 run goenv-which go
  assert_success
  assert_output "${GOENV_ROOT}/versions/1.11.1/bin/go"
}

@test "prints path to executable of first version specified when multiple versions separated by ':' from 'GOENV_VERSION' environment variable executable argument is found in 'GOENV_ROOT/versions/<version>/bin/<executable>'" {
  create_executable "1.10.3" "gofmt"
  create_executable "1.11.1" "gofmt"

  GOENV_VERSION=1.11.1:1.10.3 run goenv-which gofmt
  assert_success
  assert_output "${GOENV_ROOT}/versions/1.11.1/bin/gofmt"

  GOENV_VERSION=1.10.3:1.11.1 run goenv-which gofmt
  assert_success
  assert_output "${GOENV_ROOT}/versions/1.10.3/bin/gofmt"
}

@test "fails when specified version by 'GOENV_VERSION' environment variable is not installed" {
  GOENV_VERSION=1.10.3 run goenv-which go
  assert_failure "goenv: version '1.10.3' is not installed (set by GOENV_VERSION environment variable)"
}

@test "fails when specified versions separated by ':' from 'GOENV_VERSION' environment variable are not installed" {
  GOENV_VERSION=1.10.3:1.11.1 run goenv-which go
  assert_failure
  assert_output <<OUT
goenv: version '1.10.3' is not installed (set by GOENV_VERSION environment variable)
goenv: version '1.11.1' is not installed (set by GOENV_VERSION environment variable)
OUT
}

@test "fails when specified executable argument is not found for installed version specified in 'GOENV_VERSION' environment variable" {
  create_executable "1.8.1" "go"
  GOENV_VERSION=1.8.1 run goenv-which gofmt
  assert_failure "goenv: 'gofmt' command not found"
}

@test "fails when no executable found for system version specified in 'GOENV_VERSION' environment variable" {
  export PATH="$(path_without "go")"
  GOENV_VERSION=system run goenv-which go
  assert_failure "goenv: 'go' command not found"
}

@test "fails when executable found in other versions but not in version specified in 'GOENV_VERSION' environment variable" {
  create_executable "1.4.0" "go"
  create_executable "1.10.3" "gofmt"
  create_executable "1.11.1" "gofmt"

  GOENV_VERSION=1.4.0 run goenv-which gofmt
  assert_failure
  assert_output <<OUT
goenv: 'gofmt' command not found

The 'gofmt' command exists in these Go versions:
  1.10.3
  1.11.1
OUT
}

@test "carries original IFS within hooks" {
  create_hook which hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
exit
SH

  IFS=$' \t\n' GOENV_VERSION=system run goenv-which anything
  assert_success
  assert_output "HELLO=:hello:ugly:world:again"
}

@test "prints executable found in version discovered from 'goenv-version-name' (GOENV_ROOT/version)" {
  mkdir -p "$GOENV_ROOT"
  echo "1.6.1" > "${GOENV_ROOT}/version"
  create_executable "1.6.1" "go"

  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"

  GOENV_VERSION= run goenv-which go
  assert_success "${GOENV_ROOT}/versions/1.6.1/bin/go"
}

@test "prints executable found in PATH for 'system' specified by 'GOENV_VERSION' environment variable" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"

  GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "prints executable found in PATH for 'system' specified by 'GOENV_VERSION' environment variable excludes 'GOENV_ROOT/shims'" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"
  create_executable "${GOENV_ROOT}/shims" "kill-all-humans"

  PATH="${GOENV_ROOT}/shims:$PATH" GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"

  PATH="$PATH:${GOENV_ROOT}/shims" GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"

  PATH="${GOENV_ROOT}/shims:${GOENV_ROOT}/shims:/tmp/non-existent:$PATH:${GOENV_ROOT}/shims" \
    GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "fails when executable is not found in PATH for 'system' specified by 'GOENV_VERSION' even though it's present in current dir" {
  export PATH="$(path_without "kill-all-humans")"
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  touch kill-all-humans
  chmod +x kill-all-humans

  GOENV_VERSION=system run goenv-which kill-all-humans
  assert_failure "goenv: 'kill-all-humans' command not found"
}

