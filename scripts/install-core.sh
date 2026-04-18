#!/usr/bin/env bash
set -e

# Install the Core CLI (Unix/Linux/macOS)
# Run: ./scripts/install-core.sh
#
# REQUIREMENTS:
# - curl or wget
# - sha256sum (Linux) or shasum (macOS)
# - git and go (for building from source)
#
# SECURITY CONTROLS:
# - Version pinning prevents supply chain attacks via tag manipulation
# - SHA256 hash verification ensures binary integrity
# - Secure temp file creation prevents symlink attacks
# - Symlink detection prevents directory traversal attacks
# - GPG signature verification (when available) ensures code authenticity
#
# KNOWN LIMITATIONS:
# - Checksums are fetched from same origin as binaries (consider separate trust root)
# - No TLS certificate pinning (relies on system CA store)

REPO="host-uk/core"
VERSION="v0.1.0"  # Pinned version - update when releasing new versions
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BUILD_FROM_SOURCE="${BUILD_FROM_SOURCE:-auto}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

has() {
    command -v "$1" &> /dev/null
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "darwin" ;;
        Linux*)  echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

# Compute SHA256 hash (cross-platform)
compute_sha256() {
    local file=$1
    if has sha256sum; then
        sha256sum "$file" | cut -d' ' -f1
    elif has shasum; then
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
        error "Hash verification failed! Expected: $expected_hash, Got: $actual_hash. The downloaded file may be corrupted or tampered with."
    fi

    info "Hash verification passed (SHA256: ${actual_hash:0:16}...)"
}

# Check if directory is a symlink (security check)
check_not_symlink() {
    local path=$1

    if [[ -L "$path" ]]; then
        error "Directory '$path' is a symbolic link. Possible symlink attack detected."
    fi

    # Additional check using file type
    if [[ -e "$path" ]] && [[ ! -d "$path" ]]; then
        error "Path '$path' exists but is not a directory."
    fi
}

# Try to download pre-built binary with integrity verification
download_binary() {
    local os arch binary_name binary_url checksum_url
    os=$(detect_os)
    arch=$(detect_arch)
    binary_name="core-${os}-${arch}"
    binary_url="https://github.com/$REPO/releases/download/$VERSION/${binary_name}"
    checksum_url="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

    if [[ "$os" == "windows" ]]; then
        binary_name="${binary_name}.exe"
        binary_url="${binary_url}.exe"
    fi

    info "Attempting to download pre-built binary (version $VERSION)..."
    info "URL: $binary_url"

    # Use secure temp file (prevents symlink attacks)
    local temp_file
    temp_file=$(mktemp "${TMPDIR:-/tmp}/core.XXXXXXXXXX") || error "Failed to create temp file"

    # Ensure cleanup on exit/interrupt
    trap "rm -f '$temp_file'" EXIT INT TERM

    # Download binary
    if ! curl -fsSL -o "$temp_file" "$binary_url" 2>/dev/null; then
        rm -f "$temp_file"
        trap - EXIT INT TERM
        warn "No pre-built binary available, will build from source"
        return 1
    fi

    # Download and verify checksum
    info "Verifying download integrity..."
    local checksums
    checksums=$(curl -fsSL "$checksum_url" 2>/dev/null) || {
        rm -f "$temp_file"
        trap - EXIT INT TERM
        warn "Could not download checksums, will build from source"
        return 1
    }

    # Parse checksum file (format: "hash  filename")
    local expected_hash
    expected_hash=$(echo "$checksums" | grep -E "^[a-fA-F0-9]{64}\s+.*${binary_name}$" | head -1 | cut -d' ' -f1)

    if [[ -z "$expected_hash" ]]; then
        rm -f "$temp_file"
        trap - EXIT INT TERM
        error "Could not find checksum for $binary_name in checksums.txt"
    fi

    # Verify hash BEFORE any move operation
    verify_hash "$temp_file" "$expected_hash"

    # Verify install directory is safe
    check_not_symlink "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    check_not_symlink "$INSTALL_DIR"  # Re-verify after mkdir

    # Make executable and move to final location
    chmod +x "$temp_file"
    mv "$temp_file" "$INSTALL_DIR/core"

    trap - EXIT INT TERM  # Clear trap since file was moved

    info "Downloaded and verified: $INSTALL_DIR/core"
    return 0
}

# Verify GPG signature on git tag (if gpg is available)
verify_git_tag_signature() {
    local repo_path=$1
    local tag=$2

    if ! has gpg; then
        warn "GPG not available - skipping tag signature verification"
        warn "For enhanced security, install GPG and import the project signing key"
        return 0  # Continue without verification
    fi

    pushd "$repo_path" > /dev/null
    local result
    result=$(git tag -v "$tag" 2>&1) || {
        # Check if tag is unsigned vs signature invalid
        if echo "$result" | grep -q "error: no signature found"; then
            warn "Tag $tag is not signed - continuing without signature verification"
            popd > /dev/null
            return 0
        else
            popd > /dev/null
            error "GPG signature verification FAILED for tag $tag. Possible tampering detected."
        fi
    }
    info "GPG signature verified for tag $tag"
    popd > /dev/null
}

# Build from source with security checks
build_from_source() {
    if ! has git; then
        error "Git is required to build from source. Run './scripts/install-deps.sh' first"
    fi

    if ! has go; then
        error "Go is required to build from source. Run './scripts/install-deps.sh' first"
    fi

    # Create secure temp directory
    local tmpdir
    tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/core-build.XXXXXXXXXX") || error "Failed to create temp directory"

    # Set restrictive permissions on temp directory
    chmod 700 "$tmpdir"

    # Ensure cleanup on exit/interrupt
    trap "rm -rf '$tmpdir'" EXIT INT TERM

    info "Cloning $REPO (version $VERSION)..."
    local clone_dir="$tmpdir/Core"

    # Clone specific tag for reproducibility
    if ! git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$clone_dir"; then
        error "Failed to clone repository at version $VERSION"
    fi

    # Verify GPG signature on tag (if available)
    verify_git_tag_signature "$clone_dir" "$VERSION"

    info "Building core CLI..."
    pushd "$clone_dir" > /dev/null
    if ! go build -o core ./cmd/core; then
        popd > /dev/null
        error "Go build failed"
    fi
    popd > /dev/null

    # Verify install directory is safe
    check_not_symlink "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    check_not_symlink "$INSTALL_DIR"  # Re-verify after mkdir

    # Move built binary to install location
    mv "$clone_dir/core" "$INSTALL_DIR/core"

    trap - EXIT INT TERM  # Clear trap
    rm -rf "$tmpdir"

    info "Built and installed to $INSTALL_DIR/core"
}

# Add to PATH if needed
setup_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""

        # Try to add to current session
        export PATH="$PATH:$INSTALL_DIR"
    fi
}

# Verify installation
verify() {
    if has core; then
        info "Verifying installation..."
        core --help | head -5
        echo ""
        info "core CLI installed successfully!"
    else
        setup_path
        if [[ -f "$INSTALL_DIR/core" ]]; then
            info "core CLI installed to $INSTALL_DIR/core"
            info "Restart your shell or run: export PATH=\"\$PATH:$INSTALL_DIR\""
        else
            error "Installation failed"
        fi
    fi
}

main() {
    info "Installing Core CLI (version $VERSION)..."

    # Verify install directory is safe before starting
    check_not_symlink "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    check_not_symlink "$INSTALL_DIR"

    # Try download first, fallback to build
    if [[ "$BUILD_FROM_SOURCE" == "true" ]]; then
        build_from_source
    elif [[ "$BUILD_FROM_SOURCE" == "false" ]]; then
        download_binary || error "Download failed and BUILD_FROM_SOURCE=false"
    else
        # auto: try download, fallback to build
        download_binary || build_from_source
    fi

    setup_path
    verify
}

main "$@"
