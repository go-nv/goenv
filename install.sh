#!/usr/bin/env bash
# goenv installer script for Linux/macOS
# Usage: curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

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

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
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

# Get latest release version
get_latest_version() {
    echo -e "${YELLOW}Fetching latest release...${NC}"
    
    if command -v curl >/dev/null 2>&1; then
        LATEST_VERSION=$(curl -sL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        LATEST_VERSION=$(wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        echo -e "${RED}Error: Neither curl nor wget found. Please install one of them.${NC}" >&2
        exit 1
    fi
    
    if [ -z "$LATEST_VERSION" ]; then
        echo -e "${RED}Failed to fetch latest version${NC}" >&2
        exit 1
    fi
    
    echo -e "${GREEN}Latest version: ${LATEST_VERSION}${NC}"
}

# Download and install binary
install_binary() {
    local version="${LATEST_VERSION#v}"  # Remove 'v' prefix if present
    local archive_name="goenv_${version}_${OS}_${ARCH}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${archive_name}"
    local tmp_dir=$(mktemp -d)
    
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
    
    echo -e "${GREEN}✓ goenv installed successfully!${NC}"
}

# Print setup instructions
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
    echo -e "${YELLOW}Add the following to your shell configuration:${NC}"
    echo -e "  ${shell_config}"
    echo ""
    echo '  export GOENV_ROOT="$HOME/.goenv"'
    echo '  export PATH="$GOENV_ROOT/bin:$PATH"'
    echo '  eval "$(goenv init -)"'
    echo ""
    echo -e "${YELLOW}Then reload your shell:${NC}"
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
    print_instructions
}

main "$@"
