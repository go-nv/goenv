#!/usr/bin/env bats

load test_helper

create_executable() {
  local bin
  if [[ $1 == */* ]]; then bin="$1"
  else bin="${GOENV_ROOT}/versions/${1}/bin"
  fi
  mkdir -p "$bin"
  touch "${bin}/$2"
  chmod +x "${bin}/$2"
}

@test "outputs path to executable" {
  create_executable "2.7" "python"
  create_executable "3.4" "py.test"

  GOENV_VERSION=2.7 run goenv-which python
  assert_success "${GOENV_ROOT}/versions/2.7/bin/python"

  GOENV_VERSION=3.4 run goenv-which py.test
  assert_success "${GOENV_ROOT}/versions/3.4/bin/py.test"

  GOENV_VERSION=3.4:2.7 run goenv-which py.test
  assert_success "${GOENV_ROOT}/versions/3.4/bin/py.test"
}

@test "searches PATH for system version" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"
  create_executable "${GOENV_ROOT}/shims" "kill-all-humans"

  GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "searches PATH for system version (shims prepended)" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"
  create_executable "${GOENV_ROOT}/shims" "kill-all-humans"

  PATH="${GOENV_ROOT}/shims:$PATH" GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "searches PATH for system version (shims appended)" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"
  create_executable "${GOENV_ROOT}/shims" "kill-all-humans"

  PATH="$PATH:${GOENV_ROOT}/shims" GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "searches PATH for system version (shims spread)" {
  create_executable "${GOENV_TEST_DIR}/bin" "kill-all-humans"
  create_executable "${GOENV_ROOT}/shims" "kill-all-humans"

  PATH="${GOENV_ROOT}/shims:${GOENV_ROOT}/shims:/tmp/non-existent:$PATH:${GOENV_ROOT}/shims" \
    GOENV_VERSION=system run goenv-which kill-all-humans
  assert_success "${GOENV_TEST_DIR}/bin/kill-all-humans"
}

@test "doesn't include current directory in PATH search" {
  export PATH="$(path_without "kill-all-humans")"
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  touch kill-all-humans
  chmod +x kill-all-humans
  GOENV_VERSION=system run goenv-which kill-all-humans
  assert_failure "goenv: kill-all-humans: command not found"
}

@test "version not installed" {
  create_executable "3.4" "py.test"
  GOENV_VERSION=3.3 run goenv-which py.test
  assert_failure "goenv: version \`3.3' is not installed (set by GOENV_VERSION environment variable)"
}

@test "versions not installed" {
  create_executable "3.4" "py.test"
  GOENV_VERSION=2.7:3.3 run goenv-which py.test
  assert_failure <<OUT
goenv: version \`2.7' is not installed (set by GOENV_VERSION environment variable)
goenv: version \`3.3' is not installed (set by GOENV_VERSION environment variable)
OUT
}

@test "no executable found" {
  create_executable "2.7" "py.test"
  GOENV_VERSION=2.7 run goenv-which fab
  assert_failure "goenv: fab: command not found"
}

@test "no executable found for system version" {
  export PATH="$(path_without "py.test")"
  GOENV_VERSION=system run goenv-which py.test
  assert_failure "goenv: py.test: command not found"
}

@test "executable found in other versions" {
  create_executable "2.7" "python"
  create_executable "3.3" "py.test"
  create_executable "3.4" "py.test"

  GOENV_VERSION=2.7 run goenv-which py.test
  assert_failure
  assert_output <<OUT
goenv: py.test: command not found

The \`py.test' command exists in these Go versions:
  3.3
  3.4
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

@test "discovers version from goenv-version-name" {
  mkdir -p "$GOENV_ROOT"
  cat > "${GOENV_ROOT}/version" <<<"1.6.1"
  create_executable "1.6.1" "go"

  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"

  GOENV_VERSION= run goenv-which go
  assert_success "${GOENV_ROOT}/versions/1.6.1/bin/go"
}
