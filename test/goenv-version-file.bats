#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

create_file() {
  mkdir -p "$(dirname "$1")"
  touch "$1"
}

@test "detects global 'version' file" {
  create_file "${GOENV_ROOT}/version"
  run goenv-version-file
  assert_success "${GOENV_ROOT}/version"
}

@test "prints global file if no version files exist" {
  assert [ ! -e "${GOENV_ROOT}/version" ]
  assert [ ! -e ".go-version" ]
  run goenv-version-file
  assert_success "${GOENV_ROOT}/version"
}

@test "in current directory" {
  create_file ".go-version"
  run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/.go-version"
}

@test "in parent directory" {
  create_file ".go-version"
  mkdir -p project
  cd project
  run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/.go-version"
}

@test "topmost file has precedence" {
  create_file ".go-version"
  create_file "project/.go-version"
  cd project
  run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "GOENV_DIR has precedence over PWD" {
  create_file "widget/.go-version"
  create_file "project/.go-version"
  cd project
  GOENV_DIR="${GOENV_TEST_DIR}/widget" run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/widget/.go-version"
}

@test "PWD is searched if GOENV_DIR yields no results" {
  mkdir -p "widget/blank"
  create_file "project/.go-version"
  cd project
  GOENV_DIR="${GOENV_TEST_DIR}/widget/blank" run goenv-version-file
  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "finds version file in target directory" {
  create_file "project/.go-version"
  run goenv-version-file "${PWD}/project"
  assert_success "${GOENV_TEST_DIR}/project/.go-version"
}

@test "fails when no version file in target directory" {
  run goenv-version-file "$PWD"
  assert_failure ""
}
