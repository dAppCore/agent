#!/usr/bin/env bash
# build-and-ship.sh — Build Docker image locally and ship to homelab.
#
# Usage: ./build-and-ship.sh APP_DIR IMAGE_NAME [CONTAINER_NAME]
#
# Examples:
#   ./build-and-ship.sh ~/Code/lab/host.uk.com lthn-sh lthn-sh-hub
#   ./build-and-ship.sh ~/Code/lab/lthn.ai lthn-ai lthn-ai
set -euo pipefail

APP_DIR="${1:?Usage: build-and-ship.sh APP_DIR IMAGE_NAME [CONTAINER_NAME]}"
IMAGE_NAME="${2:?Usage: build-and-ship.sh APP_DIR IMAGE_NAME [CONTAINER_NAME]}"
CONTAINER_NAME="${3:-$IMAGE_NAME}"

HOMELAB_HOST="10.69.69.165"
HOMELAB_USER="claude"
HOMELAB_PASS="claude"
TARBALL="/tmp/${IMAGE_NAME}.tar.gz"

ssh_cmd() {
    sshpass -p "$HOMELAB_PASS" ssh -o StrictHostKeyChecking=no "$HOMELAB_USER@$HOMELAB_HOST" "$@"
}

scp_cmd() {
    sshpass -p "$HOMELAB_PASS" scp -o StrictHostKeyChecking=no "$@"
}

sudo_cmd() {
    ssh_cmd "echo $HOMELAB_PASS | sudo -S $*"
}

echo "=== Build & Ship to Homelab ==="
echo "App:       $APP_DIR"
echo "Image:     $IMAGE_NAME:latest"
echo "Container: $CONTAINER_NAME"
echo "Target:    $HOMELAB_USER@$HOMELAB_HOST"
echo ""

# Step 1: Build dependencies
echo "--- Step 1: Dependencies ---"
cd "$APP_DIR"
composer install --no-dev --optimize-autoloader --quiet
npm ci --silent
npm run build

# Step 2: Docker build
echo ""
echo "--- Step 2: Docker Build (linux/amd64) ---"
docker build --platform linux/amd64 -t "${IMAGE_NAME}:latest" .

# Step 3: Save and transfer
echo ""
echo "--- Step 3: Save & Transfer ---"
echo "Saving image..."
docker save "${IMAGE_NAME}:latest" | gzip > "$TARBALL"
SIZE=$(du -h "$TARBALL" | cut -f1)
echo "Tarball: $TARBALL ($SIZE)"

echo "Transferring to homelab..."
scp_cmd "$TARBALL" "${HOMELAB_USER}@${HOMELAB_HOST}:/tmp/"

# Step 4: Load on homelab
echo ""
echo "--- Step 4: Load Image ---"
sudo_cmd "docker load < /tmp/${IMAGE_NAME}.tar.gz"

# Step 5: Restart container
echo ""
echo "--- Step 5: Restart Container ---"
sudo_cmd "docker restart $CONTAINER_NAME" 2>/dev/null || echo "Container $CONTAINER_NAME not running — start manually"

# Step 6: Post-deploy
echo ""
echo "--- Step 6: Post-Deploy ---"
sleep 3
sudo_cmd "docker exec $CONTAINER_NAME php artisan migrate --force" 2>/dev/null || echo "Migration skipped (container may not be running)"
sudo_cmd "docker exec $CONTAINER_NAME php artisan config:cache" 2>/dev/null || true
sudo_cmd "docker exec $CONTAINER_NAME php artisan route:cache" 2>/dev/null || true
sudo_cmd "docker exec $CONTAINER_NAME php artisan view:cache" 2>/dev/null || true

# Step 7: Health check
echo ""
echo "--- Step 7: Health Check ---"
sleep 2
if sudo_cmd "curl -sf http://localhost:8088/up" >/dev/null 2>&1; then
    echo "Health check: OK"
else
    echo "Health check: FAILED (may need manual start)"
fi

# Cleanup
rm -f "$TARBALL"
echo ""
echo "=== Deploy Complete ==="
