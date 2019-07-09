#!/usr/bin/env bats

project_root="${PWD%plugins/go-build}"
load test_helper

export PATH="${project_root}libexec:$PATH"

@test "has usage instructions" {
  run goenv-help --usage uninstall
  assert_success
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>
OUT
}

@test "has completion support" {
  run goenv-install --complete
  assert_success
  assert_output <<'OUT'
--list
--force
--skip-existing
--keep
--patch
--verbose
--version
--debug
1.2.2
1.3.0
1.3.1
1.3.2
1.3.3
1.4.0
1.4.1
1.4.2
1.4.3
1.5.0
1.5.1
1.5.2
1.5.3
1.5.4
1.6.0
1.6.1
1.6.2
1.6.3
1.6.4
1.7.0
1.7.1
1.7.3
1.7.4
1.7.5
1.8.0
1.8.1
1.8.3
1.8.4
1.8.5
1.8.7
1.9.0
1.9.1
1.9.2
1.9.3
1.9.4
1.9.5
1.9.6
1.9.7
1.10.0
1.10beta2
1.10rc1
1.10rc2
1.10.1
1.10.2
1.10.3
1.10.4
1.10.5
1.10.6
1.10.7
1.10.8
1.11.0
1.11beta2
1.11beta3
1.11rc1
1.11rc2
1.11.1
1.11.2
1.11.3
1.11.4
1.11.5
1.11.6
1.11.7
1.11.8
1.11.9
1.11.10
1.11.11
1.11.12
1.12.0
1.12beta1
1.12beta2
1.12rc1
1.12.1
1.12.2
1.12.3
1.12.4
1.12.5
1.12.6
1.12.7
1.13beta1
OUT
}

@test "prints full usage when '-h' is first argument given" {
  run goenv-install -h
  assert_success
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "prints full usage when '--help' is first argument given" {
  run goenv-install --help
  assert_success
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails and prints full usage when no arguments are given" {
  run goenv-install
  assert_failure
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails and prints full usage when '-f' is given and no other arguments" {
  run goenv-install -f
  assert_failure
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails and prints full usage when '--force' is given and no other arguments" {
  run goenv-install --force
  assert_failure
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails and prints full usage when '-f' is given and '-' version argument" {
  run goenv-install -f -
  assert_failure
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails and prints full usage when '--force' is given and '-' version argument" {
  run goenv-install --force
  assert_failure
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>
       goenv install [-f] [-kvp] <definition-file>
       goenv install -l|--list
       goenv install --version

  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "carries original IFS within hooks" {
  create_hook install hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
exit
SH

  IFS=$' \t\n' run goenv-install 1.1.1
  remove_hook install hello.bash
  assert_success
  assert_output "HELLO=:hello:ugly:world:again"
}

@test "fails when '-f' argument and version argument is not a known version definition" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-install -f 1.2.3

  assert_failure
  assert_output <<-OUT
go-build: definition not found: 1.2.3

See all available versions with 'goenv install --list'.

If the version you need is missing, try upgrading goenv:

  cd ${BATS_TEST_DIRNAME}/../../.. && git pull && cd -
OUT
}

@test "fails when '--force' argument and version argument is not a known version definition" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-install --force 1.2.3

  assert_failure
  assert_output <<-OUT
go-build: definition not found: 1.2.3

See all available versions with 'goenv install --list'.

If the version you need is missing, try upgrading goenv:

  cd ${BATS_TEST_DIRNAME}/../../.. && git pull && cd -
OUT
}

@test "prints all available definitions when '-l' argument is given" {
  run goenv-install -l

  assert_success
  assert_output <<-OUT
Available versions:
  1.2.2
  1.3.0
  1.3.1
  1.3.2
  1.3.3
  1.4.0
  1.4.1
  1.4.2
  1.4.3
  1.5.0
  1.5.1
  1.5.2
  1.5.3
  1.5.4
  1.6.0
  1.6.1
  1.6.2
  1.6.3
  1.6.4
  1.7.0
  1.7.1
  1.7.3
  1.7.4
  1.7.5
  1.8.0
  1.8.1
  1.8.3
  1.8.4
  1.8.5
  1.8.7
  1.9.0
  1.9.1
  1.9.2
  1.9.3
  1.9.4
  1.9.5
  1.9.6
  1.9.7
  1.10.0
  1.10beta2
  1.10rc1
  1.10rc2
  1.10.1
  1.10.2
  1.10.3
  1.10.4
  1.10.5
  1.10.6
  1.10.7
  1.10.8
  1.11.0
  1.11beta2
  1.11beta3
  1.11rc1
  1.11rc2
  1.11.1
  1.11.2
  1.11.3
  1.11.4
  1.11.5
  1.11.6
  1.11.7
  1.11.8
  1.11.9
  1.11.10
  1.11.11
  1.11.12
  1.12.0
  1.12beta1
  1.12beta2
  1.12rc1
  1.12.1
  1.12.2
  1.12.3
  1.12.4
  1.12.5
  1.12.6
  1.12.7
  1.13beta1
OUT
}

@test "prints all available definitions when '--list' argument is given" {
  run goenv-install --list

  assert_success
  assert_output <<-OUT
Available versions:
  1.2.2
  1.3.0
  1.3.1
  1.3.2
  1.3.3
  1.4.0
  1.4.1
  1.4.2
  1.4.3
  1.5.0
  1.5.1
  1.5.2
  1.5.3
  1.5.4
  1.6.0
  1.6.1
  1.6.2
  1.6.3
  1.6.4
  1.7.0
  1.7.1
  1.7.3
  1.7.4
  1.7.5
  1.8.0
  1.8.1
  1.8.3
  1.8.4
  1.8.5
  1.8.7
  1.9.0
  1.9.1
  1.9.2
  1.9.3
  1.9.4
  1.9.5
  1.9.6
  1.9.7
  1.10.0
  1.10beta2
  1.10rc1
  1.10rc2
  1.10.1
  1.10.2
  1.10.3
  1.10.4
  1.10.5
  1.10.6
  1.10.7
  1.10.8
  1.11.0
  1.11beta2
  1.11beta3
  1.11rc1
  1.11rc2
  1.11.1
  1.11.2
  1.11.3
  1.11.4
  1.11.5
  1.11.6
  1.11.7
  1.11.8
  1.11.9
  1.11.10
  1.11.11
  1.11.12
  1.12.0
  1.12beta1
  1.12beta2
  1.12rc1
  1.12.1
  1.12.2
  1.12.3
  1.12.4
  1.12.5
  1.12.6
  1.12.7
  1.13beta1
OUT
}

@test "prints go-build version when '--version' argument is given" {
  run goenv-install --version

  assert_success
  assert_output <<-OUT
go-build 2.0.0beta1
OUT
}

@test "{before,after}_install hooks get triggered when '--force' argument and version argument is not already installed version and gets installed" {
  create_hook install hello.bash <<SH
before_install 'echo before: \$PREFIX'
after_install 'echo after: \$STATUS'
SH

  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.2 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  goenv-rehash() {
    echo "REHASHED"
  }

  export -f goenv-rehash

  run goenv-install --force 1.2.2

  rm -rf $GOENV_ROOT

  assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux 64bit 1.2.2...
Installed Go Linux 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT

  assert_success
}

@test "{before,after}_install hooks get triggered when '-f' argument and version argument is not already installed version and gets installed" {
  create_hook install hello.bash <<SH
before_install 'echo before: \$PREFIX'
after_install 'echo after: \$STATUS'
SH

  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.2 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  goenv-rehash() {
    echo "REHASHED"
  }

  export -f goenv-rehash

  run goenv-install -f 1.2.2

  rm -rf $GOENV_ROOT

  assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux 64bit 1.2.2...
Installed Go Linux 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT

  assert_success
}

@test "rehashes shims when '-f' argument and version argument is not already installed version and gets installed" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.2 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  run goenv-install -f 1.2.2

  assert_output <<-OUT
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux 64bit 1.2.2...
Installed Go Linux 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

OUT
  assert_success

  assert [ -f "${GOENV_ROOT}/versions/1.2.2/bin/go" ]
  run cat "${GOENV_ROOT}/versions/1.2.2/bin/go"

  expected_binary_contents=`cat test/http-definitions/1.2.2/go/bin/go`

  assert [ $output = $expected_binary_contents ]

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


@test "adds patch version '0' to definition when version argument is not already installed version and gets installed" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.0 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  run goenv-install -f 1.2

  assert_output <<-OUT
Adding patch version 0 to 1.2
Downloading 1.2.0.tar.gz...
-> http://localhost:8090/1.2.0/1.2.0.tar.gz
Installing Go Linux 64bit 1.2.0...
Installed Go Linux 64bit 1.2.0 to ${GOENV_ROOT}/versions/1.2.0

OUT
  assert_success

  assert [ -f "${GOENV_ROOT}/versions/1.2.0/bin/go" ]
  run cat "${GOENV_ROOT}/versions/1.2.0/bin/go"
}
