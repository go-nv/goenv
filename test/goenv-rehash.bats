#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage rehash
  assert_success <<'OUT'
Usage: goenv rehash
OUT
}

@test "creates 'GOENV_ROOT/shims' when it does not exist" {
  assert [ ! -d "${GOENV_ROOT}/shims" ]
  run goenv-rehash
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

  run goenv-rehash
  assert_failure "goenv: cannot rehash: ${GOENV_ROOT}/shims isn't writable"
}

@test "fails when 'GOENV_ROOT/shims/.goenv-shim' is present, meaning rehash in progress or interrupted via SIGKILL" {
  mkdir -p "${GOENV_ROOT}/shims"
  touch "${GOENV_ROOT}/shims/.goenv-shim"

  run goenv-rehash
  assert_failure "goenv: cannot rehash: ${GOENV_ROOT}/shims/.goenv-shim exists (rehash in progress or interrupted via SIGKILL)"
}

@test "succeeds in creating executable shims for binaries present in 'GOENV_ROOT/versions/<version>/bin'" {
  export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
  create_executable "1.11.1" "go"
  create_executable "1.9.0" "godoc"

  assert [ ! -e "${GOENV_ROOT}/shims/go" ]
  assert [ -e "${GOENV_ROOT}/versions/1.11.1/bin/go" ]
  assert [ -e "${GOENV_ROOT}/versions/1.9.0/bin/godoc" ]

  run goenv-rehash
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

  run goenv-rehash
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

  assert [ -x "${GOENV_ROOT}/shims/go" ]
}

@test "succeeds in creating shims for binaries present in 'GOENV_ROOT/versions/<version>/bin', even though 'version' contains spaces" {
  create_executable "dirname1 p247" "go"

  assert [ ! -e "${GOENV_ROOT}/shims/go" ]

  run type go
  assert_contains $output 'type: go: not found'

  run goenv-rehash
  assert_success ""

  run ls "${GOENV_ROOT}/shims"
  assert_success
  assert_output <<OUT
go
OUT

  run type go
  assert_output "go is ${GOENV_ROOT}/shims/go"
}

@test "carries original IFS within hooks" {
  create_hook rehash hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
exit
SH

  IFS=$' \t\n' run goenv-rehash
  assert_success
  assert_output "HELLO=:hello:ugly:world:again"
}
