load ./test_assert_helpers
load ./test_assert_helpers_ext

export TMP="$BATS_TEST_DIRNAME/tmp"

unset GOENV_VERSION
unset GOENV_DIR

# guard against executing this block twice due to bats internals
if [ -z "$GOENV_TEST_DIR" ]; then
  GOENV_TEST_DIR="${BATS_TMPDIR}/goenv"
  export GOENV_TEST_DIR="$(mktemp -d "${GOENV_TEST_DIR}.XXX" 2>/dev/null || echo "$GOENV_TEST_DIR")"

  export GOENV_ROOT="${GOENV_TEST_DIR}/root"
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

if [ "$FIXTURE_ROOT" != "$BATS_TEST_DIRNAME/fixtures" ]; then
  export FIXTURE_ROOT="$BATS_TEST_DIRNAME/fixtures"
  export INSTALL_ROOT="$TMP/install"
  PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
  PATH="$TMP/bin:$PATH"
  export PATH
fi

teardown() {
  rm -rf "$GOENV_TEST_DIR"
  rm -fr "${TMP:?}"/*
}

