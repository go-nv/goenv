#!/usr/bin/env bats

load test_helper

setup() {
  mkdir -p "${GOENV_ROOT}/versions/1.9.2"
  mkdir -p "${GOENV_ROOT}/versions/1.10.1"
}

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

  assert_success "1.10.1
1.9.2
commands
completions
exec
global
help
hooks
init
install
installed
latest
local
prefix
rehash
root
shell
shims
system
uninstall
version
version-file
version-file-read
version-file-write
version-name
version-origin
versions
whence
which"
}

@test "'commands --sh' returns only commands containing 'sh'" {
  run goenv-commands --sh
  assert_success "rehash
shell"

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
  assert_success "1.10.1
1.9.2
commands
completions
exec
global
help
hooks
init
install
installed
latest
local
prefix
rehash
root
shims
system
uninstall
version
version-file
version-file-read
version-file-write
version-name
version-origin
versions
whence
which"

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
