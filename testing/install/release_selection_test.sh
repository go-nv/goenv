#!/usr/bin/env bash
# Hermetic regression tests for install.sh's release selection.
#
# Reproduces issue #582: the GitHub /releases/latest endpoint returns the most
# recently published release across *all* branches. The v2 maintenance branch
# publishes documentation-only releases with no binary assets, so installing
# from "latest" 404s on the archive download. The installer must instead pick
# the newest release that actually publishes an archive for the target platform.
#
# Runs offline against testing/fixtures/github-releases.json.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE="${REPO_ROOT}/testing/fixtures/github-releases.json"

FAILURES=0
SERVER_PID=""

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    [ -n "${SERVE_DIR:-}" ] && rm -rf "$SERVE_DIR"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    FAILURES=$((FAILURES + 1))
}

pass() {
    echo "ok: $*"
}

# Serve the fixture at <root>/repos/go-nv/goenv/releases so the installer's
# real URL construction is exercised, not a stubbed-out fetch.
start_fixture_server() {
    SERVE_DIR="$(mktemp -d)"
    mkdir -p "${SERVE_DIR}/repos/go-nv/goenv"
    # The installer requests ".../releases?per_page=50"; python's http.server
    # ignores the query string and serves the "releases" file.
    cp "$FIXTURE" "${SERVE_DIR}/repos/go-nv/goenv/releases"

    local python_bin
    python_bin="$(command -v python3 || command -v python)"
    if [ -z "$python_bin" ]; then
        echo "python3 is required for these tests" >&2
        exit 1
    fi

    # -u keeps the "Serving HTTP on ... port N" line unbuffered so we can read
    # the port the OS picked.
    "$python_bin" -u -m http.server 0 --bind 127.0.0.1 --directory "$SERVE_DIR" \
        >"${SERVE_DIR}/server.log" 2>&1 &
    SERVER_PID=$!

    # Wait for the port to be reported.
    local port=""
    for _ in $(seq 1 50); do
        port="$(sed -nE 's/.*port ([0-9]+).*/\1/p' "${SERVE_DIR}/server.log" | head -1)"
        [ -n "$port" ] && break
        sleep 0.1
    done

    if [ -z "$port" ]; then
        echo "fixture server failed to start:" >&2
        cat "${SERVE_DIR}/server.log" >&2
        exit 1
    fi

    FIXTURE_URL="http://127.0.0.1:${port}"
}

# select_version <os> <arch> -> prints the tag the installer would use
select_version() {
    local os="$1" arch="$2"
    (
        # Sourcing install.sh defines its functions without running main.
        # shellcheck disable=SC1091
        GOENV_GITHUB_API="$FIXTURE_URL" source "${REPO_ROOT}/install.sh"
        OS="$os"
        ARCH="$arch"
        LATEST_VERSION=""
        get_latest_version >/dev/null 2>&1
        echo "$LATEST_VERSION"
    )
}

assert_selects() {
    local os="$1" arch="$2" want="$3"
    local got
    got="$(select_version "$os" "$arch")"

    if [ "$got" = "$want" ]; then
        pass "${os}_${arch} selects ${want}"
    else
        fail "${os}_${arch}: expected '${want}', got '${got}'"
    fi
}

start_fixture_server

echo "== issue #582: must skip the newest release when it has no binaries =="
# 2.2.43 is newest but publishes nothing; 3.2.0-rc1 has assets but is a
# prerelease. 3.1.4 is the newest usable release.
assert_selects linux amd64 "3.1.4"
assert_selects darwin arm64 "3.1.4"
assert_selects linux arm64 "3.1.4"

echo
echo "== falls through to an older release when this platform is missing =="
# No release publishes linux/riscv64, so nothing should be selected.
got="$(select_version linux riscv64)"
if [ -z "$got" ]; then
    pass "unsupported platform selects nothing instead of a broken release"
else
    fail "unsupported platform should select nothing, got '${got}'"
fi

echo
echo "== sourcing install.sh must not run the installer =="
# If the source guard regressed, sourcing would attempt a real install.
probe_root="$(mktemp -d)"
(
    # shellcheck disable=SC1091
    GOENV_ROOT="$probe_root" source "${REPO_ROOT}/install.sh"
) >/dev/null 2>&1
if [ -e "${probe_root}/bin/goenv" ]; then
    fail "sourcing install.sh installed a binary; the source guard is broken"
else
    pass "sourcing is side-effect free"
fi
rm -rf "$probe_root"

echo
echo "== release notes must never be parsed as release metadata =="
# The 2.2.42 fixture publishes no usable binary but its body quotes
# goenv_2.2.42_linux_amd64.tar.gz. Selecting it would produce a 404 on download
# — the exact failure issue #582 was filed for.
got="$(select_version linux amd64)"
if [ "$got" = "2.2.42" ]; then
    fail "selected 2.2.42 by matching a filename quoted in its release notes"
else
    pass "asset names in prose do not select a release (got '${got}')"
fi

echo
echo "== API errors must be diagnosed, not reported as 'no binary for your platform' =="

# run_installer_against <served-file-content> <http-ish scenario name>
# Serves a payload that is not a release array and captures the error output.
check_api_error() {
    local label="$1" payload="$2" expect="$3"
    local dir output
    dir="$(mktemp -d)"
    mkdir -p "${dir}/repos/go-nv/goenv"
    printf '%s' "$payload" > "${dir}/repos/go-nv/goenv/releases"

    local python_bin port=""
    python_bin="$(command -v python3 || command -v python)"
    "$python_bin" -u -m http.server 0 --bind 127.0.0.1 --directory "$dir" \
        >"${dir}/server.log" 2>&1 &
    local pid=$!

    for _ in $(seq 1 50); do
        port="$(sed -nE 's/.*port ([0-9]+).*/\1/p' "${dir}/server.log" | head -1)"
        [ -n "$port" ] && break
        sleep 0.1
    done

    output=$(
        (
            # shellcheck disable=SC1091
            GOENV_GITHUB_API="http://127.0.0.1:${port}" source "${REPO_ROOT}/install.sh"
            OS=linux
            ARCH=amd64
            LATEST_VERSION=""
            get_latest_version
        ) 2>&1
    ) || true

    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    rm -rf "$dir"

    if printf '%s' "$output" | grep -q "$expect"; then
        pass "${label}: reported as an API problem"
    else
        fail "${label}: expected output matching '${expect}', got: ${output}"
    fi
}

check_api_error "rate limit (403 body)" \
    '{"message":"API rate limit exceeded for 1.2.3.4.","documentation_url":"https://docs.github.com/"}' \
    "rate limit"

check_api_error "empty response" \
    '' \
    "empty response"

echo
echo "== the fixture must match the shape the real API returns =="
# The fixture is only useful if it exercises the same parsing path as a real
# response. An earlier fixture used compact "assets":[{...}] with no nested
# structure, so the parser passed here while failing against the live API, whose
# asset objects are pretty-printed and contain nested "]" characters.
if grep -q '"assets": \[' "$FIXTURE" && grep -q '"name": "goenv_' "$FIXTURE"; then
    pass "fixture uses the pretty-printed key spacing GitHub actually returns"
else
    fail "fixture does not match real API formatting; parser bugs will not be caught"
fi
if grep -q '"uploader"' "$FIXTURE"; then
    pass "fixture includes nested asset objects"
else
    fail "fixture asset objects lack nesting; the real API nests an uploader object containing ']' characters, which is what broke naive array delimiting"
fi

echo
if [ "$FAILURES" -ne 0 ]; then
    echo "${FAILURES} test(s) failed" >&2
    exit 1
fi
echo "All install.sh release-selection tests passed"
