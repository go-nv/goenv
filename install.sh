#!/usr/bin/env bash
# goenv installer script for Linux/macOS
# Usage: curl -sfL https://raw.githubusercontent.com/go-nv/goenv/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GOENV_ROOT="${GOENV_ROOT:-$HOME/.goenv}"
GITHUB_REPO="go-nv/goenv"
INSTALL_DIR="$GOENV_ROOT/bin"

# API base URL. Overridable for GitHub Enterprise, mirrors, and for the
# hermetic release-selection tests in CI.
GITHUB_API="${GOENV_GITHUB_API:-https://api.github.com}"

# Detect OS and architecture
detect_platform() {
    # Declared separately so a failing command is not masked by 'local'
    # always returning 0 (shellcheck SC2155).
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    
    case "$os" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        freebsd*)
            OS="freebsd"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $os${NC}" >&2
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv7"
            ;;
        armv6l)
            ARCH="armv6"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $arch${NC}" >&2
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}Detected platform: ${OS}_${ARCH}${NC}"
}

# Fetch a URL. Prints the HTTP status on the first line, then the body.
#
# The status is returned through stdout rather than a global, because callers
# invoke this via command substitution — which runs the function in a subshell,
# so any variable it assigns is discarded when that subshell exits.
#
# Deliberately does NOT use curl -f. The GitHub API returns a JSON body
# explaining *why* a request failed (rate limit, bad token), and -f throws that
# body away — leaving only "exit code 22" to diagnose from.
#
# Sends an Authorization header when a token is available. Unauthenticated
# GitHub API access is limited to 60 requests/hour per IP, which CI runners
# routinely exhaust because their addresses are shared.
fetch_url() {
    local url="$1"
    local token="${GOENV_GITHUB_TOKEN:-${GITHUB_TOKEN:-${GH_TOKEN:-}}}"

    if command -v curl >/dev/null 2>&1; then
        local response status body
        # The token is passed as a header argument only; it is never echoed.
        if [ -n "$token" ]; then
            response=$(curl -sSL -H "Authorization: Bearer ${token}" \
                --write-out '\n%{http_code}' "$url" 2>/dev/null) || return 1
        else
            response=$(curl -sSL --write-out '\n%{http_code}' "$url" 2>/dev/null) || return 1
        fi
        status="${response##*$'\n'}"
        body="${response%$'\n'*}"
        printf '%s\n%s' "$status" "$body"
        return 0
    fi

    if command -v wget >/dev/null 2>&1; then
        local body
        # wget does not expose the status as simply; "000" means "unknown", and
        # the caller falls back to inspecting the body.
        if [ -n "$token" ]; then
            body=$(wget -qO- --header="Authorization: Bearer ${token}" "$url") || return 1
        else
            body=$(wget -qO- "$url") || return 1
        fi
        printf '000\n%s' "$body"
        return 0
    fi

    echo -e "${RED}Error: Neither curl nor wget found. Please install one of them.${NC}" >&2
    exit 1
}

# rate_limit_advice prints guidance for an HTTP 403/429 from the GitHub API.
rate_limit_advice() {
    echo -e "${RED}This is usually the GitHub API rate limit (60 requests/hour for unauthenticated${NC}" >&2
    echo -e "${RED}callers, counted per IP address — shared CI runners hit it routinely).${NC}" >&2
    echo -e "${RED}Set GITHUB_TOKEN (or GOENV_GITHUB_TOKEN) to raise the limit:${NC}" >&2
    echo -e "${RED}  GITHUB_TOKEN=\$YOUR_TOKEN curl -sfL .../install.sh | bash${NC}" >&2
    echo -e "${RED}In GitHub Actions: env: { GITHUB_TOKEN: \\\${{ secrets.GITHUB_TOKEN }} }${NC}" >&2
}

# Get the newest release that actually ships a binary for this platform.
#
# /releases/latest cannot be used: it returns the most recently published
# release across *all* branches, and the v2 maintenance branch publishes
# documentation-only releases that carry no binary assets at all. Installing
# from one of those fails with a 404 on the archive download (issue #582).
get_latest_version() {
    echo -e "${YELLOW}Fetching latest release...${NC}"

    local response releases status
    if ! response=$(fetch_url "${GITHUB_API}/repos/${GITHUB_REPO}/releases?per_page=50"); then
        echo -e "${RED}Could not reach the GitHub releases API.${NC}" >&2
        echo -e "${RED}Check your network connection, or set GOENV_GITHUB_API to a reachable mirror.${NC}" >&2
        exit 1
    fi

    # fetch_url returns "status\nbody"; a global would not survive the command
    # substitution above. When the body is empty the separator newline is the
    # last character and command substitution strips it, leaving just the
    # status — so the no-newline case has to be handled explicitly rather than
    # letting "${response#*\n}" fall through and return the status as the body.
    status="${response%%$'\n'*}"
    if [ "$status" = "$response" ]; then
        releases=""
    else
        releases="${response#*$'\n'}"
    fi

    case "${status}" in
        ""|000|2*) ;;
        403|429)
            echo -e "${RED}The GitHub releases API refused the request (HTTP ${status}).${NC}" >&2
            rate_limit_advice
            exit 1
            ;;
        *)
            echo -e "${RED}The GitHub releases API returned HTTP ${status}.${NC}" >&2
            echo -e "${RED}See https://github.com/${GITHUB_REPO}/releases to download manually.${NC}" >&2
            exit 1
            ;;
    esac

    if [ -z "$releases" ]; then
        echo -e "${RED}The GitHub releases API returned an empty response.${NC}" >&2
        exit 1
    fi

    # A rate-limit or error payload is a JSON object with a "message" field,
    # not the expected array. Diagnosing it here avoids reporting "no release
    # found for your platform", which sends people looking in the wrong place.
    case "$releases" in
        \[*) ;;
        *)
            local api_message
            api_message=$(printf '%s' "$releases" | tr -d '\n' |
                sed -n 's/.*"message" *: *"\([^"]*\)".*/\1/p')
            echo -e "${RED}The GitHub releases API returned an error.${NC}" >&2
            [ -n "$api_message" ] && echo -e "${RED}  ${api_message}${NC}" >&2
            case "$api_message" in
                *[Rr]ate*limit*) rate_limit_advice ;;
                *) echo -e "${RED}See https://github.com/${GITHUB_REPO}/releases to download manually.${NC}" >&2 ;;
            esac
            exit 1
            ;;
    esac

    # Collapse the response so each release occupies exactly one line, so the
    # fields of one release can be examined without pulling in a JSON parser
    # (jq is not guaranteed to be installed, least of all on Alpine).
    #
    # The assets key is canonicalised first so the split below works against
    # both compact and pretty-printed responses.
    local records
    records=$(printf '%s' "$releases" | tr -d '\n' | awk '{
        gsub(/"assets"[ \t]*:[ \t]*\[/, "\"assets\":[");
        gsub(/"tag_name"/, "\n\"tag_name\"");
        print
    }')

    # GitHub returns releases newest-first. Take the first stable release that
    # publishes the archive for this OS/arch.
    local record meta tag version
    while IFS= read -r record; do
        case "$record" in
            '"tag_name"'*) ;;
            *) continue ;;
        esac

        # Split the record at the assets array. Everything before it is release
        # metadata, which is where the draft/prerelease flags live.
        meta="${record%%\"assets\":\[*}"

        case "$meta" in
            *'"draft":true'* | *'"draft": true'*) continue ;;
        esac
        case "$meta" in
            *'"prerelease":true'* | *'"prerelease": true'*) continue ;;
        esac

        tag=$(printf '%s' "$meta" | sed -E 's/^"tag_name" *: *"([^"]+)".*/\1/')
        [ -n "$tag" ] || continue

        version="${tag#v}"

        # Require the archive to appear as an asset's "name" field, i.e. in JSON
        # key/value syntax. Matching the bare filename would also match release
        # notes, which routinely quote download filenames — and selecting a
        # release from its prose produces a 404 on download, the exact failure
        # issue #582 was filed for.
        #
        # The assets array is not delimited by hand here: asset objects contain
        # nested "]" characters, so cutting at the first one silently drops most
        # of the assets.
        case "$record" in
            *"\"name\":\"goenv_${version}_${OS}_${ARCH}.tar.gz\""* | \
            *"\"name\": \"goenv_${version}_${OS}_${ARCH}.tar.gz\""*)
                LATEST_VERSION="$tag"
                break
                ;;
        esac
    done <<EOF
$records
EOF

    if [ -z "$LATEST_VERSION" ]; then
        echo -e "${RED}No release found containing a ${OS}_${ARCH} binary (goenv_<version>_${OS}_${ARCH}.tar.gz)${NC}" >&2
        echo -e "${RED}See https://github.com/${GITHUB_REPO}/releases for available downloads.${NC}" >&2
        exit 1
    fi

    echo -e "${GREEN}Latest version: ${LATEST_VERSION}${NC}"
}

# Download and install binary
#
# The archive is fetched from github.com (not api.github.com), which serves
# public release assets without authentication and is not subject to the API's
# 60/hour limit. No token is sent here on purpose: this URL redirects to a
# separate CDN host, and curl -L would forward an Authorization header across
# that redirect, handing the token to a different origin.
install_binary() {
    local version="${LATEST_VERSION#v}"  # Remove 'v' prefix if present
    local archive_name="goenv_${version}_${OS}_${ARCH}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${archive_name}"
    local tmp_dir
    tmp_dir=$(mktemp -d)
    
    echo -e "${YELLOW}Downloading goenv...${NC}"
    echo "URL: $download_url"
    
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sfL "$download_url" -o "$tmp_dir/$archive_name"; then
            echo -e "${RED}Failed to download goenv${NC}" >&2
            rm -rf "$tmp_dir"
            exit 1
        fi
    else
        if ! wget -q "$download_url" -O "$tmp_dir/$archive_name"; then
            echo -e "${RED}Failed to download goenv${NC}" >&2
            rm -rf "$tmp_dir"
            exit 1
        fi
    fi
    
    echo -e "${YELLOW}Extracting archive...${NC}"
    tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    
    echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
    mkdir -p "$INSTALL_DIR"
    
    # Copy binary
    cp "$tmp_dir/goenv" "$INSTALL_DIR/goenv"
    chmod +x "$INSTALL_DIR/goenv"
    
    # Copy completions if they exist
    if [ -d "$tmp_dir/completions" ]; then
        mkdir -p "$GOENV_ROOT/completions"
        cp -r "$tmp_dir/completions/"* "$GOENV_ROOT/completions/" 2>/dev/null || true
    fi
    
    # Cleanup
    rm -rf "$tmp_dir"
    
    # Remove stale goenv shim from v2 installations.
	# v2's goenv-rehash bakes the Cellar/libexec path into shims at creation time
	# (e.g. exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv"). After
	# upgrading to v3, the old shim shadows the real v3 binary. We only remove
	# it if it contains "libexec/goenv" or "libexec\goenv" — the v2 fingerprint.
	if [ -f "$GOENV_ROOT/shims/goenv" ] && grep -qE 'libexec[/\\]goenv' "$GOENV_ROOT/shims/goenv" 2>/dev/null; then
		echo -e "${YELLOW}Removing stale v2 goenv shim...${NC}"
		if rm -f "$GOENV_ROOT/shims/goenv" 2>/dev/null; then
			echo -e "${GREEN}✓ Stale shim removed${NC}"
		else
			echo -e "${YELLOW}⚠ Warning: Failed to remove stale v2 goenv shim${NC}"
		fi
  fi
    
    echo -e "${GREEN}✓ goenv installed successfully!${NC}"
}

# Auto-configure shell profile
setup_shell_profile() {
    local shell_config
    
    # Detect shell config file
    if [ -n "$BASH_VERSION" ]; then
        if [ -f "$HOME/.bash_profile" ]; then
            shell_config="$HOME/.bash_profile"
        else
            shell_config="$HOME/.bashrc"
        fi
    elif [ -n "$ZSH_VERSION" ]; then
        shell_config="$HOME/.zshrc"
    else
        shell_config="$HOME/.profile"
    fi
    
    # Check if goenv is already configured
    if [ -f "$shell_config" ] && grep -q "goenv init" "$shell_config"; then
        echo -e "${GREEN}✓ goenv is already configured in ${shell_config}${NC}"
        return 0
    fi
    
    # Add goenv configuration with markers
    echo -e "${YELLOW}Adding goenv configuration to ${shell_config}...${NC}"
    
    cat >> "$shell_config" << 'EOF'

# goenv - Go version manager (auto-configured by installer)
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
EOF
    
    echo -e "${GREEN}✓ Shell profile configured successfully!${NC}"
}

# Print setup completion message
print_instructions() {
    local shell_config
    
    # Detect shell
    if [ -n "$BASH_VERSION" ]; then
        if [ -f "$HOME/.bash_profile" ]; then
            shell_config="$HOME/.bash_profile"
        else
            shell_config="$HOME/.bashrc"
        fi
    elif [ -n "$ZSH_VERSION" ]; then
        shell_config="$HOME/.zshrc"
    else
        shell_config="$HOME/.profile"
    fi
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Installation complete!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${YELLOW}To start using goenv, reload your shell:${NC}"
    echo "  exec \$SHELL"
    echo ""
    echo -e "${YELLOW}Or source your config now:${NC}"
    echo "  source ${shell_config}"
    echo ""
    echo -e "${YELLOW}Quick start:${NC}"
    echo "  goenv install 1.22.0     # Install Go 1.22.0"
    echo "  goenv global 1.22.0      # Set as default"
    echo "  goenv versions           # List installed versions"
    echo ""
    echo -e "${YELLOW}Enable tab completion (optional):${NC}"
    echo "  goenv completion --install"
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main installation flow
main() {
    echo -e "${GREEN}goenv installer${NC}"
    echo ""
    
    detect_platform
    get_latest_version
    install_binary
    setup_shell_profile
    print_instructions
}

# Run the installer, unless this script was sourced (which the CI tests do so
# they can call individual functions). Using ${BASH_SOURCE[0]:-$0} keeps the
# "curl ... | bash" path working, where BASH_SOURCE is unset and $0 is "bash".
if [ "${BASH_SOURCE[0]:-$0}" = "$0" ]; then
    main "$@"
fi
