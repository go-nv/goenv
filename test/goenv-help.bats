#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help help
  assert_success <<OUT
goenv help [--usage] COMMAND
OUT
}

@test "without args shows summary of common commands" {
  run goenv-help
  assert_success
  assert_output <<OUT
Usage: goenv <command> [<args>]

Some useful goenv commands are:
   commands    List all available commands of goenv
   local       Set or show the local application-specific Go version
   global      Set or show the global Go version
   shell       Set or show the shell-specific Go version
   rehash      Rehash goenv shims (run this after installing executables)
   version     Show the current Go version and its origin
   versions    List all Go versions available to goenv
   which       Display the full path to an executable
   whence      List all Go versions that contain the given executable

See 'goenv help <command>' for information on a specific command.
For full documentation, see: https://github.com/syndbg/goenv#readme
OUT
}

@test "fails when command argument does not exist" {
  run goenv-help hello
  assert_failure "goenv: no such command \`hello'"
}

@test "shows help for a specific command that exists" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  cat > "${GOENV_TEST_DIR}/bin/goenv-hello" <<SH
#!shebang
# Usage: goenv hello <world>
# Summary: Says "hello" to you, from goenv
# This command is useful for saying hello.
echo hello
SH

  run goenv-help hello
  assert_success
  assert_output <<SH
Usage: goenv hello <world>

This command is useful for saying hello.
SH
}

@test "replaces missing extended help with summary text for a specific command that exists" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  cat > "${GOENV_TEST_DIR}/bin/goenv-hello" <<SH
#!shebang
# Usage: goenv hello <world>
# Summary: Says "hello" to you, from goenv
echo hello
SH

  run goenv-help hello
  assert_success
  assert_output <<SH
Usage: goenv hello <world>

Says "hello" to you, from goenv
SH
}

@test "extracts only usage when '--usage' for a specific command that exists" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  cat > "${GOENV_TEST_DIR}/bin/goenv-hello" <<SH
#!shebang
# Usage: goenv hello <world>
# Summary: Says "hello" to you, from goenv
# This extended help won't be shown.
echo hello
SH

  run goenv-help --usage hello
  assert_success "Usage: goenv hello <world>"
}

@test "multiline usage section is returned when '--usage' for a specific command that exists" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  cat > "${GOENV_TEST_DIR}/bin/goenv-hello" <<SH
#!shebang
# Usage: goenv hello <world>
#        goenv hi [everybody]
#        goenv hola --translate
# Summary: Says "hello" to you, from goenv
# Help text.
echo hello
SH

  run goenv-help hello
  assert_success
  assert_output <<SH
Usage: goenv hello <world>
       goenv hi [everybody]
       goenv hola --translate

Help text.
SH
}

@test "multiline extended help section is returned for a specific command that exists" {
  mkdir -p "${GOENV_TEST_DIR}/bin"
  cat > "${GOENV_TEST_DIR}/bin/goenv-hello" <<SH
#!shebang
# Usage: goenv hello <world>
# Summary: Says "hello" to you, from goenv
# This is extended help text.
# It can contain multiple lines.
#
# And paragraphs.

echo hello
SH

  run goenv-help hello
  assert_success
  assert_output <<SH
Usage: goenv hello <world>

This is extended help text.
It can contain multiple lines.

And paragraphs.
SH
}
