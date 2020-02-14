stub() {
  local program="$1"
  local prefix="$(echo "$program" | tr a-z- A-Z_)"
  shift

  export "${prefix}_STUB_PLAN"="${TMP}/${program}-stub-plan"
  export "${prefix}_STUB_RUN"="${TMP}/${program}-stub-run"
  export "${prefix}_STUB_END"=

  mkdir -p "${TMP}/bin"
  ln -sf "${BATS_TEST_DIRNAME}/stubs/stub" "${TMP}/bin/${program}"

  touch "${TMP}/${program}-stub-plan"
  for arg in "$@"; do
    printf "%s\n" "$arg" >> "${TMP}/${program}-stub-plan";
  done
}

unstub() {
  local program="$1"
  local prefix="$(echo "$program" | tr a-z- A-Z_)"
  local path="${TMP}/bin/${program}"

  export "${prefix}_STUB_END"=1

  local STATUS=0
  "$path" || STATUS="$?"

  rm -f "$path"
  rm -f "${TMP}/${program}-stub-plan" "${TMP}/${program}-stub-run"
  return "$STATUS"
}

run_inline_definition() {
  local definition="${TMP}/build-definition"
  cat > "$definition"
  run go-build "$definition" "${1:-$INSTALL_ROOT}"
}

install_fixture() {
  local args

  while [ "${1#-}" != "$1" ]; do
    args="$args $1"
    shift 1
  done

  local name="$1"
  local destination="$2"
  [ -n "$destination" ] || destination="$INSTALL_ROOT"

  run go-build $args "$FIXTURE_ROOT/$name" "$destination"
}

assert_output_contains() {
  if [ -n "$GOENV_DEBUG" ]; then
    echo "actual: $output"
    echo "'GOENV_DEBUG=1' detected. Test assertion with 'assert_output' will always fail. Re-run test without 'GOENV_DEBUG'"
    exit 1
  fi

  local expected
  if [ $# -eq 0 ]; then
    expected="$(cat -)"
  else
    expected="$1"
  fi

  echo "$output" | $(type -p ggrep grep | head -1) -F "$expected" >/dev/null || {
      echo "expected to contain: $expected"
      echo "actual: $output"
    } | flunk
}
