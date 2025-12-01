#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage init
  assert_success_out <<'OUT'
Usage: eval "$(goenv init - [--no-rehash] [<shell>])"
OUT
}

@test "has completion support" {
  run goenv-init --complete
  assert_success_out <<OUT
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

  assert_line 0 "export GOENV_SHELL=bash"
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

  assert_success_out <<'OUT'
# Load goenv automatically by appending
# the following to ~/.bash_profile:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'zsh'" {
  run goenv-init zsh

  assert_success_out <<'OUT'
# Load goenv automatically by appending
# the following to ~/.zshrc:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'fish'" {
  run goenv-init fish

  assert_success_out <<OUT
# Load goenv automatically by appending
# the following to ~/.config/fish/config.fish:

status --is-interactive; and source (goenv init -|psub)
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is 'ksh'" {
  run goenv-init ksh

  assert_success_out <<'OUT'
# Load goenv automatically by appending
# the following to ~/.profile:

eval "$(goenv init -)"
OUT
}

@test "prints usage snippet when no '-' argument is given, but shell given is none of the well known ones" {
  run goenv-init magicalshell

  assert_success_out <<'OUT'
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

  assert_line 0  'export GOENV_SHELL=bash'
  assert_line 1  "export GOENV_ROOT=$GOENV_ROOT"
  assert_line 2  'if [ -z "${GOENV_RC_FILE:-}" ]; then'
  assert_line 3  '  GOENV_RC_FILE="${HOME}/.goenvrc"'
  assert_line 4  'fi'
  assert_line 5  'if [ -e "${GOENV_RC_FILE:-}" ]; then'
  assert_line 6  '  source "${GOENV_RC_FILE}"'
  assert_line 7  'fi'
  assert_line 8  'if [ "${PATH#*$GOENV_ROOT/shims}" = "${PATH}" ]; then'
  assert_line 9  '  if [ "${GOENV_PATH_ORDER:-}" = "front" ] ; then'
  assert_line 10 '    export PATH="${GOENV_ROOT}/shims:${PATH}"'
  assert_line 11 '  else'
  assert_line 12 '    export PATH="${PATH}:${GOENV_ROOT}/shims"'
  assert_line 13 '  fi'
  assert_line 14 'fi'
  assert_line 15 "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.bash'"
  assert_line 16 'command goenv rehash 2>/dev/null'
  assert_line 17 'goenv() {'
  assert_line 18 '  local command'
  assert_line 19 '  command="$1"'
  assert_line 20 '  if [ "$#" -gt 0 ]; then'
  assert_line 21 '    shift'
  assert_line 22 '  fi'
  assert_line 23 '  case "$command" in'
  assert_line 24 '  rehash|shell)'
  assert_line 25 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 26 '  *)'
  assert_line 27 '    command goenv "$command" "$@";;'
  assert_line 28 '  esac'
  assert_line 29 '}'


  assert_success
}

@test "prints bootstrap script with auto-completion when '-' and 'zsh' are specified" {
  run goenv-init - zsh

  assert_line 0  'export GOENV_SHELL=zsh'
  assert_line 1  "export GOENV_ROOT=$GOENV_ROOT"
  assert_line 2  'if [ -z "${GOENV_RC_FILE:-}" ]; then'
  assert_line 3  '  GOENV_RC_FILE="${HOME}/.goenvrc"'
  assert_line 4  'fi'
  assert_line 5  'if [ -e "${GOENV_RC_FILE:-}" ]; then'
  assert_line 6  '  source "${GOENV_RC_FILE}"'
  assert_line 7  'fi'
  assert_line 8  'if [ "${PATH#*$GOENV_ROOT/shims}" = "${PATH}" ]; then'
  assert_line 9  '  if [ "${GOENV_PATH_ORDER:-}" = "front" ] ; then'
  assert_line 10 '    export PATH="${GOENV_ROOT}/shims:${PATH}"'
  assert_line 11 '  else'
  assert_line 12 '    export PATH="${PATH}:${GOENV_ROOT}/shims"'
  assert_line 13 '  fi'
  assert_line 14 'fi'
  assert_line 15 "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.zsh'"
  assert_line 16 'command goenv rehash 2>/dev/null'
  assert_line 17 'goenv() {'
  assert_line 18 '  local command'
  assert_line 19 '  command="$1"'
  assert_line 20 '  if [ "$#" -gt 0 ]; then'
  assert_line 21 '    shift'
  assert_line 22 '  fi'
  assert_line 23 '  case "$command" in'
  assert_line 24 '  rehash|shell)'
  assert_line 25 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 26 '  *)'
  assert_line 27 '    command goenv "$command" "$@";;'
  assert_line 28 '  esac'
  assert_line 29 '}'

  assert_success
}

@test "prints bootstrap script with auto-completion when '-' and 'fish' are specified" {
  run goenv-init - fish

  assert_line 0  'set -gx GOENV_SHELL fish'
  assert_line 1  "set -gx GOENV_ROOT $GOENV_ROOT"
  assert_line 2  'if test -z $GOENV_RC_FILE'
  assert_line 3  '  set GOENV_RC_FILE $HOME/.goenvrc'
  assert_line 4  'end'
  assert_line 5  'if test -e $GOENV_RC_FILE'
  assert_line 6  '  source $GOENV_RC_FILE'
  assert_line 7  'end'
  assert_line 8  'if not contains $GOENV_ROOT/shims $PATH'
  assert_line 9  '  if test "$GOENV_PATH_ORDER" = "front"'
  assert_line 10 '    set -gx PATH $GOENV_ROOT/shims $PATH'
  assert_line 11 '  else'
  assert_line 12 '    set -gx PATH $PATH $GOENV_ROOT/shims'
  assert_line 13 '  end'
  assert_line 14 'end'
  assert_line 15 "source '$BATS_TEST_DIRNAME/../libexec/../completions/goenv.fish'"
  assert_line 16 'command goenv rehash 2>/dev/null'
  assert_line 17 'function goenv'
  assert_line 18 '  set command $argv[1]'
  assert_line 19 '  set -e argv[1]'
  assert_line 20 '  switch "$command"'
  assert_line 21 '  case rehash shell'
  assert_line 22 '    source (goenv "sh-$command" $argv|psub)'
  assert_line 23 "  case '*'"
  assert_line 24 '    command goenv "$command" $argv'
  assert_line 25 '  end'
  assert_line 26 'end'

  assert_success
}

@test "prints bootstrap script without auto-completion when '-' and 'ksh' are specified" {
  run goenv-init - ksh

  assert_line 0  'export GOENV_SHELL=ksh'
  assert_line 1  "export GOENV_ROOT=$GOENV_ROOT"
  assert_line 2  'if [ -z "${GOENV_RC_FILE:-}" ]; then'
  assert_line 3  '  GOENV_RC_FILE="${HOME}/.goenvrc"'
  assert_line 4  'fi'
  assert_line 5  'if [ -e "${GOENV_RC_FILE:-}" ]; then'
  assert_line 6  '  source "${GOENV_RC_FILE}"'
  assert_line 7  'fi'
  assert_line 8  'if [ "${PATH#*$GOENV_ROOT/shims}" = "${PATH}" ]; then'
  assert_line 9  '  if [ "${GOENV_PATH_ORDER:-}" = "front" ] ; then'
  assert_line 10 '    export PATH="${GOENV_ROOT}/shims:${PATH}"'
  assert_line 11 '  else'
  assert_line 12 '    export PATH="${PATH}:${GOENV_ROOT}/shims"'
  assert_line 13 '  fi'
  assert_line 14 'fi'
  assert_line 15 'command goenv rehash 2>/dev/null'
  assert_line 16 'function goenv {'
  assert_line 17 '  typeset command'
  assert_line 18 '  command="$1"'
  assert_line 19 '  if [ "$#" -gt 0 ]; then'
  assert_line 20 '    shift'
  assert_line 21 '  fi'
  assert_line 22 '  case "$command" in'
  assert_line 23 '  rehash|shell)'
  assert_line 24 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 25 '  *)'
  assert_line 26 '    command goenv "$command" "$@";;'
  assert_line 27 '  esac'
  assert_line 28 '}'

  assert_success
}

@test "prints bootstrap script without auto-completion when '-' and unknown shell 'magicshell' are specified" {
  run goenv-init - magicshell

  # NOTE: This is very likely to be invalid for your specific shell
  assert_line 0  'export GOENV_SHELL=magicshell'
  assert_line 1  "export GOENV_ROOT=$GOENV_ROOT"
  assert_line 2  'if [ -z "${GOENV_RC_FILE:-}" ]; then'
  assert_line 3  '  GOENV_RC_FILE="${HOME}/.goenvrc"'
  assert_line 4  'fi'
  assert_line 5  'if [ -e "${GOENV_RC_FILE:-}" ]; then'
  assert_line 6  '  source "${GOENV_RC_FILE}"'
  assert_line 7  'fi'
  assert_line 8  'if [ "${PATH#*$GOENV_ROOT/shims}" = "${PATH}" ]; then'
  assert_line 9  '  if [ "${GOENV_PATH_ORDER:-}" = "front" ] ; then'
  assert_line 10 '    export PATH="${GOENV_ROOT}/shims:${PATH}"'
  assert_line 11 '  else'
  assert_line 12 '    export PATH="${PATH}:${GOENV_ROOT}/shims"'
  assert_line 13 '  fi'
  assert_line 14 'fi'
  assert_line 15 'command goenv rehash 2>/dev/null'
  assert_line 16 'goenv() {'
  assert_line 17 '  local command'
  assert_line 18 '  command="$1"'
  assert_line 19 '  if [ "$#" -gt 0 ]; then'
  assert_line 20 '    shift'
  assert_line 21 '  fi'
  assert_line 22 '  case "$command" in'
  assert_line 23 '  rehash|shell)'
  assert_line 24 '    eval "$(goenv "sh-$command" "$@")";;'
  assert_line 25 '  *)'
  assert_line 26 '    command goenv "$command" "$@";;'
  assert_line 27 '  esac'
  assert_line 28 '}'

  assert_success
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is not set for bash" {
  run goenv-init - bash
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 0 for bash" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=0 run goenv-init - bash
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "includes automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 1 for bash" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=1 run goenv-init - bash
  
  assert_success
  assert [ -n "$(echo "$output" | grep "__goenv_auto_detect_version()")" ]
  assert [ -n "$(echo "$output" | grep "PROMPT_COMMAND=")" ]
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is not set for zsh" {
  run goenv-init - zsh
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 0 for zsh" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=0 run goenv-init - zsh
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "includes automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 1 for zsh" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=1 run goenv-init - zsh
  
  assert_success
  assert [ -n "$(echo "$output" | grep "__goenv_auto_detect_version()")" ]
  assert [ -n "$(echo "$output" | grep "chpwd_functions")" ]
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is not set for fish" {
  run goenv-init - fish
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "does not include automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 0 for fish" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=0 run goenv-init - fish
  
  assert_success
  assert [ -z "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
}

@test "includes automatic version detection hook when GOENV_AUTOMATICALLY_DETECT_VERSION is 1 for fish" {
  GOENV_AUTOMATICALLY_DETECT_VERSION=1 run goenv-init - fish
  
  assert_success
  assert [ -n "$(echo "$output" | grep "__goenv_auto_detect_version")" ]
  assert [ -n "$(echo "$output" | grep "on-variable PWD")" ]
}

@test "includes PATH collision warning when system go exists and GOENV_PATH_ORDER is not front for bash" {
  # Create a fake go binary in PATH
  create_executable "${GOENV_TEST_DIR}/bin" "go"
  
  unset GOENV_DISABLE_PATH_WARNING
  run goenv-init - bash
  
  assert_success
  assert_line 'if [ "${GOENV_PATH_ORDER:-}" != "front" ]; then'
  assert [ -n "$(echo "$output" | grep 'WARNING: System.*go.*found')" ]
}

@test "does not include PATH collision warning when GOENV_DISABLE_PATH_WARNING is set to 1" {
  create_executable "${GOENV_TEST_DIR}/bin" "go"
  
  GOENV_DISABLE_PATH_WARNING=1 run goenv-init - bash
  
  assert_success
  refute_line '  system_go_path="$(command -v go 2>/dev/null || true)"'
}

@test "warning executes and shows message when system go exists and GOENV_PATH_ORDER not set" {
  # Create a fake go binary
  create_executable "${GOENV_TEST_DIR}/bin" "go"
  
  # Don't disable the warning for this test
  unset GOENV_DISABLE_PATH_WARNING
  unset GOENV_PATH_ORDER
  
  # Eval the init output and capture stderr
  run bash -c 'eval "$(GOENV_DISABLE_PATH_WARNING=0 goenv-init - bash 2>&1)" 2>&1'
  
  assert_success
  assert [ -n "$(echo "$output" | grep 'WARNING: System.*go.*found')" ]
  assert [ -n "$(echo "$output" | grep 'GOENV_PATH_ORDER=front')" ]
}

@test "warning does not execute when GOENV_PATH_ORDER is set to front" {
  create_executable "${GOENV_TEST_DIR}/bin" "go"
  
  unset GOENV_DISABLE_PATH_WARNING
  export GOENV_PATH_ORDER=front
  
  run bash -c 'eval "$(GOENV_DISABLE_PATH_WARNING=0 goenv-init - bash 2>&1)" 2>&1'
  
  assert_success
  assert [ -z "$(echo "$output" | grep 'WARNING')" ]
}
