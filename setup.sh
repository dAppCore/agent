#!/usr/bin/env bash
# Quick setup script - just run: ./setup.sh
set -e

cd "$(dirname "$0")"

echo "=== Host UK Developer Workspace Setup ==="
echo ""

# Install dependencies
./scripts/install-deps.sh

# Install core CLI
./scripts/install-core.sh

# Run doctor to verify
echo ""
echo "=== Verifying environment ==="
core doctor || true

# Clone repos
echo ""
echo "=== Cloning repositories ==="
core setup

echo ""
echo "=== Setup complete! ==="
echo "Run 'core health' to check status"
