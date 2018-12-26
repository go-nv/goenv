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

flunk() {
  {
    if [ "$#" -eq 0 ]; then cat -
      else echo "$@"
    fi
  } | sed "s:${GOENV_TEST_DIR}:TEST_DIR:g" >&2
  return 1
}

assert_success() {
  if [ "$status" -ne 0 ]; then
    flunk "command failed with exit status $status"
  elif [ "$#" -gt 0 ]; then
    assert_output "$1"
  fi
}

assert_failure() {
  if [ "$status" -eq 0 ]; then
    flunk "expected failed exit status"
  elif [ "$#" -gt 0 ]; then
    assert_output "$1"
  fi
}

assert_equal() {
  if [ "$1" != "$2" ]; then
    {
      echo "expected: $1"
      echo "actual: $2"
    } | flunk
  fi
}

assert_contains() {
  if [ "$1" == *"$2"* ]; then
    {
      echo "expected to contain: $1"
      echo "actual: $2"
    } | flunk
  fi
}

assert_output() {
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

  assert_equal "$expected" "$output"
}

assert_line() {
  if [ "$1" -ge 0 ] 2>/dev/null; then
    assert_equal "$2" "${lines[$1]}"
  else
    local line
    for line in "${lines[@]}"; do
      if [ "$line" = "$1" ]; then
        return 0;
      fi
    done
    flunk "expected line \`$1'"
  fi
}

refute_line() {
  if [ "$1" -ge 0 ] 2>/dev/null; then
    local num_lines="${#lines[@]}"
    if [ "$1" -lt "$num_lines" ]; then
      flunk "output has $num_lines lines"
    fi
  else
    local line
    for line in "${lines[@]}"; do
      if [ "$line" = "$1" ]; then
        flunk "expected to not find line \`$line'"
      fi
    done
  fi
}

assert() {
  if ! "$@"; then
    flunk "failed: $@"
  fi
}

# Output a modified PATH that ensures that the given executable is not present,
# but in which system utils necessary for goenv operation are still available.
path_without() {
  local exe="$1"
  local path=":${PATH}:"
  local found alt util
  for found in $(which -a "$exe"); do
    found="${found%/*}"
    if [ "$found" != "${GOENV_ROOT}/shims" ]; then
      alt="${GOENV_TEST_DIR}/$(echo "${found#/}" | tr '/' '-')"
      mkdir -p "$alt"
      for util in bash head cut readlink greadlink; do
        if [ -x "${found}/$util" ]; then
          ln -s "${found}/$util" "${alt}/$util"
        fi
      done
      path="${path/:${found}:/:${alt}:}"
    fi
  done
  path="${path#:}"
  echo "${path%:}"
}

create_hook() {
  mkdir -p "${GOENV_HOOK_PATH}/$1"
  touch "${GOENV_HOOK_PATH}/$1/$2"
  if [ ! -t 0 ]; then
    cat > "${GOENV_HOOK_PATH}/$1/$2"
  fi
}

create_executable() {
  goenv_version="${1?}"
  name="${2?}"
  shift 1
  shift 1

  bin="${GOENV_ROOT}/versions/${goenv_version}/bin"

  mkdir -p "$bin"
  {
    if [ $# -eq 0 ]; then
      echo ''
    else
      echo "$@"
    fi
  } | sed -Ee '1s/^ +//' > "${bin}/${name}"

  chmod +x "${bin}/${name}"
}

create_version() {
  mkdir -p "${GOENV_ROOT}/versions/$1"
}

create_file() {
  mkdir -p "$(dirname "$1")"
  touch "$1"
}

