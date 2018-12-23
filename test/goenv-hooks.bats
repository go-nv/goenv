#!/usr/bin/env bats

load test_helper

@test "prints usage help given no argument" {
  run goenv-hooks
  assert_failure "Usage: goenv hooks <command>"
}

@test "prints list of hooks" {
  path1="${GOENV_TEST_DIR}/goenv.d"
  path2="${GOENV_TEST_DIR}/etc/goenv_hooks"
  GOENV_HOOK_PATH="$path1"
  create_hook exec "hello.bash"
  create_hook exec "ahoy.bash"
  create_hook exec "invalid.sh"
  create_hook which "boom.bash"
  GOENV_HOOK_PATH="$path2"
  create_hook exec "bueno.bash"

  GOENV_HOOK_PATH="$path1:$path2" run goenv-hooks exec
  assert_success
  assert_output <<OUT
${GOENV_TEST_DIR}/goenv.d/exec/ahoy.bash
${GOENV_TEST_DIR}/goenv.d/exec/hello.bash
${GOENV_TEST_DIR}/etc/goenv_hooks/exec/bueno.bash
OUT
}

@test "supports hook paths with spaces" {
  path1="${GOENV_TEST_DIR}/my hooks/goenv.d"
  path2="${GOENV_TEST_DIR}/etc/goenv hooks"
  GOENV_HOOK_PATH="$path1"
  create_hook exec "hello.bash"
  GOENV_HOOK_PATH="$path2"
  create_hook exec "ahoy.bash"

  GOENV_HOOK_PATH="$path1:$path2" run goenv-hooks exec
  assert_success
  assert_output <<OUT
${GOENV_TEST_DIR}/my hooks/goenv.d/exec/hello.bash
${GOENV_TEST_DIR}/etc/goenv hooks/exec/ahoy.bash
OUT
}

@test "resolves relative paths" {
  GOENV_HOOK_PATH="${GOENV_TEST_DIR}/goenv.d"
  create_hook exec "hello.bash"
  mkdir -p "$HOME"

  GOENV_HOOK_PATH="${HOME}/../goenv.d" run goenv-hooks exec
  assert_success "${GOENV_TEST_DIR}/goenv.d/exec/hello.bash"
}

@test "resolves symlinks" {
  path="${GOENV_TEST_DIR}/goenv.d"
  mkdir -p "${path}/exec"
  mkdir -p "$HOME"
  touch "${HOME}/hola.bash"
  ln -s "../../home/hola.bash" "${path}/exec/hello.bash"
  touch "${path}/exec/bright.sh"
  ln -s "bright.sh" "${path}/exec/world.bash"

  GOENV_HOOK_PATH="$path" run goenv-hooks exec
  assert_success
  assert_output <<OUT
${HOME}/hola.bash
${GOENV_TEST_DIR}/goenv.d/exec/bright.sh
OUT
}
