#!/usr/bin/env bats

load test_helper

@test "blank invocation" {
  run goenv
  assert_failure
  assert_line 0 "$(goenv---version)"
}

@test "invalid command" {
  run goenv does-not-exist
  assert_failure
  assert_output "goenv: no such command \`does-not-exist'"
}

@test "default GOENV_ROOT" {
  GOENV_ROOT="" HOME=/home/mislav run goenv root
  assert_success
  assert_output "/home/mislav/.goenv"
}

@test "inherited GOENV_ROOT" {
  GOENV_ROOT=/opt/goenv run goenv root
  assert_success
  assert_output "/opt/goenv"
}

@test "default GOENV_DIR" {
  run goenv echo GOENV_DIR
  assert_output "$(pwd)"
}

@test "inherited GOENV_DIR" {
  dir="${BATS_TMPDIR}/myproject"
  mkdir -p "$dir"
  GOENV_DIR="$dir" run goenv echo GOENV_DIR
  assert_output "$dir"
}

@test "invalid GOENV_DIR" {
  dir="${BATS_TMPDIR}/does-not-exist"
  assert [ ! -d "$dir" ]
  GOENV_DIR="$dir" run goenv echo GOENV_DIR
  assert_failure
  assert_output "goenv: cannot change working directory to \`$dir'"
}

@test "adds its own libexec to PATH" {
  run goenv echo "PATH"
  assert_success "${BATS_TEST_DIRNAME%/*}/libexec:$PATH"
}

@test "adds plugin bin dirs to PATH" {
  mkdir -p "$GOENV_ROOT"/plugins/go-build/bin
  mkdir -p "$GOENV_ROOT"/plugins/goenv-each/bin
  run goenv echo -F: "PATH"
  assert_success
  assert_line 0 "${BATS_TEST_DIRNAME%/*}/libexec"
  assert_line 1 "${GOENV_ROOT}/plugins/go-build/bin"
  assert_line 2 "${GOENV_ROOT}/plugins/goenv-each/bin"
}

@test "GOENV_HOOK_PATH preserves value from environment" {
  GOENV_HOOK_PATH=/my/hook/path:/other/hooks run goenv echo -F: "GOENV_HOOK_PATH"
  assert_success
  assert_line 0 "/my/hook/path"
  assert_line 1 "/other/hooks"
  assert_line 2 "${GOENV_ROOT}/goenv.d"
}

@test "GOENV_HOOK_PATH includes goenv built-in plugins" {
  unset GOENV_HOOK_PATH
  run goenv echo "GOENV_HOOK_PATH"
  assert_success "${GOENV_ROOT}/goenv.d:${BATS_TEST_DIRNAME%/*}/goenv.d:/usr/local/etc/goenv.d:/etc/goenv.d:/usr/lib/goenv/hooks"
}
