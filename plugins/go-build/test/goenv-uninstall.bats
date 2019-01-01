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
  run goenv-uninstall --complete
  assert_success
  assert_output <<'OUT'
--force
OUT
}

@test "prints full usage when '-h' is first argument given" {
  run goenv-uninstall -h
  assert_success
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "prints full usage when '--help' is first argument given" {
  run goenv-uninstall --help
  assert_success
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "fails and prints full usage when no arguments are given" {
  run goenv-uninstall
  assert_failure
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "fails and prints full usage when '-f' is given and no other arguments" {
  run goenv-uninstall -f
  assert_failure
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "fails and prints full usage when '--force' is given and no other arguments" {
  run goenv-uninstall --force
  assert_failure
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "fails and prints full usage when '-f' is given and '-' version argument" {
  run goenv-uninstall -f -
  assert_failure
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "fails and prints full usage when '--force' is given and '-' version argument" {
  run goenv-uninstall --force
  assert_failure
  assert_output <<'OUT'
Usage: goenv uninstall [-f|--force] <version>

   -f  Attempt to remove the specified version without prompting
       for confirmation. Still displays error message if version does not exist.

See `goenv versions` for a complete list of installed versions.
OUT
}

@test "carries original IFS within hooks" {
  create_hook uninstall hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
exit
SH

  IFS=$' \t\n' run goenv-uninstall 1.1.1
  remove_hook uninstall hello.bash
  assert_success
  assert_output "HELLO=:hello:ugly:world:again"
}

@test "{before,after}_uninstall hooks get triggered when version argument is already installed version and gets uninstalled" {
  create_hook uninstall hello.bash <<SH
before_uninstall 'echo before: \$PREFIX'
after_uninstall 'echo after.'
rm() {
  echo "rm \$@"
  command rm "\$@"
}
SH

  stub goenv-hooks "uninstall : echo '$HOOK_PATH'/uninstall.bash"

  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-uninstall -f 1.2.3
  remove_hook uninstall hello.bash

  assert_success
  assert_output <<-OUT
before: ${GOENV_ROOT}/versions/1.2.3
rm -rf ${GOENV_ROOT}/versions/1.2.3
after.
OUT

  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
}

@test "no hooks gets triggered when version argument is not already installed" {
  create_hook uninstall hello.bash <<SH
before_uninstall 'echo before: NOT_TRIGGERED'
after_uninstall 'echo after: NOT_TRIGGERED'
rm() {
  echo "rm \$@"
  command rm "\$@"
}
SH

  stub goenv-hooks "uninstall : echo '$HOOK_PATH'/uninstall.bash"

  run goenv-uninstall -f 1.2.3
  remove_hook uninstall hello.bash

  assert_failure
  assert_output <<-OUT
goenv: version '1.2.3' not installed
OUT
}

@test "fails when '-f' argument and version argument is not already installed version" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-uninstall -f 1.2.3

  assert_failure
  assert_output <<-OUT
goenv: version '1.2.3' not installed
OUT
}

@test "fails when '--force' argument and version argument is not already installed version" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-uninstall --force 1.2.3

  assert_failure
  assert_output <<-OUT
goenv: version '1.2.3' not installed
OUT
}

@test "fails when when version argument is not already installed version" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-uninstall 1.2.3

  assert_failure
  assert_output <<-OUT
goenv: version '1.2.3' not installed
OUT
}

@test "fails when version argument is not already installed version" {
  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]

  run goenv-uninstall 1.2.3

  assert_failure
  assert_output <<-OUT
goenv: version '1.2.3' not installed
OUT
}

@test "shims get rehashed and version uninstalled when version argument is already installed version" {
  assert [ ! -e "${GOENV_ROOT}/shims/gofmt" ]
  create_executable "1.10.3" "gofmt"

  # NOTE: Rehash to make shim present
  run goenv-rehash
  assert_success

  assert [ -e "${GOENV_ROOT}/shims/gofmt" ]

  run goenv-uninstall -f 1.10.3

  assert_success
  assert_output ''

  assert [ ! -d "${GOENV_ROOT}/versions/1.2.3" ]
  assert [ ! -e "${GOENV_ROOT}/shims/gofmt" ]
}

