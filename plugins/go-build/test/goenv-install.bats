#!/usr/bin/env bats

project_root="${PWD%plugins/go-build}"
load "${project_root}test/test_helper.bash"
load test_helper_ext

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
1.11.0
1.11beta2
1.11beta3
1.11rc1
1.11rc2
1.11.1
1.11.2
1.11.3
1.11.4
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
#
#@test "carries original IFS within hooks" {
#  create_hook uninstall hello.bash <<SH
#hellos=(\$(printf "hello\\tugly world\\nagain"))
#echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
#exit
#SH
#
#  IFS=$' \t\n' run goenv-uninstall 1.1.1
#  remove_hook uninstall hello.bash
#  assert_success
#  assert_output "HELLO=:hello:ugly:world:again"
#}
#
#@test "{before,after}_uninstall hooks get triggered when version argument is already installed version and gets uninstalled" {
#  create_hook uninstall hello.bash <<SH
#before_uninstall 'echo before: \$PREFIX'
#after_uninstall 'echo after.'
#rm() {
#  echo "rm \$@"
#  command rm "\$@"
#}
#SH
#
#  stub goenv-hooks "uninstall : echo '$HOOK_PATH'/uninstall.bash"
#
#  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
#  run goenv-uninstall -f 1.2.3
#  remove_hook uninstall hello.bash
#
#  assert_success
#  assert_output <<-OUT
#before: ${GOENV_ROOT}/versions/1.2.3
#rm -rf ${GOENV_ROOT}/versions/1.2.3
#after.
#OUT
#
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#}
#
#@test "no hooks gets triggered when version argument is not already installed" {
#  create_hook uninstall hello.bash <<SH
#before_uninstall 'echo before: NOT_TRIGGERED'
#after_uninstall 'echo after: NOT_TRIGGERED'
#rm() {
#  echo "rm \$@"
#  command rm "\$@"
#}
#SH
#
#  stub goenv-hooks "uninstall : echo '$HOOK_PATH'/uninstall.bash"
#
#  run goenv-uninstall -f 1.2.3
#  remove_hook uninstall hello.bash
#
#  assert_failure
#  assert_output <<-OUT
#goenv: version '1.2.3' not installed
#OUT
#}
#
#@test "fails when '-f' argument and version argument is not already installed version" {
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#
#  run goenv-uninstall -f 1.2.3
#
#  assert_failure
#  assert_output <<-OUT
#goenv: version '1.2.3' not installed
#OUT
#}
#
#@test "fails when '--force' argument and version argument is not already installed version" {
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#
#  run goenv-uninstall --force 1.2.3
#
#  assert_failure
#  assert_output <<-OUT
#goenv: version '1.2.3' not installed
#OUT
#}
#
#@test "fails when when version argument is not already installed version" {
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#
#  run goenv-uninstall 1.2.3
#
#  assert_failure
#  assert_output <<-OUT
#goenv: version '1.2.3' not installed
#OUT
#}
#
#@test "fails when version argument is not already installed version" {
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#
#  run goenv-uninstall 1.2.3
#
#  assert_failure
#  assert_output <<-OUT
#goenv: version '1.2.3' not installed
#OUT
#}
#
#@test "shims get rehashed and version uninstalled when version argument is already installed version" {
#  assert [ ! -e "${GOENV_ROOT}/shims/gofmt" ]
#  create_executable "1.10.3" "gofmt"
#
#  # NOTE: Rehash to make shim present
#  run goenv-rehash
#  assert_success
#
#  assert [ -e "${GOENV_ROOT}/shims/gofmt" ]
#
#  run goenv-uninstall -f 1.10.3
#
#  assert_success
#  assert_output ''
#
#  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
#  assert [ ! -e "${GOENV_ROOT}/shims/gofmt" ]
#}

