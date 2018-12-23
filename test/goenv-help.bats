#!/usr/bin/env bats

load test_helper

@test "without args shows summary of common commands" {
  run goenv-help
  assert_success
  assert_line "Usage: goenv <command> [<args>]"
  assert_line "Some useful goenv commands are:"
}

@test "invalid command" {
  run goenv-help hello
  assert_failure "goenv: no such command \`hello'"
}

@test "shows help for a specific command" {
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

@test "replaces missing extended help with summary text" {
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

@test "extracts only usage" {
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

@test "multiline usage section" {
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

@test "multiline extended help section" {
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
