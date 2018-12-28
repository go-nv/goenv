#!/usr/bin/env bats

load test_helper

@test "installs go-build into PREFIX" {
  cd "$TMP"
  PREFIX="${PWD}/usr" run "${BATS_TEST_DIRNAME}/../install.sh"
  assert_success ""

  cd usr

  assert [ -x bin/go-build ]
  assert [ -x bin/goenv-install ]
  assert [ -x bin/goenv-uninstall ]

  assert [ -e share/go-build/1.8.4 ]
  assert [ -e share/go-build/1.11.1 ]
}

@test "build definitions don't have the executable bit" {
  cd "$TMP"
  PREFIX="${PWD}/usr" run "${BATS_TEST_DIRNAME}/../install.sh"
  assert_success ""

  run $BASH -c 'ls -l usr/share/go-build | tail -2 | cut -c1-10'
  assert_output <<OUT
-rw-r--r--
-rw-r--r--
OUT
}

@test "overwrites old installation" {
  cd "$TMP"
  mkdir -p bin share/go-build
  touch bin/go-build
  touch share/go-build/1.2.2

  PREFIX="$PWD" run "${BATS_TEST_DIRNAME}/../install.sh"
  assert_success ""

  assert [ -x bin/go-build ]
  run grep "install_linux_64bit" share/go-build/1.2.2
  assert_success
}

@test "unrelated files are untouched" {
  cd "$TMP"
  mkdir -p bin share/bananas
  chmod g-w bin
  touch bin/bananas
  touch share/bananas/docs

  PREFIX="$PWD" run "${BATS_TEST_DIRNAME}/../install.sh"
  assert_success ""

  assert [ -e bin/bananas ]
  assert [ -e share/bananas/docs ]

  run ls -ld bin
  assert_equal "r-x" "${output:4:3}"
}
