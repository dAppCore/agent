#!/usr/bin/env bash
set -e

# Install system dependencies for Host UK development
# Supports: macOS (brew), Linux (apt/dnf), Windows (choco via WSL)
#
# SECURITY NOTES:
# - External install scripts (Homebrew, NodeSource) are downloaded over HTTPS
# - Go binary is verified via SHA256 checksum
# - Composer installer is verified via SHA256 checksum
# - Consider auditing external scripts before running in sensitive environments

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Pinned versions and checksums for security
GO_VERSION="1.22.0"
GO_AMD64_SHA256="f6c8a87aa03b92c4b0bf3d558e28ea03006eb29db78917daec5cfb6ec1046265"
COMPOSER_EXPECTED_SIG="dac665fdc30fdd8ec78b38b9800061b4150413ff2e3b6f88543c636f7cd84f6db9189d43a81e5503cda447da73c7e5b6"

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Compute SHA256 hash (cross-platform)
compute_sha256() {
    local file=$1
    if command -v sha256sum &> /dev/null; then
        sha256sum "$file" | cut -d' ' -f1
    elif command -v shasum &> /dev/null; then
        shasum -a 256 "$file" | cut -d' ' -f1
    else
        error "No SHA256 tool available (need sha256sum or shasum)"
    fi
}

# Verify SHA256 hash of downloaded file
verify_hash() {
    local file=$1
    local expected_hash=$2
    local actual_hash

    actual_hash=$(compute_sha256 "$file")

    if [[ "${actual_hash,,}" != "${expected_hash,,}" ]]; then
        rm -f "$file"
        error "Hash verification failed! Expected: $expected_hash, Got: $actual_hash"
    fi

    info "Hash verification passed"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "macos" ;;
        Linux*)  echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

# Check if command exists
has() {
    command -v "$1" &> /dev/null
}

# Install Homebrew (macOS/Linux)
# NOTE: Homebrew's install script changes frequently, making checksum verification impractical.
# The script is fetched over HTTPS from GitHub. For high-security environments,
# consider auditing the script manually before running.
install_brew() {
    if has brew; then
        info "Homebrew already installed"
        return
    fi

    info "Installing Homebrew..."
    warn "This downloads and executes a script from GitHub. Review at: https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"

    # Download to temp file first (allows manual inspection if needed)
    local temp_script
    temp_script=$(mktemp "${TMPDIR:-/tmp}/brew-install.XXXXXXXXXX.sh")
    curl -fsSL -o "$temp_script" https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh || {
        rm -f "$temp_script"
        error "Failed to download Homebrew installer"
    }

    /bin/bash "$temp_script" || {
        rm -f "$temp_script"
        error "Homebrew installation failed"
    }
    rm -f "$temp_script"

    # Add to PATH for this session
    if [[ -f /opt/homebrew/bin/brew ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [[ -f /home/linuxbrew/.linuxbrew/bin/brew ]]; then
        eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
    fi
}

# Install packages via brew
brew_install() {
    local pkg=$1
    if has "$pkg"; then
        info "$pkg already installed"
    else
        info "Installing $pkg..."
        brew install "$pkg"
    fi
}

# Install packages via apt
apt_install() {
    local pkg=$1
    if has "$pkg"; then
        info "$pkg already installed"
    else
        info "Installing $pkg..."
        sudo apt-get update -qq
        sudo apt-get install -y "$pkg"
    fi
}

# macOS setup
setup_macos() {
    info "Setting up macOS environment..."

    install_brew

    brew_install git
    brew_install gh
    brew_install go
    brew_install php
    brew_install composer
    brew_install node
    brew_install pnpm

    # Optional
    if ! has docker; then
        warn "Docker not installed. Install Docker Desktop manually if needed."
    fi
}

# Linux setup
setup_linux() {
    info "Setting up Linux environment..."

    # Detect package manager
    if has apt-get; then
        setup_linux_apt
    elif has dnf; then
        setup_linux_dnf
    elif has brew; then
        setup_linux_brew
    else
        warn "Unknown package manager. Installing Homebrew..."
        install_brew
        setup_linux_brew
    fi
}

setup_linux_apt() {
    apt_install git
    apt_install gh || warn "GitHub CLI may need manual install: https://cli.github.com"

    # Go
    if ! has go; then
        info "Installing Go..."
        sudo apt-get install -y golang-go || {
            # Fallback to manual install for newer version with integrity verification
            local go_tarball="go${GO_VERSION}.linux-amd64.tar.gz"
            local go_url="https://go.dev/dl/${go_tarball}"
            local temp_file
            temp_file=$(mktemp "${TMPDIR:-/tmp}/go.XXXXXXXXXX.tar.gz")

            info "Downloading Go $GO_VERSION..."
            curl -fsSL -o "$temp_file" "$go_url" || {
                rm -f "$temp_file"
                error "Failed to download Go"
            }

            info "Verifying Go download integrity..."
            verify_hash "$temp_file" "$GO_AMD64_SHA256"

            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "$temp_file"
            rm -f "$temp_file"
            export PATH=$PATH:/usr/local/go/bin
        }
    fi

    # PHP
    apt_install php

    # Composer (with installer signature verification)
    if ! has composer; then
        info "Installing Composer..."
        local temp_dir
        temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/composer.XXXXXXXXXX")
        chmod 700 "$temp_dir"

        # Download installer
        curl -fsSL -o "$temp_dir/composer-setup.php" https://getcomposer.org/installer

        # Verify installer signature (SHA384)
        local actual_sig
        actual_sig=$(php -r "echo hash_file('sha384', '$temp_dir/composer-setup.php');")
        if [[ "$actual_sig" != "$COMPOSER_EXPECTED_SIG" ]]; then
            rm -rf "$temp_dir"
            error "Composer installer signature verification failed!"
        fi
        info "Composer installer signature verified"

        # Run installer
        php "$temp_dir/composer-setup.php" --install-dir="$temp_dir"
        sudo mv "$temp_dir/composer.phar" /usr/local/bin/composer
        rm -rf "$temp_dir"
    fi

    # Node (via NodeSource)
    # NOTE: NodeSource setup script changes frequently, making checksum verification impractical.
    # For high-security environments, consider using nvm or building from source.
    if ! has node; then
        info "Installing Node.js..."
        warn "This downloads and executes a script from NodeSource. Review at: https://deb.nodesource.com/setup_20.x"

        local temp_script
        temp_script=$(mktemp "${TMPDIR:-/tmp}/nodesource-setup.XXXXXXXXXX.sh")
        curl -fsSL -o "$temp_script" https://deb.nodesource.com/setup_20.x || {
            rm -f "$temp_script"
            error "Failed to download NodeSource setup script"
        }

        sudo -E bash "$temp_script" || {
            rm -f "$temp_script"
            error "NodeSource setup failed"
        }
        rm -f "$temp_script"
        sudo apt-get install -y nodejs
    fi

    # pnpm
    if ! has pnpm; then
        info "Installing pnpm..."
        npm install -g pnpm
    fi
}

setup_linux_dnf() {
    sudo dnf install -y git gh golang php composer nodejs
    npm install -g pnpm
}

setup_linux_brew() {
    install_brew
    brew_install git
    brew_install gh
    brew_install go
    brew_install php
    brew_install composer
    brew_install node
    brew_install pnpm
}

# Windows setup (via chocolatey in PowerShell or WSL)
setup_windows() {
    warn "Windows detected. Please run scripts/install-deps.ps1 in PowerShell"
    warn "Or use WSL with: wsl ./scripts/install-deps.sh"
    exit 1
}

# Main
main() {
    local os=$(detect_os)
    info "Detected OS: $os"

    case "$os" in
        macos)   setup_macos ;;
        linux)   setup_linux ;;
        windows) setup_windows ;;
        *)       error "Unsupported OS: $os" ;;
    esac

    info "Dependencies installed!"
    echo ""
    echo "Next: Run './scripts/install-core.sh' to install the core CLI"
}

main "$@"
