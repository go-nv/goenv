#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage init
  assert_success <<'OUT'
eval "$(goenv init - [--no-rehash] [<shell>])"
OUT
}

@test "has completion support" {
  run goenv-init --complete
  assert_success <<OUT
-
--no-rehash
bash
fish
ksh
zsh
OUT
}

@test 'detects parent shell when '-' argument is given only' {
  SHELL=/bin/false run goenv-init -

  assert_line 1 "export GOENV_SHELL=bash"
  assert_success
}


@test 'detects parent shell from script when '-' argument is given only' {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  cat > myscript.sh <<OUT
#!/bin/sh
eval "\$(goenv-init -)"
echo \$GOENV_SHELL
OUT

  chmod +x myscript.sh
  # NOTE: Run with a different shell to make sure detection works
  run ./myscript.sh /bin/bash
  # NOTE: It's 'sh' due to shebang in script specifying how to execute
  assert_success "sh"
}


@test "does not create GOENV_ROOT/{shims,versions} when no '-' argument is given" {
  run goenv-init

  assert_success
  assert [ ! -d "${GOENV_ROOT}/shims" ]
  assert [ ! -d "${GOENV_ROOT}/versions" ]
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'bash'" {
  run goenv-init bash

  assert_success
  assert_output <<'OUT'
# Load goenv automatically by appending
# the following to ~/.bash_profile:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'zsh'" {
  run goenv-init zsh

  assert_success
  assert_output <<'OUT'
# Load goenv automatically by appending
# the following to ~/.zshrc:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'fish'" {
  run goenv-init fish

  assert_success
  assert_output <<'OUT'
# Load goenv automatically by appending
# the following to ~/.config/fish/config.fish:

status --is-interactive; and source (goenv init -|psub)
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'ksh'" {
  run goenv-init ksh

  assert_success
  assert_output <<'OUT'
# Load goenv automatically by appending
# the following to ~/.profile:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is none of the well known ones" {
  run goenv-init magicalshell

  assert_success
  assert_output <<"OUT"
# Load goenv automatically by appending
# the following to <unknown shell: magicalshell, replace with your profile path>:

eval "$(goenv init -)"
OUT
}

@test "creates shims and versions directories when '-' argument is given" {
  assert [ ! -d "${GOENV_ROOT}/shims" ]
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-init -
  assert_success
  assert [ -d "${GOENV_ROOT}/shims" ]
  assert [ -d "${GOENV_ROOT}/versions" ]
}

@test "includes 'goenv rehash' when '-' is specified and '--no-rehash' is not specified" {
  run goenv-init -
  assert_success
  assert_line "command goenv rehash 2>/dev/null"
}


@test "does not include 'goenv rehash' when '-' and '--no-rehash' are specified" {
  run goenv-init - --no-rehash
  assert_success
  refute_line "command goenv rehash 2>/dev/null"
}

@test "prints bootstrap script with auto-completion when '-' and 'bash' are specified" {
  run goenv-init - bash

  assert_line 1  'export GOENV_SHELL=bash'
  assert_line 2  "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.bash'"
  assert_line 3  'command goenv rehash 2>/dev/null'
  assert_line 4  'goenv() {'
  assert_line 5  '  local command'
  assert_line 6  '  command="$1"'
  assert_line 7  '  if [ "$#" -gt 0 ]; then'
  assert_line 8  '    shift'
  assert_line 9  '  fi'
  assert_line 10 '  case "$command" in'
  assert_line 11 '  rehash|shell)'
  assert_line 12 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 13 '  *)'
  assert_line 14 '    command goenv "$command" "$@";;'
  assert_line 15 '  esac'
  assert_line 16 '}'

  assert_success
}

@test "prints bootstrap script with auto-completion when '-' and 'zsh' are specified" {
  run goenv-init - zsh

  assert_line 1  'export GOENV_SHELL=zsh'
  assert_line 2  "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.zsh'"
  assert_line 3  'command goenv rehash 2>/dev/null'
  assert_line 4  'goenv() {'
  assert_line 5  '  local command'
  assert_line 6  '  command="$1"'
  assert_line 7  '  if [ "$#" -gt 0 ]; then'
  assert_line 8  '    shift'
  assert_line 9  '  fi'
  assert_line 10 '  case "$command" in'
  assert_line 11 '  rehash|shell)'
  assert_line 12 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 13 '  *)'
  assert_line 14 '    command goenv "$command" "$@";;'
  assert_line 15 '  esac'
  assert_line 16 '}'

  assert_success
}

@test "prints bootstrap script with auto-completion when '-' and 'fish' are specified" {
  run goenv-init - fish

  assert_line 1  'set -gx GOENV_SHELL fish'
  assert_line 2  "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.fish'"
  assert_line 3  'command goenv rehash 2>/dev/null'
  assert_line 4  'function goenv'
  assert_line 5  '  set command $argv[1]'
  assert_line 6  '  set -e argv[1]'
  assert_line 7  '  switch "$command"'
  assert_line 8  '  case rehash shell'
  assert_line 9  '    source (goenv "sh-$command" $argv|psub)'
  assert_line 10 "  case '*'"
  assert_line 11 '    command goenv "$command" $argv'
  assert_line 12 '  end'
  assert_line 13 'end'

  assert_success
}

@test "prints bootstrap script without auto-completion when '-' and 'ksh' are specified" {
  run goenv-init - ksh

  assert_line 1  'export GOENV_SHELL=ksh'
  assert_line 2  'command goenv rehash 2>/dev/null'
  assert_line 3  'function goenv {'
  assert_line 4  '  typeset command'
  assert_line 5  '  command="$1"'
  assert_line 6  '  if [ "$#" -gt 0 ]; then'
  assert_line 7  '    shift'
  assert_line 8  '  fi'
  assert_line 9  '  case "$command" in'
  assert_line 10 '  rehash|shell)'
  assert_line 11 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 12 '  *)'
  assert_line 13 '    command goenv "$command" "$@";;'
  assert_line 14 '  esac'

  assert_success
}

@test "prints bootstrap script without auto-completion when '-' and unknown shell 'magicshell' are specified" {
  run goenv-init - magicshell

  # NOTE: This is very likely to be invalid for your specific shell
  assert_line 1  'export GOENV_SHELL=magicshell'
  assert_line 2  'command goenv rehash 2>/dev/null'
  assert_line 3  'goenv() {'
  assert_line 4  '  local command'
  assert_line 5  '  command="$1"'
  assert_line 6  '  if [ "$#" -gt 0 ]; then'
  assert_line 7  '    shift'
  assert_line 8  '  fi'
  assert_line 9  '  case "$command" in'
  assert_line 10 '  rehash|shell)'
  assert_line 11 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 12 '  *)'
  assert_line 13 '    command goenv "$command" "$@";;'
  assert_line 14 '  esac'
  assert_line 15 '}'

  assert_success
}

