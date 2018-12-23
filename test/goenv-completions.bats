#!/usr/bin/env bats

load test_helper

create_command() {
  bin="${GOENV_TEST_DIR}/bin"
  mkdir -p "$bin"
  echo "$2" > "${bin}/$1"
  chmod +x "${bin}/$1"
}

@test "has usage instructions" {
  run goenv-help --usage completions
  assert_success "Usage: goenv completions <command> [arg1 arg2...]"
}

@test "it returns '--help' for command with no completion support" {
  create_command "goenv-hello" "#!$BASH
    echo hello"
  run goenv-completions hello
  assert_success "--help"
}

@test "it returns '--help' as first argument for command with completion support" {
  create_command "goenv-hello" "#!$BASH
# Provide goenv completions
if [[ \$1 = --complete ]]; then
  echo not_important
else
  exit 1
fi"
  run goenv-completions hello
  assert_success
  assert_line '--help'
}

@test "it returns specified command with completion support's completion suggestions" {
  create_command "goenv-hello" "#!$BASH
# Provide goenv completions
if [[ \$1 = --complete ]]; then
  echo hello
else
  exit 1
fi"
  run goenv-completions hello
  assert_success
  assert_output <<OUT
--help
hello
OUT
}

@test "it returns as 'completion' commands only commands that have '# Provide goenv completions'" {
  create_command "goenv-hello" "#!$BASH
# Provide goenv completions
if [[ \$1 = --complete ]]; then
  echo hello
else
  exit 1
fi"
  run goenv-completions hello
  assert_success
  assert_output <<OUT
--help
hello
OUT

  create_command "goenv-world" "#!$BASH
echo 'will not be seen'
fi"
  run goenv-completions world
  assert_success '--help'
}

@test "it forwards extra arguments to specified command with completion support" {
  create_command "goenv-hello" "#!$BASH
# provide goenv completions
if [[ \$1 = --complete ]]; then
  shift 1
  for arg; do echo \$arg; done
else
  exit 1
fi"
  run goenv-completions hello happy world
  assert_success
  assert_output <<OUT
--help
happy
world
OUT
}
