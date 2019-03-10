load ./test_assert_helpers

unset GOENV_VERSION
unset GOENV_DIR

# guard against executing this block twice due to bats internals
if [ -z "$GOENV_TEST_DIR" ]; then
  GOENV_TEST_DIR="${BATS_TMPDIR}/goenv"
  export GOENV_TEST_DIR="$(mktemp -d "${GOENV_TEST_DIR}.XXX" 2>/dev/null || echo "$GOENV_TEST_DIR")"

  if enable -f "${BATS_TEST_DIRNAME}"/../libexec/goenv-realpath.dylib realpath 2>/dev/null; then
    export GOENV_TEST_DIR="$(realpath "$GOENV_TEST_DIR")"
  else
    if [ -n "$GOENV_NATIVE_EXT" ]; then
      echo "goenv: failed to load 'realpath' builtin" >&2
      exit 1
    fi
  fi

  export GOENV_ROOT="${GOENV_TEST_DIR}/root"
  export ORIGINAL_HOME="${HOME}"
  export HOME="${GOENV_TEST_DIR}/home"
  export GOENV_HOOK_PATH="${GOENV_ROOT}/goenv.d"

  PATH=/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin
  PATH="${GOENV_TEST_DIR}/bin:$PATH"
  PATH="${BATS_TEST_DIRNAME}/../libexec:$PATH"
  PATH="${BATS_TEST_DIRNAME}/libexec:$PATH"
  PATH="${GOENV_ROOT}/shims:$PATH"
  export PATH

  for xdg_var in `env 2>/dev/null | grep ^XDG_ | cut -d= -f1`;
    do unset "$xdg_var";
  done
  unset xdg_var
fi

teardown() {
  rm -rf "$GOENV_TEST_DIR"
}

