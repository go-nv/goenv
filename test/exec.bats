#!/usr/bin/env bats

load test_helper

create_executable() {
  name="${1?}"
  shift 1
  bin="${GOENV_ROOT}/versions/${GOENV_VERSION}/bin"
  mkdir -p "$bin"
  { if [ $# -eq 0 ]; then cat -
    else echo "$@"
    fi
  } | sed -Ee '1s/^ +//' > "${bin}/$name"
  chmod +x "${bin}/$name"
}

@test "fails with invalid version" {
  export GOENV_VERSION="1.6.1"
  run goenv-exec go version
  assert_failure "goenv: version \`1.6.1' is not installed (set by GOENV_VERSION environment variable)"
}

@test "fails with invalid version set from file" {
  mkdir -p "$GOENV_TEST_DIR"
  cd "$GOENV_TEST_DIR"
  echo 1.6.1 > .go-version
  run goenv-exec go build
  assert_failure "goenv: version \`1.6.1' is not installed (set by $PWD/.go-version)"
}

@test "completes with names of executables" {
  export GOENV_VERSION="1.6.1"
  create_executable "go" "#!/bin/sh"

  goenv-rehash
  run goenv-completions exec
  assert_success
  assert_output <<OUT
--help
go
OUT
}

@test "carries original IFS within hooks" {
  create_hook exec hello.bash <<SH
hellos=(\$(printf "hello\\tugly world\\nagain"))
echo HELLO="\$(printf ":%s" "\${hellos[@]}")"
SH

  export GOENV_VERSION=system
  IFS=$' \t\n' run goenv-exec env
  assert_success
  assert_line "HELLO=:hello:ugly:world:again"
}

@test "forwards all arguments" {
  export GOENV_VERSION="1.6.1"
  create_executable "go" <<SH
#!$BASH
echo \$0
for arg; do
  # hack to avoid bash builtin echo which can't output '-e'
  printf "  %s\\n" "\$arg"
done
SH

  run goenv-exec go run "/path to/go script.go" -- extra args
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
