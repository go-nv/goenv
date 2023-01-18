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
  export USE_FAKE_DEFINITIONS=true
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
1.0.0
1.2.0
1.2.2
1.3beta1
OUT
  unset USE_FAKE_DEFINITIONS
}

@test "prints full usage when '-h' is first argument given" {
  run goenv-install -h
  assert_success
  assert_output <<'OUT'
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
Usage: goenv install [-f] [-kvp] <version>|latest|unstable
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
  export USE_FAKE_DEFINITIONS=true
  run goenv-install -l

  assert_success
  assert_output <<-OUT
Available versions:
  1.0.0
  1.2.0
  1.2.2
  1.3beta1
OUT
  unset USE_FAKE_DEFINITIONS
}

@test "prints all available definitions when '--list' argument is given" {
  export USE_FAKE_DEFINITIONS=true
  run goenv-install --list

  assert_success
  assert_output <<-OUT
Available versions:
  1.0.0
  1.2.0
  1.2.2
  1.3beta1
OUT
  unset USE_FAKE_DEFINITIONS
}

@test "prints go-build version when '--version' argument is given" {
  run goenv-install --version

  assert_success
  assert_output <<-OUT
go-build 2.0.5
OUT
}

@test "installs the latest version when latest is given as an argument to install" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build

  LATEST_VERSION=1.2.2

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  export USE_FAKE_DEFINITIONS=true

  run goenv-install latest

  unset USE_FAKE_DEFINITIONS

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
Installing latest version ${LATEST_VERSION}...
Downloading ${LATEST_VERSION}.tar.gz...
-> http://localhost:8090/${LATEST_VERSION}/${LATEST_VERSION}.tar.gz
Installing Go Linux${arch}64bit ${LATEST_VERSION}...
Installed Go Linux${arch}64bit ${LATEST_VERSION} to ${GOENV_ROOT}/versions/${LATEST_VERSION}

OUT
    ;;
  Darwin*)
    assert_output <<-OUT
Installing latest version ${LATEST_VERSION}...
Downloading ${LATEST_VERSION}.tar.gz...
-> http://localhost:8090/${LATEST_VERSION}/${LATEST_VERSION}.tar.gz
Installing Go Darwin 10.8 64bit ${LATEST_VERSION}...
Installed Go Darwin 10.8 64bit ${LATEST_VERSION} to ${GOENV_ROOT}/versions/${LATEST_VERSION}

OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

  assert_success

  assert [ -f "${GOENV_ROOT}/versions/${LATEST_VERSION}/bin/go" ]
  run cat "${GOENV_ROOT}/versions/${LATEST_VERSION}/bin/go"
}

@test "installs the latest (including unstable) version when unstable is given as an argument to install" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build

  LATEST_VERSION=1.3beta1

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  export USE_FAKE_DEFINITIONS=true

  run goenv-install unstable

  unset USE_FAKE_DEFINITIONS

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
Installing latest (including unstable) version ${LATEST_VERSION}...
Downloading ${LATEST_VERSION}.tar.gz...
-> http://localhost:8090/${LATEST_VERSION}/${LATEST_VERSION}.tar.gz
Installing Go Linux${arch}64bit ${LATEST_VERSION}...
Installed Go Linux${arch}64bit ${LATEST_VERSION} to ${GOENV_ROOT}/versions/${LATEST_VERSION}

OUT
    ;;
  Darwin*)
    assert_output <<-OUT
Installing latest (including unstable) version ${LATEST_VERSION}...
Downloading ${LATEST_VERSION}.tar.gz...
-> http://localhost:8090/${LATEST_VERSION}/${LATEST_VERSION}.tar.gz
Installing Go Darwin 10.8 64bit ${LATEST_VERSION}...
Installed Go Darwin 10.8 64bit ${LATEST_VERSION} to ${GOENV_ROOT}/versions/${LATEST_VERSION}

OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

  assert_success

  assert [ -f "${GOENV_ROOT}/versions/${LATEST_VERSION}/bin/go" ]
  run cat "${GOENV_ROOT}/versions/${LATEST_VERSION}/bin/go"
}

@test "install does not silently fail when no available version to install" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build

  LATEST_VERSION=1.0.0

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  export USE_FAKE_DEFINITIONS=true

  run goenv-install ${LATEST_VERSION}

  unset USE_FAKE_DEFINITIONS

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  assert_output <<-OUT
No installable version found for $(uname -s) $(uname -m)

OUT

  assert_failure
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

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux${arch}64bit 1.2.2...
Installed Go Linux${arch}64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT
    ;;
  Darwin*)
    assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Darwin 10.8 64bit 1.2.2...
Installed Go Darwin 10.8 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

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

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux${arch}64bit 1.2.2...
Installed Go Linux${arch}64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT
    ;;
  Darwin*)
    assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Darwin 10.8 64bit 1.2.2...
Installed Go Darwin 10.8 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

after: 0
REHASHED
OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

  assert_success
}

@test "rehashes shims when '-f' argument and version argument is not already installed version and gets installed" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.2 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  run goenv-install -f 1.2.2

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux${arch}64bit 1.2.2...
Installed Go Linux${arch}64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

OUT
    ;;
  Darwin*)
    assert_output <<-OUT
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Darwin 10.8 64bit 1.2.2...
Installed Go Darwin 10.8 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

  assert_success

  assert [ -f "${GOENV_ROOT}/versions/1.2.2/bin/go" ]
  run cat "${GOENV_ROOT}/versions/1.2.2/bin/go"

  expected_binary_contents=$(cat test/http-definitions/1.2.2/go/bin/go)

  assert [ $output = $expected_binary_contents ]

  run "$(cat ${GOENV_ROOT}/shims/go)"

  # NOTE: Don't assert line 0 since bats modifies it
  assert_line 1 'set -e'
  assert_line 2 '[ -n "$GOENV_DEBUG" ] && set -x'
  assert_line 3 'program="${0##*/}"'
  assert_line 4 'if [[ "$program" = "go"* ]]; then'
  assert_line 5 '  for arg; do'
  assert_line 6 '    case "$arg" in'
  assert_line 7 '    -c* | -- ) break ;;'
  assert_line 8 '    */* )'
  assert_line 9 '      if [ -f "$arg" ]; then'
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

@test "installs latest patch version to definition when version argument is not already installed version and gets installed" {
  # NOTE: Create fake definition to install
  mkdir -p $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.0 $GOENV_ROOT/plugins/go-build/share/go-build
  cp $BATS_TEST_DIRNAME/fixtures/definitions/1.2.2 $GOENV_ROOT/plugins/go-build/share/go-build

  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"

  run goenv-install -f 1.2

  arch=" "
  if [ "$(uname -m)" = "aarch64" ]; then
    arch=" arm "
  fi

  unameOut="$(uname -s)"
  case "${unameOut}" in
  Linux*)
    assert_output <<-OUT
Using latest patch version 1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Linux${arch}64bit 1.2.2...
Installed Go Linux${arch}64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

OUT
    ;;
  Darwin*)
    assert_output <<-OUT
Using latest patch version 1.2.2
Downloading 1.2.2.tar.gz...
-> http://localhost:8090/1.2.2/1.2.2.tar.gz
Installing Go Darwin 10.8 64bit 1.2.2...
Installed Go Darwin 10.8 64bit 1.2.2 to ${GOENV_ROOT}/versions/1.2.2

OUT
    ;;
  *) machine="UNKNOWN:${unameOut}" ;;
  esac
  echo ${machine}

  assert_success

  assert [ -f "${GOENV_ROOT}/versions/1.2.2/bin/go" ]
  run cat "${GOENV_ROOT}/versions/1.2.2/bin/go"
}
