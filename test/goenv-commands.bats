#!/usr/bin/env bats

load test_helper

@test "has completion support" {
  run goenv-commands --complete

  assert_success <<OUT
--sh
--no-sh
OUT
}

@test "has usage instructions" {
  run goenv-help --usage commands
  assert_success "Usage: goenv commands [--sh|--no-sh]"
}

@test "'commands' returns all commands" {
  run goenv-commands

  assert_success
  assert_line "commands"
  assert_line "completions"
  assert_line "exec"
  assert_line "global"
  assert_line "help"
  assert_line "hooks"
  assert_line "init"
  assert_line "local"
  assert_line "prefix"
  assert_line "rehash"
  assert_line "root"
  assert_line "shell"
  assert_line "shims"
  assert_line "version"
  assert_line "--version"
  assert_line "version-file"
  assert_line "version-file-read"
  assert_line "version-file-write"
  assert_line "version-name"
  assert_line "version-origin"
  assert_line "versions"
  assert_line "whence"
  assert_line "which"
}

@test "'commands --sh' returns only commands containing 'sh'" {
  run goenv-commands --sh
  assert_success
  refute_line "commands"
  refute_line "completions"
  refute_line "exec"
  refute_line "global"
  refute_line "help"
  refute_line "hooks"
  refute_line "init"
  refute_line "local"
  refute_line "prefix"
  refute_line "root"
  refute_line "shims"
  refute_line "version"
  refute_line "--version"
  refute_line "version-file"
  refute_line "version-file-read"
  refute_line "version-file-write"
  refute_line "version-name"
  refute_line "version-origin"
  refute_line "versions"
  refute_line "whence"
  refute_line "which"

  assert_line "rehash"
  assert_line "shell"
}


@test "'commands --no-sh' returns only commands not containing 'sh'" {
  run goenv-commands --no-sh
  assert_success

  assert_line "commands"
  assert_line "completions"
  assert_line "exec"
  assert_line "global"
  assert_line "help"
  assert_line "hooks"
  assert_line "init"
  assert_line "local"
  assert_line "prefix"
  assert_line "rehash"
  assert_line "root"
  assert_line "shims"
  assert_line "version"
  assert_line "--version"
  assert_line "version-file"
  assert_line "version-file-read"
  assert_line "version-file-write"
  assert_line "version-name"
  assert_line "version-origin"
  assert_line "versions"
  assert_line "whence"
  assert_line "which"

  refute_line "shell"
}

@test "'commands --sh' with command  in path with spaces still returns it" {
  path="${GOENV_TEST_DIR}/my commands"
  cmd="${path}/goenv-sh-hello"
  mkdir -p "$path"
  touch "$cmd"
  chmod +x "$cmd"

  PATH="${path}:$PATH" run goenv-commands --sh
  assert_line "hello"
  # NOTE: Still verify expected behavior due to `--sh` usage
  assert_line "rehash"
  assert_line "shell"
}
