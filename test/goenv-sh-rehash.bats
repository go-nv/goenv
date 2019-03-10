#!/usr/bin/env bats

load test_helper

@test "has usage instructions for goenv-sh-rehash" {
  run goenv-help --usage sh-rehash
  assert_success <<'OUT'
Usage: goenv sh-rehash
OUT
}

@test "has completion support (but pointless)" {
  run goenv-sh-rehash --complete
  assert_success
  assert_output ""
}

@test "when current set 'version' is 'system', it does not export GOPATH and GOROOT env variables" {
  export GOENV_VERSION=system
  export GOENV_SHELL=bash
  run goenv-sh-rehash
  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'bash', it only echoes rehash of binaries" {
  export GOENV_SHELL=bash

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 run goenv-sh-rehash

  assert_output "hash -r 2>/dev/null || true"
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'fish', it does not echo anything" {
  export GOENV_SHELL=fish

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 run goenv-sh-rehash

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'ksh', it only echoes rehash of binaries" {
  export GOENV_SHELL=ksh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 run goenv-sh-rehash

  assert_output "hash -r 2>/dev/null || true"
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'zsh', it only echoes rehash of binaries" {
  export GOENV_SHELL=zsh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 run goenv-sh-rehash

  assert_output "hash -r 2>/dev/null || true"
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash', it echoes export of 'GOPATH' and rehash of binaries" {
  export GOENV_SHELL=bash

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh', it echoes export of 'GOPATH' and rehash of binaries" {
  export GOENV_SHELL=ksh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh', it echoes export of 'GOPATH' and rehash of binaries" {
  export GOENV_SHELL=zsh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish', it echoes only export of 'GOPATH'" {
  export GOENV_SHELL=fish

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
set -gx GOPATH "${HOME}/go/1.12.0"
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash' and 'GOENV_GOPATH_PREFIX' is empty, it echoes export of 'GOROOT', 'GOPATH=\$HOME/go' and rehash of binaries" {
  export GOENV_SHELL=bash

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh' and 'GOENV_GOPATH_PREFIX' is empty, it echoes export of 'GOROOT', 'GOPATH=\$HOME/go' and rehash of binaries" {
  export GOENV_SHELL=ksh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh' and 'GOENV_GOPATH_PREFIX' is empty, it echoes export of 'GOROOT', 'GOPATH=\$HOME/go' and rehash of binaries" {
  export GOENV_SHELL=zsh

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="${HOME}/go/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish' and 'GOENV_GOPATH_PREFIX' is empty, it echoes export of 'GOROOT', 'GOPATH=\$HOME/go'" {
  export GOENV_SHELL=fish

  create_version "1.12.0"

  GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
set -gx GOROOT "$(GOENV_VERSION=1.12.0 goenv-prefix)"
set -gx GOPATH "${HOME}/go/1.12.0"
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash' and 'GOENV_GOPATH_PREFIX' is present, it echoes export of 'GOROOT', 'GOPATH=<set>' and rehash of binaries" {
  export GOENV_SHELL=bash

  create_version "1.12.0"

  GOENV_GOPATH_PREFIX=/tmp/example GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="/tmp/example/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh' and 'GOENV_GOPATH_PREFIX' is present, it echoes export of 'GOROOT', 'GOPATH=<set>' and rehash of binaries" {
  export GOENV_SHELL=ksh

  create_version "1.12.0"

  GOENV_GOPATH_PREFIX=/tmp/example GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="/tmp/example/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh' and 'GOENV_GOPATH_PREFIX' is present, it echoes export of 'GOROOT', 'GOPATH=<set>' and rehash of binaries" {
  export GOENV_SHELL=zsh

  create_version "1.12.0"

  GOENV_GOPATH_PREFIX=/tmp/example GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
export GOROOT="$(GOENV_VERSION=1.12.0 goenv-prefix)"
export GOPATH="/tmp/example/1.12.0"
hash -r 2>/dev/null || true
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish' and 'GOENV_GOPATH_PREFIX' is present, it echoes export of 'GOROOT' and 'GOPATH=<set>'" {
  export GOENV_SHELL=fish

  create_version "1.12.0"

  GOENV_GOPATH_PREFIX=/tmp/example GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 run goenv-sh-rehash

  assert_output <<OUT
set -gx GOROOT "$(GOENV_VERSION=1.12.0 goenv-prefix)"
set -gx GOPATH "/tmp/example/1.12.0"
OUT
  assert_success
}

@test "creates 'GOENV_ROOT/shims' when it does not exist" {
  assert [ ! -d "${GOENV_ROOT}/shims" ]
  run goenv-sh-rehash
  assert_success ""
  assert [ -d "${GOENV_ROOT}/shims" ]
  rmdir "${GOENV_ROOT}/shims"
}

@test "fails when shims directory at 'GOENV_ROOT/shims' is not writable" {
  if [ "$(whoami)" = "root" ]; then
      skip "running as root. permissions won't matter."
  fi

  mkdir -p "${GOENV_ROOT}/shims"
  chmod 0444 "${GOENV_ROOT}/shims"

  run goenv-sh-rehash
  assert_failure "goenv: cannot rehash: ${GOENV_ROOT}/shims isn't writable"
}

@test "fails when 'GOENV_ROOT/shims/.goenv-shim' is present, meaning rehash in progress or interrupted via SIGKILL" {
  mkdir -p "${GOENV_ROOT}/shims"
  touch "${GOENV_ROOT}/shims/.goenv-shim"

  run goenv-sh-rehash
  assert_failure "goenv: cannot rehash: ${GOENV_ROOT}/shims/.goenv-shim exists (rehash in progress or interrupted via SIGKILL)"
}

@test "succeeds in creating executable shims for binaries present in 'GOENV_ROOT/versions/<version>/bin'" {
  export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
  create_executable "1.11.1" "go"
  create_executable "1.9.0" "godoc"

  assert [ ! -e "${GOENV_ROOT}/shims/go" ]
  assert [ -e "${GOENV_ROOT}/versions/1.11.1/bin/go" ]
  assert [ -e "${GOENV_ROOT}/versions/1.9.0/bin/godoc" ]

  run goenv-sh-rehash
  assert_success ""

  run ls "${GOENV_ROOT}/shims"
  assert_success
  assert_output <<OUT
go
godoc
OUT

  run "$(cat ${GOENV_ROOT}/shims/go)"

  # NOTE: Don't assert line 0 since bats modifies it
  assert_line 1  'set -e'
  assert_line 2  '[ -n "$GOENV_DEBUG" ] && set -x'
  assert_line 3  'program="${0##*/}"'
  assert_line 4  'if [[ "$program" = "go"* ]]; then'
  assert_line 5  '  for arg; do'
  assert_line 6  '    case "$arg" in'
  assert_line 7  '    -c* | -- ) break ;;'
  assert_line 8  '    */* )'
  assert_line 9  '      if [ -f "$arg" ]; then'
  assert_line 10 '        export GOENV_FILE_ARG="$arg"'
  assert_line 11 '        break'
  assert_line 12 '      fi'
  assert_line 13 '      ;;'
  assert_line 14 '    esac'
  assert_line 15 '  done'
  assert_line 16 'fi'
  assert_line 17 "export GOENV_ROOT=\"$GOENV_ROOT\""
  # TODO: Fix line 18 assertion showing "No such file or directory"
#  assert_line 18 "exec \"$(command -v goenv)\" exec \"\$program\" \"\$@\""

  assert [ -x "${GOENV_ROOT}/shims/go" ]

  run "$(cat ${GOENV_ROOT}/shims/godoc)"

  # NOTE: Don't assert line 0 since bats modifies it
  assert_line 1  'set -e'
  assert_line 2  '[ -n "$GOENV_DEBUG" ] && set -x'
  assert_line 3  'program="${0##*/}"'
  assert_line 4  'if [[ "$program" = "go"* ]]; then'
  assert_line 5  '  for arg; do'
  assert_line 6  '    case "$arg" in'
  assert_line 7  '    -c* | -- ) break ;;'
  assert_line 8  '    */* )'
  assert_line 9  '      if [ -f "$arg" ]; then'
  assert_line 10 '        export GOENV_FILE_ARG="$arg"'
  assert_line 11 '        break'
  assert_line 12 '      fi'
  assert_line 13 '      ;;'
  assert_line 14 '    esac'
  assert_line 15 '  done'
  assert_line 16 'fi'
  assert_line 17 "export GOENV_ROOT=\"$GOENV_ROOT\""
  # TODO: Fix line 18 assertion showing "No such file or directory"
#  assert_line 18 "exec \"$(command -v goenv)\" exec \"\$program\" \"\$@\""

  assert [ -x "${GOENV_ROOT}/shims/godoc" ]
}

@test "removes stale shims that are not present anymore in 'GOENV_ROOT/versions/<version>/bin' and rehashes" {
  mkdir -p "${GOENV_ROOT}/shims"
  touch "${GOENV_ROOT}/shims/oldshim1"
  chmod +x "${GOENV_ROOT}/shims/oldshim1"

  run type oldshim1
  assert_output "oldshim1 is ${GOENV_ROOT}/shims/oldshim1"

  create_executable "1.11.1" "go"

  run type go
  assert_contains $output 'type: go: not found'

  run goenv-sh-rehash
  assert_success ""

  assert [ ! -e "${GOENV_ROOT}/shims/oldshim1" ]
  run type go
  assert_output "go is ${GOENV_ROOT}/shims/go"

  assert [ -x "${GOENV_ROOT}/shims/go" ]

  run "$(cat ${GOENV_ROOT}/shims/go)"

  # NOTE: Don't assert line 0 since bats modifies it
  assert_line 1  'set -e'
  assert_line 2  '[ -n "$GOENV_DEBUG" ] && set -x'
  assert_line 3  'program="${0##*/}"'
  assert_line 4  'if [[ "$program" = "go"* ]]; then'
  assert_line 5  '  for arg; do'
  assert_line 6  '    case "$arg" in'
  assert_line 7  '    -c* | -- ) break ;;'
  assert_line 8  '    */* )'
  assert_line 9  '      if [ -f "$arg" ]; then'
  assert_line 10 '        export GOENV_FILE_ARG="$arg"'
  assert_line 11 '        break'
  assert_line 12 '      fi'
  assert_line 13 '      ;;'
  assert_line 14 '    esac'
  assert_line 15 '  done'
  assert_line 16 'fi'
  assert_line 17 "export GOENV_ROOT=\"$GOENV_ROOT\""
  # TODO: Fix line 18 assertion showing "No such file or directory"
#  assert_line 18 "exec \"$(command -v goenv)\" exec \"\$program\" \"\$@\""

  assert [ -x "${GOENV_ROOT}/shims/go" ]
}

@test "succeeds in creating shims for binaries present in 'GOENV_ROOT/versions/<version>/bin', even though 'version' contains spaces" {
  create_executable "dirname1 p247" "go"

  assert [ ! -e "${GOENV_ROOT}/shims/go" ]

  run type go
  assert_contains $output 'type: go: not found'

  run goenv-sh-rehash
  assert_success ""

  run ls "${GOENV_ROOT}/shims"
  assert_success
  assert_output <<OUT
go
OUT

  run type go
  assert_output "go is ${GOENV_ROOT}/shims/go"
}

