#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage exec
  assert_success "Usage: goenv exec <command> [arg1 arg2...]"
}

@test "fails with usage instructions when no command is specified" {
  run goenv-exec
  assert_failure
  assert_output <<OUT
Usage: goenv exec <command> [arg1 arg2...]
OUT
}

@test "fails with version that's not installed but specified by GOENV_VERSION" {
  GOENV_VERSION=1.6.1 run goenv-exec go version
  assert_failure "goenv: version '1.6.1' is not installed (set by GOENV_VERSION environment variable)"
}

@test "fails with version that's not installed but specified by '.go-version' file" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  echo 1.6.1 > .go-version

  run goenv-exec go build
  assert_failure "goenv: version '1.6.1' is not installed (set by $PWD/.go-version)"
}

@test "succeeds with version that's installed and specified by GOENV_VERSION" {
  export GOENV_VERSION="1.6.1"
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  goenv-rehash
  run goenv-exec Zgo123unique

  assert_output ""
  assert_success
}

@test "succeeds with version that's installed and specified by '.go-version' file" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  echo 1.6.1 > .go-version
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  goenv-rehash
  run goenv-exec Zgo123unique
  assert_success
}

@test "completes with names of executables for version that's specified by GOENV_VERSION environment variable" {
  create_executable "1.6.1" "Zgo123unique" "#!/bin/sh"

  GOENV_VERSION=1.6.1 goenv-rehash
  GOENV_VERSION=1.6.1 run goenv-completions exec
  assert_success
  assert_output <<OUT
--help
Zgo123unique
OUT
}

@test "carries original IFS within hooks for version that's specified by GOENV_VERSION environment variable" {
  create_hook exec hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  GOENV_VERSION=system IFS=$' \t\n' run goenv-exec env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "forwards all arguments for command that's specified by GOENV_VERSION environment variable" {
  create_executable "1.6.1" "go" <<SH
#!$BASH
echo \$0
for arg; do
  # hack to avoid bash builtin echo which can't output '-e'
  printf "  %s\\n" "\$arg"
done
SH

  GOENV_VERSION=1.6.1 run goenv-exec go run "/path to/go script.go" -- extra args
  assert_success
  assert_output <<OUT
${GOENV_ROOT}/versions/1.6.1/bin/go
  run
  /path to/go script.go
  --
  extra
  args
OUT
}

@test "when current set 'version' is 'system', it does not export GOPATH and GOROOT env variables" {
  create_file "${GOENV_TEST_DIR}/go-paths"
  chmod +x "${GOENV_TEST_DIR}/go-paths"
  cat > "${GOENV_TEST_DIR}/go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_VERSION=system GOENV_SHELL=bash GOROOT="" GOPATH="" PATH="$GOENV_TEST_DIR:$PATH" run goenv-exec go-paths

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'bash', it does not export GOPATH or GOROOT" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOPATH="" GOROOT="" GOENV_SHELL=bash GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'zsh', it does not export GOPATH or GOROOT" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOPATH="" GOROOT="" GOENV_SHELL=zsh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'ksh', it does not export GOPATH or GOROOT" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOPATH="" GOROOT="" GOENV_SHELL=ksh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'fish', it does not export GOPATH or GOROOT" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOPATH="" GOROOT="" GOENV_SHELL=fish GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=1 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output ""
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash', it exports 'GOPATH'" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOROOT="" GOENV_SHELL=bash GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT

$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh', it exports 'GOPATH'" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOROOT="" GOENV_SHELL=ksh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT

$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh', it exports 'GOPATH'" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOROOT="" GOENV_SHELL=zsh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT

$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish', it exports 'GOPATH'" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOROOT="" GOENV_SHELL=fish GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=1 GOENV_DISABLE_GOPATH=0 PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT

$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash' and GOENV_GOPATH_PREFIX is empty, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=bash GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh' and GOENV_GOPATH_PREFIX is empty, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=zsh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh' and GOENV_GOPATH_PREFIX is empty, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=ksh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish' and GOENV_GOPATH_PREFIX is empty, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=fish GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
$HOME/go/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash' and GOENV_GOPATH_PREFIX is present, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=bash GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="/tmp/goenv/example" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
/tmp/goenv/example/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'ksh' and GOENV_GOPATH_PREFIX is present, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=ksh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="/tmp/goenv/example" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
/tmp/goenv/example/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'zsh' and GOENV_GOPATH_PREFIX is present, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=zsh GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="/tmp/goenv/example" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
/tmp/goenv/example/1.12.0
OUT
  assert_success
}

@test "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish' and GOENV_GOPATH_PREFIX is present, it exports 'GOROOT' and 'GOPATH'=\$HOME/go/<version>" {
  create_version "1.12.0"
  create_executable "1.12.0" "go-paths" <<SH
#!$BASH
echo \$GOROOT
echo \$GOPATH
SH

  GOENV_SHELL=fish GOENV_VERSION=1.12.0 GOENV_DISABLE_GOROOT=0 GOENV_DISABLE_GOPATH=0 GOENV_GOPATH_PREFIX="/tmp/goenv/example" PATH=${GOENV_TEST_DIR}:${PATH} run goenv-exec go-paths

  assert_output <<OUT
$GOENV_ROOT/versions/1.12.0
/tmp/goenv/example/1.12.0
OUT
  assert_success
}
