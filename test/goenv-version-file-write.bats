#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
}

@test "has usage instructions" {
  run goenv-help --usage version-file-write
  assert_success_out <<OUT
Usage: goenv version-file-write <file> <version>...
OUT
}

@test "prints usage instructions when 2 arguments aren't specified" {
  run goenv-version-file-write

  assert_failure "Usage: goenv version-file-write <file> <version>..."

  run goenv-version-file-write "one"
  assert_failure "Usage: goenv version-file-write <file> <version>..."
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

@test "remove local version when 'system' version is given and any local version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-local latest
  assert_success ""
  assert [ "$(cat .go-version)" = "1.2.3" ]

  # Fail first because system version doesn't exist
  run goenv-local system
  assert_failure "goenv: system version not found in PATH"

  # Make sure system version exists this time
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"
  # Make test harder by referencing not installed version
  echo "4.5.6" > .go-version

  run goenv-local system
  assert_success "goenv: using system version instead of 4.5.6 now"
  assert [ ! -f ".go-version" ]
  run goenv-local
  assert_failure "goenv: no local version configured for this directory"
}


@test "remove global version when 'system' version is given and any global version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-global latest
  assert_success ""
  assert [ "$(cat ${GOENV_ROOT}/version)" = "1.2.3" ]

  # Fail first because system version doesn't exist
  run goenv-global system
  assert_failure "goenv: system version not found in PATH"

  # Make sure system version exists this time
  mkdir -p "${GOENV_TEST_DIR}/bin"
  touch "${GOENV_TEST_DIR}/bin/go"
  chmod +x "${GOENV_TEST_DIR}/bin/go"
  # Make test harder by referencing not installed version
  echo "4.5.6" > ${GOENV_ROOT}/version

  run goenv-global system
  assert_success "goenv: using system version instead of 4.5.6 now"
  assert [ ! -f "${GOENV_ROOT}/version" ]
  run goenv-global
  assert_success "system"
}

