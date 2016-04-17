#!/usr/bin/env bats

load test_helper

@test "prefixes" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"
  mkdir -p "${GOENV_ROOT}/versions/2.7.10"
  GOENV_VERSION="system:2.7.10" run goenv-prefix
  assert_success "${GOENV_TEST_DIR}:${GOENV_ROOT}/versions/2.7.10"
  GOENV_VERSION="2.7.10:system" run goenv-prefix
  assert_success "${GOENV_ROOT}/versions/2.7.10:${GOENV_TEST_DIR}"
}

@test "should use dirname of file argument as GOENV_DIR" {
  mkdir -p "${GOENV_TEST_DIR}/dir1"
  touch "${GOENV_TEST_DIR}/dir1/file.py"
  GOENV_FILE_ARG="${GOENV_TEST_DIR}/dir1/file.py" run goenv echo GOENV_DIR
  assert_output "${GOENV_TEST_DIR}/dir1"
}

@test "should follow symlink of file argument (#379, #404)" {
  mkdir -p "${GOENV_TEST_DIR}/dir1"
  mkdir -p "${GOENV_TEST_DIR}/dir2"
  touch "${GOENV_TEST_DIR}/dir1/file.py"
  ln -s "${GOENV_TEST_DIR}/dir1/file.py" "${GOENV_TEST_DIR}/dir2/symlink.py"
  GOENV_FILE_ARG="${GOENV_TEST_DIR}/dir2/symlink.py" run goenv echo GOENV_DIR
  assert_output "${GOENV_TEST_DIR}/dir1"
}
