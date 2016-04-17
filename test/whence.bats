#!/usr/bin/env bats

load test_helper

create_executable() {
  local bin="${GOENV_ROOT}/versions/${1}/bin"
  mkdir -p "$bin"
  touch "${bin}/$2"
  chmod +x "${bin}/$2"
}

@test "finds versions where present" {
  create_executable "1.6.0" "go"
  create_executable "1.6.1" "go"

  run goenv-whence go
  assert_success
  assert_output <<OUT
1.6.0
1.6.1
OUT
}
