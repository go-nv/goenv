#!/usr/bin/env bats

load test_helper

setup() {
  export GOENV_ROOT="${TMP}/goenv"
  export HOOK_PATH="${TMP}/i has hooks"
  mkdir -p "$HOOK_PATH"
}

@test "goenv-install hooks" {
  cat > "${HOOK_PATH}/install.bash" <<OUT
before_install 'echo before: \$PREFIX'
after_install 'echo after: \$STATUS'
OUT
  stub goenv-hooks "install : echo '$HOOK_PATH'/install.bash"
  stub goenv-rehash "echo rehashed"

  definition="${TMP}/3.2.1"
  cat > "$definition" <<<"echo go-build"
  run goenv-install "$definition"

  assert_success
  assert_output <<-OUT
before: ${GOENV_ROOT}/versions/3.2.1
go-build
after: 0
rehashed
OUT
}

