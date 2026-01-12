#!/bin/sh
# Humantime Installer
# https://github.com/manav03panchal/humantime
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh
#
# Options (via environment variables):
#   HUMANTIME_INSTALL_DIR  - Custom install directory (default: ~/.local/bin or /usr/local/bin)
#   HUMANTIME_VERSION      - Specific version to install (default: latest)
#   HUMANTIME_NO_MODIFY_PATH - Set to 1 to skip PATH modification
#
# Examples:
#   # Install latest version
#   curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh
#
#   # Install specific version
#   curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_VERSION=v0.3.0 sh
#
#   # Install to custom directory
#   curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_INSTALL_DIR=/opt/bin sh

set -e

# Colors (disabled if not a terminal or NO_COLOR is set)
setup_colors() {
    if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        YELLOW='\033[0;33m'
        BLUE='\033[0;34m'
        BOLD='\033[1m'
        RESET='\033[0m'
    else
        RED=''
        GREEN=''
        YELLOW=''
        BLUE=''
        BOLD=''
        RESET=''
    fi
}

info() {
    printf "${BLUE}info${RESET}: %s\n" "$1"
}

success() {
    printf "${GREEN}success${RESET}: %s\n" "$1"
}

warn() {
    printf "${YELLOW}warning${RESET}: %s\n" "$1"
}

error() {
    printf "${RED}error${RESET}: %s\n" "$1" >&2
}

die() {
    error "$1"
    exit 1
}

# Detect the operating system
detect_os() {
    case "$(uname -s)" in
        Linux*)
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                OS="linux"
            else
                OS="linux"
            fi
            ;;
        Darwin*)
            OS="darwin"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            OS="windows"
            ;;
        FreeBSD*)
            OS="freebsd"
            ;;
        *)
            die "Unsupported operating system: $(uname -s)"
            ;;
    esac
    echo "$OS"
}

# Detect the architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv6l)
            ARCH="arm"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            die "Unsupported architecture: $(uname -m)"
            ;;
    esac
    echo "$ARCH"
}

# Check for required commands
check_dependencies() {
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            die "Required command not found: $cmd"
        fi
    done
}

# Get the latest version from GitHub
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/manav03panchal/humantime/releases/latest" 2>/dev/null | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/' || echo ""
}

# Build from source
build_from_source() {
    INSTALL_DIR="$1"

    if ! command -v go >/dev/null 2>&1; then
        die "No releases available and Go is not installed. Please install Go first: https://go.dev/dl/"
    fi

    info "No releases found. Building from source..."

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    info "Cloning repository..."
    if ! git clone --depth 1 https://github.com/manav03panchal/humantime.git "$TMP_DIR/humantime" 2>/dev/null; then
        die "Failed to clone repository"
    fi

    cd "$TMP_DIR/humantime"

    info "Building humantime..."
    if ! go build -o humantime . 2>/dev/null; then
        die "Failed to build humantime"
    fi

    # Create install directory if needed
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        die "Cannot create install directory: $INSTALL_DIR"
    fi

    # Install binary
    if ! mv humantime "$INSTALL_DIR/humantime" 2>/dev/null; then
        warn "Cannot write to $INSTALL_DIR, trying with sudo..."
        if ! sudo mv humantime "$INSTALL_DIR/humantime"; then
            die "Failed to install humantime to $INSTALL_DIR"
        fi
        sudo chmod +x "$INSTALL_DIR/humantime"
    else
        chmod +x "$INSTALL_DIR/humantime"
    fi

    success "Built and installed humantime to $INSTALL_DIR/humantime"
    return 0
}

# Determine the install directory
get_install_dir() {
    if [ -n "${HUMANTIME_INSTALL_DIR:-}" ]; then
        echo "$HUMANTIME_INSTALL_DIR"
        return
    fi

    # Check if /usr/local/bin is writable
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        echo "$HOME/.local/bin"
    else
        echo "$HOME/bin"
    fi
}

# Get the appropriate shell profile file
get_shell_profile() {
    SHELL_NAME=$(basename "${SHELL:-/bin/sh}")
    case "$SHELL_NAME" in
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                echo "$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.profile"
            fi
            ;;
        zsh)
            echo "${ZDOTDIR:-$HOME}/.zshrc"
            ;;
        fish)
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Check if a directory is in PATH
is_in_path() {
    case ":$PATH:" in
        *":$1:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Add directory to PATH in shell profile
add_to_path() {
    INSTALL_DIR="$1"
    PROFILE_FILE=$(get_shell_profile)
    SHELL_NAME=$(basename "${SHELL:-/bin/sh}")

    if is_in_path "$INSTALL_DIR"; then
        info "$INSTALL_DIR is already in your PATH"
        return 0
    fi

    if [ "${HUMANTIME_NO_MODIFY_PATH:-0}" = "1" ]; then
        warn "Skipping PATH modification (HUMANTIME_NO_MODIFY_PATH is set)"
        warn "Add the following to your shell profile manually:"
        echo ""
        echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        return 0
    fi

    info "Adding $INSTALL_DIR to PATH in $PROFILE_FILE"

    # Create profile file if it doesn't exist
    mkdir -p "$(dirname "$PROFILE_FILE")"
    touch "$PROFILE_FILE"

    # Add PATH export based on shell type
    case "$SHELL_NAME" in
        fish)
            # Fish shell uses different syntax
            if ! grep -q "set -gx PATH.*$INSTALL_DIR" "$PROFILE_FILE" 2>/dev/null; then
                echo "" >> "$PROFILE_FILE"
                echo "# Humantime" >> "$PROFILE_FILE"
                echo "set -gx PATH \"$INSTALL_DIR\" \$PATH" >> "$PROFILE_FILE"
            fi
            ;;
        *)
            # POSIX shells (bash, zsh, sh, etc.)
            if ! grep -q "export PATH=.*$INSTALL_DIR" "$PROFILE_FILE" 2>/dev/null; then
                echo "" >> "$PROFILE_FILE"
                echo "# Humantime" >> "$PROFILE_FILE"
                echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$PROFILE_FILE"
            fi
            ;;
    esac

    success "Added $INSTALL_DIR to PATH in $PROFILE_FILE"
    warn "Run 'source $PROFILE_FILE' or restart your shell to use humantime"
}

# Download and install humantime
install_humantime() {
    OS=$(detect_os)
    ARCH=$(detect_arch)
    INSTALL_DIR=$(get_install_dir)
    VERSION="${HUMANTIME_VERSION:-}"

    info "Detected OS: $OS"
    info "Detected architecture: $ARCH"
    info "Install directory: $INSTALL_DIR"

    # Check for supported OS/arch combinations
    case "$OS-$ARCH" in
        darwin-amd64|darwin-arm64|linux-amd64|linux-arm64|windows-amd64)
            ;;
        *)
            die "Unsupported platform: $OS-$ARCH. Please build from source."
            ;;
    esac

    # Get version
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            warn "No releases found on GitHub"
            build_from_source "$INSTALL_DIR"
            add_to_path "$INSTALL_DIR"
            show_success_message "$INSTALL_DIR"
            return 0
        fi
    fi
    info "Installing version: $VERSION"

    # Construct download URL
    BINARY_NAME="humantime-$OS-$ARCH"
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="$BINARY_NAME.exe"
    fi

    DOWNLOAD_URL="https://github.com/manav03panchal/humantime/releases/download/$VERSION/$BINARY_NAME"

    info "Downloading from: $DOWNLOAD_URL"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Download binary
    TMP_FILE="$TMP_DIR/$BINARY_NAME"
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
        die "Failed to download humantime. Please check if the version exists and your network connection."
    fi

    # Create install directory if needed
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        die "Cannot create install directory: $INSTALL_DIR. Try running with sudo or set HUMANTIME_INSTALL_DIR."
    fi

    # Install binary
    FINAL_NAME="humantime"
    if [ "$OS" = "windows" ]; then
        FINAL_NAME="humantime.exe"
    fi

    if ! mv "$TMP_FILE" "$INSTALL_DIR/$FINAL_NAME" 2>/dev/null; then
        # Try with sudo
        warn "Cannot write to $INSTALL_DIR, trying with sudo..."
        if ! sudo mv "$TMP_FILE" "$INSTALL_DIR/$FINAL_NAME"; then
            die "Failed to install humantime to $INSTALL_DIR"
        fi
        sudo chmod +x "$INSTALL_DIR/$FINAL_NAME"
    else
        chmod +x "$INSTALL_DIR/$FINAL_NAME"
    fi

    success "Installed humantime to $INSTALL_DIR/$FINAL_NAME"

    # Restart daemon if it was running
    restart_daemon_if_running "$INSTALL_DIR/$FINAL_NAME"

    # Add to PATH if needed
    add_to_path "$INSTALL_DIR"

    show_success_message "$INSTALL_DIR"
}

# Restart daemon if it was running before upgrade
restart_daemon_if_running() {
    BINARY="$1"

    # Check if daemon command exists and if daemon is running
    if ! "$BINARY" daemon status 2>/dev/null | grep -q "running"; then
        return 0
    fi

    info "Restarting daemon with new version..."

    # Stop the old daemon
    if "$BINARY" daemon stop >/dev/null 2>&1; then
        sleep 1
        # Start with new binary
        if "$BINARY" daemon start >/dev/null 2>&1; then
            success "Daemon restarted with new version"
        else
            warn "Failed to restart daemon. Start manually with: humantime daemon start"
        fi
    else
        warn "Failed to stop old daemon. Restart manually with: humantime daemon stop && humantime daemon start"
    fi
}

# Show success message after installation
show_success_message() {
    INSTALL_DIR="$1"

    echo ""
    printf "${BOLD}${GREEN}Humantime has been installed!${RESET}\n"
    echo ""

    if is_in_path "$INSTALL_DIR"; then
        echo "Get started by running:"
        echo ""
        echo "    humantime version"
        echo "    humantime start on myproject"
        echo ""
    else
        echo "Get started by running:"
        echo ""
        echo "    $INSTALL_DIR/humantime version"
        echo "    $INSTALL_DIR/humantime start on myproject"
        echo ""
        echo "Or restart your shell and run:"
        echo ""
        echo "    humantime version"
        echo ""
    fi

    echo "For more information, visit: https://github.com/manav03panchal/humantime"
}

# Show help
show_help() {
    cat << EOF
Humantime Installer

USAGE:
    curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh

OPTIONS (via environment variables):
    HUMANTIME_INSTALL_DIR      Custom install directory
    HUMANTIME_VERSION          Specific version to install (e.g., v0.3.0)
    HUMANTIME_NO_MODIFY_PATH   Set to 1 to skip PATH modification

EXAMPLES:
    # Install latest version
    curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh

    # Install specific version
    curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_VERSION=v0.3.0 sh

    # Install to custom directory
    curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_INSTALL_DIR=/opt/bin sh

    # Install without modifying PATH
    curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_NO_MODIFY_PATH=1 sh

SUPPORTED PLATFORMS:
    - macOS (Intel and Apple Silicon)
    - Linux (x86_64 and ARM64)
    - Windows (x86_64, via WSL/Git Bash/MSYS2)

For more information, visit: https://github.com/manav03panchal/humantime
EOF
}

# Show ASCII banner
show_banner() {
    printf "${BOLD}${BLUE}"
    cat << 'EOF'
( _       _ _   _   _  _)_ o  _ _   _
 ) ) (_( ) ) ) (_( ) ) (_  ( ) ) ) )_)
                                  (_
EOF
    printf "${RESET}\n"
}

# Main
main() {
    setup_colors

    # Handle help flag
    case "${1:-}" in
        -h|--help|help)
            show_help
            exit 0
            ;;
    esac

    echo ""
    show_banner
    printf "${BOLD}Humantime Installer${RESET}\n"
    echo ""

    check_dependencies
    install_humantime
}

main "$@"
