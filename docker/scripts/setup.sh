#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_DIR="$SCRIPT_DIR/.."
MNT_DIR="$REPO_ROOT/.core/vm/mnt"

echo "=== Core Agent — Local Stack Setup ==="
echo ""

# 1. Create mount directories
echo "[1/7] Creating mount directories..."
mkdir -p "$MNT_DIR"/{config/traefik/certs,data/{mariadb,qdrant,ollama,redis},log/{app,traefik}}

# 2. Generate .env if missing
if [ ! -f "$DOCKER_DIR/.env" ]; then
    echo "[2/7] Creating .env from template..."
    cp "$DOCKER_DIR/.env.example" "$DOCKER_DIR/.env"
    # Generate APP_KEY
    APP_KEY=$(openssl rand -base64 32)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|^APP_KEY=.*|APP_KEY=base64:${APP_KEY}|" "$DOCKER_DIR/.env"
    else
        sed -i "s|^APP_KEY=.*|APP_KEY=base64:${APP_KEY}|" "$DOCKER_DIR/.env"
    fi
    echo "    Generated APP_KEY"
else
    echo "[2/7] .env exists, skipping"
fi

# 3. Generate self-signed TLS certs
CERT_DIR="$MNT_DIR/config/traefik/certs"
if [ ! -f "$CERT_DIR/lthn.sh.crt" ]; then
    echo "[3/7] Generating TLS certificates for *.lthn.sh..."
    if command -v mkcert &>/dev/null; then
        mkcert -install 2>/dev/null || true
        mkcert -cert-file "$CERT_DIR/lthn.sh.crt" \
               -key-file "$CERT_DIR/lthn.sh.key" \
               "lthn.sh" "*.lthn.sh" "localhost" "127.0.0.1"
    else
        echo "    mkcert not found, using openssl self-signed cert"
        openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
            -keyout "$CERT_DIR/lthn.sh.key" \
            -out "$CERT_DIR/lthn.sh.crt" \
            -subj "/CN=*.lthn.sh" \
            -addext "subjectAltName=DNS:lthn.sh,DNS:*.lthn.sh,DNS:localhost,IP:127.0.0.1" \
            2>/dev/null
    fi
    echo "    Certs written to $CERT_DIR/"
else
    echo "[3/7] TLS certs exist, skipping"
fi

# 4. Copy Traefik TLS config
echo "[4/7] Setting up Traefik config..."
cp "$DOCKER_DIR/config/traefik-tls.yml" "$MNT_DIR/config/traefik/tls.yml"

# 5. Build Docker images
echo "[5/7] Building Docker images..."
docker compose -f "$DOCKER_DIR/docker-compose.yml" build

# 6. Start stack
echo "[6/7] Starting stack..."
docker compose -f "$DOCKER_DIR/docker-compose.yml" up -d

# 7. Pull Ollama embedding model
echo "[7/7] Pulling Ollama embedding model..."
echo "    Waiting for Ollama to start..."
sleep 5
docker exec core-ollama ollama pull embeddinggemma 2>/dev/null || \
    docker exec core-ollama ollama pull nomic-embed-text 2>/dev/null || \
    echo "    Warning: Could not pull embedding model. Pull manually: docker exec core-ollama ollama pull embeddinggemma"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Add to /etc/hosts (or use DNS):"
echo "  127.0.0.1  lthn.sh api.lthn.sh mcp.lthn.sh qdrant.lthn.sh ollama.lthn.sh traefik.lthn.sh"
echo ""
echo "Services:"
echo "  https://lthn.sh          — App"
echo "  https://api.lthn.sh      — API"
echo "  https://mcp.lthn.sh      — MCP endpoint"
echo "  https://ollama.lthn.sh   — Ollama"
echo "  https://qdrant.lthn.sh   — Qdrant"
echo "  https://traefik.lthn.sh  — Traefik dashboard"
echo ""
echo "Brain API key: $(grep CORE_BRAIN_KEY "$DOCKER_DIR/.env" | cut -d= -f2)"
