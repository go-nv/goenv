#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version-file
  assert_success <<'OUT'
Usage: goenv version-file [<dir>]
OUT
}

@test "detects global 'version' file when no arguments are given and file exists at GOENV_ROOT/version" {
  create_file "${GOENV_ROOT}/version"

  run goenv-version-file
  assert_success "${GOENV_ROOT}/version"
}

@test "prints global file when no arguments are given and if no '.go-version' files exist neither in global nor in current dir" {
  assert [ ! -e "${GOENV_ROOT}/version" ]
  assert [ ! -e ".go-version" ]

  run goenv-version-file

  assert_success "${GOENV_ROOT}/version"
}

@test "prints current file when no arguments are given and if '.go-version' file exists in current directory" {
  create_file ".go-version"

  run goenv-version-file

  assert_success "${GOENV_TEST_DIR}/.go-version"
}

@test "prints parent file when no arguments are given and if '.go-version' file exists in parent directory" {
  create_file ".go-version"
  mkdir -p project
  cd project

  run goenv-version-file

  assert_success "${GOENV_TEST_DIR}/.go-version"
}

@test "topmost '.go-version' file has precedence when no arguments are given and if parent and current dir files exist" {
  create_file ".go-version"

  create_file "project/.go-version"
  cd project

  run goenv-version-file

  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "GOENV_DIR has precedence over PWD when no arguments are given and if in both dirs '.go-version' files exist" {
  create_file "widget/.go-version"
  create_file "project/.go-version"
  # NOTE: Make sure we're currently in a dir where there's a local '.go-version' file
  cd project

  GOENV_DIR="${GOENV_TEST_DIR}/widget" run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/widget/.go-version"
}

@test "PWD is searched when no arguments are given and if GOENV_DIR yields no results if in PWD there's no '.go-version' file, but in GOENV_DIR there is" {
  mkdir -p "widget/blank"
  create_file "project/.go-version"

  # NOTE: Make sure we're currently in a dir where there's a local '.go-version' file
  cd project

  GOENV_DIR="${GOENV_TEST_DIR}/widget/blank" run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "prints version file in target directory when a target directory is specified and there's a '.go-version' file there" {
  create_file "project/.go-version"

  run goenv-version-file "${PWD}/project"
  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "fails when a target directory is specified and there's no '.go-version' file there, but in child dir there is a file there" {
  mkdir -p "widget/blank"
  create_file "widget/blank/.go-version"

  # NOTE: Make sure we're currently in a dir where there's no local '.go-version' file
  cd widget

  run goenv-version-file "$PWD"
  assert_failure ""
}

@test "fails when a target directory is specified and there's no '.go-version' file there, but a GOENV_DIR is given and there is a file there" {
  mkdir -p "widget/blank"
  create_file "project/.go-version"

  # NOTE: Make sure we're currently in a dir where there's no local '.go-version' file
  cd widget/blank

  GOENV_DIR="${GOENV_TEST_DIR}/project" run goenv-version-file "$PWD"

  assert_failure ""
}

@test "fails when a target directory is specified and there's no '.go-version' file there" {
  run goenv-version-file "$PWD"
  assert_failure ""
}
