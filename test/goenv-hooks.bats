#!/usr/bin/env bats

load test_helper

@test "has completion support" {
  run goenv-hooks --complete

  assert_success <<OUT
exec
rehash
version-name
version-origin
which
OUT
}

@test "prints usage help when no arguments are given" {
  run goenv-hooks
  assert_failure "Usage: goenv hooks <command>"
}

@test "prints list of hooks ending with '.bash', for given command" {
  path1="${GOENV_TEST_DIR}/goenv.d"
  GOENV_HOOK_PATH="$path1"

  create_hook exec "hello.bash"
  create_hook exec "ahoy.bash"
  # NOTE: This command is expecte to be ignore since it's not ending with '.bash'
  create_hook exec "invalid.sh"

  # NOTE: This command is expected to be ignored since it's for a different command
  create_hook which "ignored.bash"

  path2="${GOENV_TEST_DIR}/etc/goenv_hooks"
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

@test "prints lists of hooks with spaces in path for given command" {
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

@test "prints list of hooks and resolves relative paths for given command" {
  GOENV_HOOK_PATH="${GOENV_TEST_DIR}/goenv.d"
  create_hook exec "hello.bash"
  mkdir -p "$HOME"

  GOENV_HOOK_PATH="${HOME}/../goenv.d" run goenv-hooks exec
  assert_success "${GOENV_TEST_DIR}/goenv.d/exec/hello.bash"
}

@test "prints list of hooks and resolves symlinks for given command" {
  path="${GOENV_TEST_DIR}/goenv.d"
  mkdir -p "${path}/exec"
  mkdir -p "$HOME"
  touch "${HOME}/hola.bash"

  ln -s "../../home/hola.bash" "${path}/exec/hello.bash"

  touch "${path}/exec/bright.sh"
  ln -s "bright.sh" "${path}/exec/world.bash"

  GOENV_HOOK_PATH="$path" run goenv-hooks exec
  assert_success <<OUT
${HOME}/hola.bash
${GOENV_TEST_DIR}/goenv.d/exec/bright.sh
OUT
}
