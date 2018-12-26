#!/usr/bin/env bats

load test_helper

@test "prefixes are collected for versions separated by ':' from 'GOENV_VERSION' environment variable" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"

  mkdir -p "${GOENV_ROOT}/versions/1.11.1"

  GOENV_VERSION="system:1.11.1" run goenv-prefix
  assert_success "${GOENV_TEST_DIR}:${GOENV_ROOT}/versions/1.11.1"

  GOENV_VERSION="1.11.1:system" run goenv-prefix
  assert_success "${GOENV_ROOT}/versions/1.11.1:${GOENV_TEST_DIR}"
}

@test "uses dirname of file argument as 'GOENV_DIR' environment variable" {
  mkdir -p "${GOENV_TEST_DIR}/dir1"
  touch "${GOENV_TEST_DIR}/dir1/file.go"

  GOENV_FILE_ARG="${GOENV_TEST_DIR}/dir1/file.go" run goenv echo GOENV_DIR
  assert_output "${GOENV_TEST_DIR}/dir1"
}

@test "follows symlink of file argument as 'GOENV_DIR' environment variable" {
  mkdir -p "${GOENV_TEST_DIR}/dir1"
  mkdir -p "${GOENV_TEST_DIR}/dir2"
  touch "${GOENV_TEST_DIR}/dir1/file.go"
  ln -s "${GOENV_TEST_DIR}/dir1/file.go" "${GOENV_TEST_DIR}/dir2/symlink.go"

  GOENV_FILE_ARG="${GOENV_TEST_DIR}/dir2/symlink.go" run goenv echo GOENV_DIR
  assert_output "${GOENV_TEST_DIR}/dir1"
}
