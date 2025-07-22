#!/bin/bash
# 3-setup-server.sh
# ==============================================================================
#  Interactive Development Environment Setup Script
#
#  This script installs and configures a development environment with:
#  - NVM, Node.js, PNPM, Bazel
#  - Go and related build/debugging tools
#  - Terraform
#
#  It's designed to be idempotent, meaning it can be run multiple times
#  without causing issues. It checks for existing installations and provides
#  clear feedback.
#
#  How to use:
#  1. Save this script as a file (e.g., `setup_dev_env.sh`).
#  2. Make it executable: `chmod +x setup_dev_env.sh`
#  3. Run it: `./setup_dev_env.sh`
# ==============================================================================

# --- Helper Functions for UI and Checks ---

# Colors for better output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Prints a header for a section
print_header() {
    echo -e "\n${GREEN}======================================================================${NC}"
    echo -e "${GREEN}  $1"
    echo -e "${GREEN}======================================================================${NC}"
}

# Prints a success message
print_success() {
    echo -e "✅ ${GREEN}$1${NC}"
}

# Prints an error message
print_error() {
    echo -e "❌ ${RED}$1${NC}"
}

# Prints an informational message
print_info() {
    echo -e "ℹ️  ${YELLOW}$1${NC}"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Universal profile file detection
detect_profile() {
    if [ -n "$ZSH_VERSION" ]; then
        PROFILE_FILE="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        if [ -f "$HOME/.bashrc" ]; then
            PROFILE_FILE="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            PROFILE_FILE="$HOME/.bash_profile"
        else
            PROFILE_FILE=""
        fi
    else
        PROFILE_FILE=""
    fi
    # Fallback if detection fails
    if [ -z "$PROFILE_FILE" ]; then
        if [ -f "$HOME/.zshrc" ]; then
            PROFILE_FILE="$HOME/.zshrc"
        elif [ -f "$HOME/.bashrc" ]; then
            PROFILE_FILE="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            PROFILE_FILE="$HOME/.bash_profile"
        fi
    fi
    echo "$PROFILE_FILE"
}

# Function to run a command and check its status
run_and_check() {
    local cmd_description="$1"
    shift
    local cmd=("$@")

    echo -n "   -> $cmd_description... "
    # Execute command, suppressing output unless there's an error
    output=$("${cmd[@]}" 2>&1)
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Success${NC}"
        return 0
    else
        echo -e "${RED}Failed${NC}"
        print_error "Error running: ${cmd[*]}"
        echo "   Output:"
        echo "$output" | sed 's/^/     /'
        return 1
    fi
}

# --- Main Installation Functions ---

# --- Node.js Environment ---
install_node_env() {
    print_header "Node.js Environment Setup"

    # Install NVM (Node Version Manager)
    if [ -d "$HOME/.nvm" ]; then
        print_success "NVM is already installed."
    else
        print_info "Installing NVM (Node Version Manager)..."
        # CORRECTED LINE: Wrapped the piped command in 'bash -c "..."'
        if ! run_and_check "Downloading and running NVM install script" \
            bash -c "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash"; then
            print_error "NVM installation failed. Aborting Node setup."
            return 1
        fi
    fi

    # Source NVM script to make it available in this session
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

    # Install Node.js v22
    if nvm ls 22 | grep -q "N/A"; then
        print_info "Installing Node.js v22..."
        if ! run_and_check "Installing Node v22 via nvm" nvm install 22; then
            print_error "Node.js v22 installation failed."
            return 1
        fi
    else
        print_success "Node.js v22 is already installed."
    fi
    nvm use 22
    nvm alias default 22

    # Install PNPM
    if ! command_exists pnpm; then
        run_and_check "Installing PNPM (v8.15.8)" npm install -g pnpm@8.15.8
    else
        print_success "PNPM is already installed."
    fi
}

# --- Go Environment ---
install_go_env() {
    print_header "Go Environment Setup"

    # Check for snap
    if ! command_exists snap; then
        print_error "Snap is not installed. Cannot install Go or Terraform."
        print_info "Please install snapd first. On Debian/Ubuntu: sudo apt update && sudo apt install snapd"
        return 1
    fi

    # Install Go
    if ! command_exists go; then
        print_info "Installing Go..."
        if ! run_and_check "Installing Go via snap" sudo snap install go --classic; then
            print_error "Go installation failed."
            return 1
        fi
    else
        print_success "Go is already installed."
    fi

    # Install Go tools
    print_info "Installing Go development tools..."
    GO_TOOLS=(
        "github.com/bazelbuild/buildtools/buildifier@latest"
        "github.com/bazelbuild/buildtools/buildozer@latest"
        "github.com/bazelbuild/buildtools/unused_deps@latest"
        "github.com/go-delve/delve/cmd/dlv@latest"
    )
    for tool in "${GO_TOOLS[@]}"; do
        run_and_check "Installing $tool" go install "$tool"
    done

    # Add Go bin to PATH
    PROFILE=$(detect_profile)
    if [ -n "$PROFILE" ] && ! grep -q 'export PATH=$PATH:$HOME/go/bin' "$PROFILE"; then
        print_info "Adding Go binary path to your shell profile ($PROFILE)..."
        echo '' >> "$PROFILE"
        echo '# Add Go tools to PATH' >> "$PROFILE"
        echo 'export PATH=$PATH:$HOME/go/bin' >> "$PROFILE"
        print_success "Go path added. Please run 'source $PROFILE' or restart your terminal."
    elif [ -z "$PROFILE" ]; then
        print_error "Could not detect a shell profile file (.zshrc, .bashrc)."
        print_info "Please add 'export PATH=\$PATH:\$HOME/go/bin' to your profile manually."
    else
        print_success "Go path already exists in your shell profile."
    fi
}

install_ffmpeg() {
    print_header "FFmpeg Setup"

    # Check for snap
    if ! command_exists snap; then
        print_error "Snap is not installed. Cannot install Go or Terraform."
        print_info "Please install snapd first. On Debian/Ubuntu: sudo apt update && sudo apt install snapd"
        return 1
    fi

    # Install Go
    if ! command_exists ffmpeg; then
        print_info "Installing FFmpeg.."
        if ! run_and_check "Installing FFmpeg via snap" sudo snap install ffmpeg --classic; then
            print_error "FFmpeg installation failed."
            return 1
        fi
    else
        print_success "FFmpeg is already installed."
    fi
}

# --- Main Script Logic ---
main() {
    # Check for sudo permissions if needed for snap
    if ! command_exists sudo && ( ! command_exists go || ! command_exists terraform ); then
        print_error "sudo is not available, but is required to install Go and Terraform via snap."
        exit 1
    fi
    
    print_header "Starting Development Environment Setup"
    
    install_node_env
    install_go_env
    install_ffmpeg

    print_header "Setup Complete!"
    print_info "Please restart your terminal or run 'source $(detect_profile)' for all changes to take effect."
    print_success "Enjoy your new development environment!"
}

# Run the main function
main
