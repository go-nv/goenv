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
if [ "$FAILURES" -ne 0 ]; then
    echo "${FAILURES} test(s) failed" >&2
    exit 1
fi
echo "All install.sh release-selection tests passed"
