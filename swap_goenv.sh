#!/usr/bin/env bash
#
# swap_goenv.sh - Cross-platform goenv version swapper
#
# Safely swap between bash and Go versions of goenv on any Unix-like system.
# Works on: macOS (Intel/ARM), Linux, BSD, WSL
#
# Usage:
#   ./swap_goenv.sh build   # Build Go version
#   ./swap_goenv.sh go      # Switch to Go version
#   ./swap_goenv.sh bash    # Switch back to bash version
#   ./swap_goenv.sh status  # Show current version
#

set -e

# Colors (with fallback for systems without color support)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_BINARY="$SCRIPT_DIR/goenv"  # Build output location
BACKUP_DIR="$HOME/.goenv_backup"

# Helper functions
log() { echo -e "${BLUE}→${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Detect goenv installation location (cross-platform)
detect_goenv() {
    local goenv_path=""
    
    # Method 1: Check PATH
    if command -v goenv &> /dev/null; then
        goenv_path=$(command -v goenv)
        # Resolve symlinks (works on macOS and Linux)
        if [ -L "$goenv_path" ]; then
            if command -v readlink &> /dev/null; then
                # GNU readlink (Linux)
                if readlink -f "$goenv_path" &> /dev/null; then
                    goenv_path=$(readlink -f "$goenv_path")
                # BSD readlink (macOS)
                elif [ "$(uname)" = "Darwin" ]; then
                    goenv_path=$(readlink "$goenv_path")
                fi
            fi
        fi
        echo "$goenv_path"
        return 0
    fi
    
    # Method 2: Check Homebrew (macOS/Linux)
    if command -v brew &> /dev/null; then
        local brew_prefix
        brew_prefix=$(brew --prefix 2>/dev/null || echo "")
        if [ -n "$brew_prefix" ] && [ -f "$brew_prefix/bin/goenv" ]; then
            echo "$brew_prefix/bin/goenv"
            return 0
        fi
    fi
    
    # Method 3: Check common Homebrew locations
    local common_paths=(
        "/opt/homebrew/bin/goenv"  # ARM Mac
        "/usr/local/bin/goenv"      # Intel Mac / Linux Homebrew
        "/home/linuxbrew/.linuxbrew/bin/goenv"  # Linux Homebrew
    )
    
    for path in "${common_paths[@]}"; do
        if [ -f "$path" ]; then
            echo "$path"
            return 0
        fi
    done
    
    # Method 4: Check manual installation
    if [ -f "$HOME/.goenv/bin/goenv" ]; then
        echo "$HOME/.goenv/bin/goenv"
        return 0
    fi
    
    # Method 5: Check /usr/bin (system package managers)
    if [ -f "/usr/bin/goenv" ]; then
        echo "/usr/bin/goenv"
        return 0
    fi
    
    return 1
}

# Check if goenv is installed
check_goenv() {
    GOENV_PATH=$(detect_goenv)
    
    if [ -z "$GOENV_PATH" ]; then
        error "goenv not found. Please install goenv first.
  
Options:
  - Homebrew:    brew install goenv
  - Manual:      git clone https://github.com/go-nv/goenv ~/.goenv
  - Package mgr: apt/yum/pkg install goenv"
    fi
    
    log "Found goenv: $GOENV_PATH"
}

# Build the Go version
cmd_build() {
    log "Building Go version..."
    
    if ! command -v go &> /dev/null; then
        error "Go compiler not found. Please install Go first:
  - macOS:  brew install go
  - Linux:  apt install golang / yum install golang
  - Manual: https://golang.org/dl/"
    fi
    
    log "Running: make build"
    make build || error "Build failed"
    
    if [ ! -f "$GO_BINARY" ]; then
        error "Build completed but binary not found: $GO_BINARY"
    fi
    
    success "Built: $GO_BINARY"
    "$GO_BINARY" --version
}

# Show status
cmd_status() {
    echo "═══════════════════════════════════════"
    echo "  goenv Status"
    echo "═══════════════════════════════════════"
    
    # Detect goenv (but don't fail if not found)
    GOENV_PATH=$(detect_goenv 2>/dev/null || echo "")
    
    log "System: $(uname -s) $(uname -m)"
    
    if [ -n "$GOENV_PATH" ]; then
        log "goenv location: $GOENV_PATH"
        
        if [ -f "$GOENV_PATH" ]; then
            # Detect file type
            FILE_TYPE=$(file "$GOENV_PATH" 2>/dev/null || echo "unknown")
            
            if echo "$FILE_TYPE" | grep -qi "executable"; then
                success "Currently: Go version (binary)"
            elif echo "$FILE_TYPE" | grep -qi "script\|text"; then
                warn "Currently: Bash version (script)"
            else
                echo "  Type: Unknown ($FILE_TYPE)"
            fi
            
            log "Version:"
            "$GOENV_PATH" --version 2>&1 || echo "  (error getting version)"
        else
            error "goenv not found at $GOENV_PATH"
        fi
    else
        warn "goenv not found in PATH or common locations"
    fi
    
    echo ""
    log "Go binary: $GO_BINARY"
    if [ -f "$GO_BINARY" ]; then
        success "Exists (size: $(du -h "$GO_BINARY" 2>/dev/null | cut -f1 || echo 'unknown'))"
    else
        warn "Not built yet (run: $0 build)"
    fi
    
    echo ""
    log "Backup: $BACKUP_DIR"
    if [ -d "$BACKUP_DIR" ] && [ -f "$BACKUP_DIR/goenv.bash" ]; then
        success "Exists"
    else
        warn "No backup (will create on first swap)"
    fi
    echo "═══════════════════════════════════════"
}

# Switch to Go version
cmd_go() {
    log "Switching to Go version..."
    
    check_goenv
    
    # Check if Go binary exists
    if [ ! -f "$GO_BINARY" ]; then
        warn "Go binary not built. Building now..."
        cmd_build
    fi
    
    # Create backup if it doesn't exist
    if [ ! -f "$BACKUP_DIR/goenv.bash" ]; then
        log "Creating backup..."
        mkdir -p "$BACKUP_DIR"
        cp "$GOENV_PATH" "$BACKUP_DIR/goenv.bash"
        success "Backed up: $BACKUP_DIR/goenv.bash"
    fi
    
    # Get absolute path for the Go binary
    ABS_GO_BINARY="$(cd "$(dirname "$GO_BINARY")" && pwd)/$(basename "$GO_BINARY")"
    
    # Attempt to copy (try without sudo first)
    log "Replacing with Go version..."
    
    if cp "$ABS_GO_BINARY" "$GOENV_PATH" 2>/dev/null; then
        success "Copied: $GO_BINARY → $GOENV_PATH"
    else
        # Try with sudo if regular copy fails
        log "Regular copy failed, trying with sudo..."
        if sudo cp "$ABS_GO_BINARY" "$GOENV_PATH" 2>/dev/null; then
            success "Copied: $GO_BINARY → $GOENV_PATH (with sudo)"
        else
            error "Cannot copy to $GOENV_PATH
  
Try manually:
  sudo cp $ABS_GO_BINARY $GOENV_PATH
  
Or fix permissions:
  sudo chmod u+w $GOENV_PATH"
        fi
    fi
    
    # Verify the swap
    log "Verifying installation..."
    if "$GOENV_PATH" --version 2>&1 | grep -q "goenv"; then
        success "Switch successful!"
        "$GOENV_PATH" --version
        echo ""
        warn "Reload your shell: hash -r (or restart terminal)"
    else
        error "Verification failed"
    fi
}

# Switch back to bash version
cmd_bash() {
    log "Switching back to bash version..."
    
    check_goenv
    
    # Check if backup exists
    if [ ! -f "$BACKUP_DIR/goenv.bash" ]; then
        warn "No backup found."
        
        # Try reinstalling from package manager
        if command -v brew &> /dev/null; then
            log "Reinstalling from Homebrew..."
            brew reinstall goenv
            success "Reinstalled from Homebrew"
            return 0
        elif command -v apt-get &> /dev/null; then
            log "Reinstalling from apt..."
            sudo apt-get install --reinstall goenv
            success "Reinstalled from apt"
            return 0
        elif command -v yum &> /dev/null; then
            log "Reinstalling from yum..."
            sudo yum reinstall goenv
            success "Reinstalled from yum"
            return 0
        else
            error "Cannot restore: No backup and no package manager found"
        fi
    fi
    
    # Restore from backup
    log "Restoring from backup..."
    
    if cp "$BACKUP_DIR/goenv.bash" "$GOENV_PATH" 2>/dev/null; then
        success "Restored: $BACKUP_DIR/goenv.bash → $GOENV_PATH"
    else
        log "Regular copy failed, trying with sudo..."
        if sudo cp "$BACKUP_DIR/goenv.bash" "$GOENV_PATH" 2>/dev/null; then
            success "Restored: $BACKUP_DIR/goenv.bash → $GOENV_PATH (with sudo)"
        else
            error "Cannot restore to $GOENV_PATH"
        fi
    fi
    
    # Verify the swap
    log "Verifying installation..."
    if file "$GOENV_PATH" 2>/dev/null | grep -qi "script\|text"; then
        success "Switch successful!"
        "$GOENV_PATH" --version 2>&1 || echo "(bash version)"
        echo ""
        warn "Reload your shell: hash -r (or restart terminal)"
    else
        warn "Verification unclear. Trying package manager reinstall..."
        if command -v brew &> /dev/null; then
            brew reinstall goenv
        fi
    fi
}

# Main command dispatcher
main() {
    case "${1:-}" in
        build)
            cmd_build
            ;;
        go)
            echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
            echo -e "${GREEN}║     Switching to Go version of goenv      ║${NC}"
            echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
            echo ""
            cmd_go
            ;;
        bash)
            echo -e "${YELLOW}╔═══════════════════════════════════════════╗${NC}"
            echo -e "${YELLOW}║    Switching to Bash version of goenv     ║${NC}"
            echo -e "${YELLOW}╚═══════════════════════════════════════════╝${NC}"
            echo ""
            cmd_bash
            ;;
        status)
            cmd_status
            ;;
        *)
            echo "Usage: $0 {build|go|bash|status}"
            echo ""
            echo "Commands:"
            echo "  build   - Build the Go version"
            echo "  go      - Switch to Go version"
            echo "  bash    - Switch back to bash version"
            echo "  status  - Show current version and status"
            echo ""
            echo "Cross-platform: Works on macOS, Linux, BSD, WSL"
            echo ""
            echo "Examples:"
            echo "  $0 build    # Build Go version first"
            echo "  $0 status   # Check current version"
            echo "  $0 go       # Switch to Go version"
            echo "  $0 bash     # Switch back to bash version"
            exit 1
            ;;
    esac
}

main "$@"
