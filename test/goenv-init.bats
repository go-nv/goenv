#!/usr/bin/env bats

load test_helper

@test "creates shims and versions directories" {
  assert [ ! -d "${GOENV_ROOT}/shims" ]
  assert [ ! -d "${GOENV_ROOT}/versions" ]
  run goenv-init -
  assert_success
  assert [ -d "${GOENV_ROOT}/shims" ]
  assert [ -d "${GOENV_ROOT}/versions" ]
}

@test "auto rehash" {
  run goenv-init -
  assert_success
  assert_line "command goenv rehash 2>/dev/null"
}

@test "setup shell completions" {
  root="$(cd $BATS_TEST_DIRNAME/.. && pwd)"
  run goenv-init - bash
  assert_success
  assert_line "source '${root}/test/../libexec/../completions/goenv.bash'"
}

@test "detect parent shell" {
  SHELL=/bin/false run goenv-init -
  assert_success
  assert_line "export GOENV_SHELL=bash"
}

@test "detect parent shell from script" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  cat > myscript.sh <<OUT
#!/bin/sh
eval "\$(goenv-init -)"
echo \$GOENV_SHELL
OUT
  chmod +x myscript.sh
  run ./myscript.sh /bin/zsh
  assert_success "sh"
}

@test "setup shell completions (fish)" {
  root="$(cd $BATS_TEST_DIRNAME/.. && pwd)"
  run goenv-init - fish
  assert_success
  assert_line "source '${root}/test/../libexec/../completions/goenv.fish'"
}

@test "fish instructions" {
  run goenv-init fish
  assert [ "$status" -eq 1 ]
  assert_line 'status --is-interactive; and . (goenv init -|psub)'
}

@test "option to skip rehash" {
  run goenv-init - --no-rehash
  assert_success
  refute_line "goenv rehash 2>/dev/null"
}

@test "adds shims to PATH" {
  export PATH="${BATS_TEST_DIRNAME}/../libexec:/usr/bin:/bin:/usr/local/bin"
  run goenv-init - bash
  assert_success
  assert_line 0 'export PATH="'${GOENV_ROOT}'/shims:${PATH}"'
}

@test "adds shims to PATH (fish)" {
  export PATH="${BATS_TEST_DIRNAME}/../libexec:/usr/bin:/bin:/usr/local/bin"
  run goenv-init - fish
  assert_success
  assert_line 0 "set -gx PATH '${GOENV_ROOT}/shims' \$PATH"
}

@test "can add shims to PATH more than once" {
  export PATH="${GOENV_ROOT}/shims:$PATH"
  run goenv-init - bash
  assert_success
  assert_line 0 'export PATH="'${GOENV_ROOT}'/shims:${PATH}"'
}

@test "can add shims to PATH more than once (fish)" {
  export PATH="${GOENV_ROOT}/shims:$PATH"
  run goenv-init - fish
  assert_success
  assert_line 0 "set -gx PATH '${GOENV_ROOT}/shims' \$PATH"
}

@test "outputs sh-compatible syntax" {
  run goenv-init - bash
  assert_success
  assert_line '  case "$command" in'

  run goenv-init - zsh
  assert_success
  assert_line '  case "$command" in'
}

@test "outputs fish-specific syntax (fish)" {
  run goenv-init - fish
  assert_success
  assert_line '  switch "$command"'
  refute_line '  case "$command" in'
}
