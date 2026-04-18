# Local Development Stack Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Single Dockerfile + docker-compose.yml that gives any community member a working core/agent stack on localhost via `*.lthn.sh` domains.

**Architecture:** Multistage Dockerfile builds the Laravel app (FrankenPHP + Octane + Horizon + Reverb). docker-compose.yml wires 6 services: app, mariadb, qdrant, ollama, redis, traefik. All persistent data mounts to `.core/vm/mnt/{config,data,log}` inside the repo clone. Traefik handles `*.lthn.sh` routing with self-signed TLS. Community members point `*.lthn.sh` DNS to 127.0.0.1 and everything works — same config as the team.

**Tech Stack:** Docker, FrankenPHP, Laravel Octane, MariaDB, Qdrant, Ollama, Redis, Traefik v3

---

## Service Map

| Service | Container | Ports | lthn.sh subdomain |
|---------|-----------|-------|-------------------|
| Laravel App | `core-app` | 8088 (HTTP), 8080 (WebSocket) | `lthn.sh`, `api.lthn.sh`, `mcp.lthn.sh` |
| MariaDB | `core-mariadb` | 3306 | — |
| Qdrant | `core-qdrant` | 6333, 6334 | `qdrant.lthn.sh` |
| Ollama | `core-ollama` | 11434 | `ollama.lthn.sh` |
| Redis | `core-redis` | 6379 | — |
| Traefik | `core-traefik` | 80, 443 | `traefik.lthn.sh` (dashboard) |

## Volume Mount Layout

```
core/agent/
├── .core/vm/mnt/           # gitignored
│   ├── config/
│   │   └── traefik/        # dynamic.yml, certs
│   ├── data/
│   │   ├── mariadb/        # MariaDB data dir
│   │   ├── qdrant/         # Qdrant storage
│   │   ├── ollama/         # Ollama models
│   │   └── redis/          # Redis persistence
│   └── log/
│       ├── app/            # Laravel logs
│       └── traefik/        # Traefik access logs
├── docker/
│   ├── Dockerfile          # Multistage Laravel build
│   ├── docker-compose.yml  # Full stack
│   ├── .env.example        # Template env vars
│   ├── config/
│   │   ├── traefik.yml     # Traefik static config
│   │   ├── dynamic.yml     # Traefik routes (*.lthn.sh)
│   │   ├── supervisord.conf
│   │   └── octane.ini
│   └── scripts/
│       ├── setup.sh        # First-run: generate certs, seed DB, pull models
│       └── entrypoint.sh   # Laravel entrypoint (migrate, cache, etc.)
└── .gitignore              # Already has .core/
```

## File Structure

| File | Purpose |
|------|---------|
| `docker/Dockerfile` | Multistage: composer install → npm build → FrankenPHP runtime |
| `docker/docker-compose.yml` | 6 services, all mounts to `.core/vm/mnt/` |
| `docker/.env.example` | Template with sane defaults for local dev |
| `docker/config/traefik.yml` | Static config: entrypoints, file provider, self-signed TLS |
| `docker/config/dynamic.yml` | Routes: `*.lthn.sh` → services |
| `docker/config/supervisord.conf` | Octane + Horizon + Scheduler + Reverb |
| `docker/config/octane.ini` | PHP OPcache + memory settings |
| `docker/scripts/setup.sh` | First-run bootstrap: mkcert, migrate, seed, pull embedding model |
| `docker/scripts/entrypoint.sh` | Per-start: migrate, cache clear, optimize |

---

## Chunk 1: Docker Foundation

### Task 1: Multistage Dockerfile

**Files:**
- Create: `docker/Dockerfile`
- Create: `docker/config/octane.ini`
- Create: `docker/config/supervisord.conf`
- Create: `docker/scripts/entrypoint.sh`

- [ ] **Step 1: Create octane.ini**

```ini
; PHP settings for Laravel Octane (FrankenPHP)
opcache.enable=1
opcache.memory_consumption=256
opcache.interned_strings_buffer=64
opcache.max_accelerated_files=32531
opcache.validate_timestamps=0
opcache.save_comments=1
opcache.jit=1255
opcache.jit_buffer_size=256M
memory_limit=512M
upload_max_filesize=100M
post_max_size=100M
```

- [ ] **Step 2: Create supervisord.conf**

Based on the production config at `/opt/services/lthn-lan/app/utils/docker/config/supervisord.prod.conf`. Runs 4 processes: Octane (port 8088), Horizon, Scheduler, Reverb (port 8080).

```ini
[supervisord]
nodaemon=true
user=root
logfile=/dev/null
logfile_maxbytes=0
pidfile=/run/supervisord.pid

[program:laravel-setup]
command=/usr/local/bin/entrypoint.sh
autostart=true
autorestart=false
startsecs=0
priority=5
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:octane]
command=php artisan octane:start --server=frankenphp --host=0.0.0.0 --port=8088 --admin-port=2019
directory=/app
autostart=true
autorestart=true
startsecs=5
priority=10
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:horizon]
command=php artisan horizon
directory=/app
autostart=true
autorestart=true
startsecs=5
priority=15
user=nobody
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:scheduler]
command=sh -c "while true; do php artisan schedule:run --verbose --no-interaction; sleep 60; done"
directory=/app
autostart=true
autorestart=true
startsecs=0
priority=20
user=nobody
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0

[program:reverb]
command=php artisan reverb:start --host=0.0.0.0 --port=8080
directory=/app
autostart=true
autorestart=true
startsecs=5
priority=25
user=nobody
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
```

- [ ] **Step 3: Create entrypoint.sh**

```bash
#!/bin/bash
set -e

cd /app

# Wait for MariaDB
until php artisan db:monitor --databases=mariadb 2>/dev/null; do
    echo "[entrypoint] Waiting for MariaDB..."
    sleep 2
done

# Run migrations
php artisan migrate --force --no-interaction

# Cache config/routes/views
php artisan config:cache
php artisan route:cache
php artisan view:cache
php artisan event:cache

# Storage link
php artisan storage:link 2>/dev/null || true

echo "[entrypoint] Laravel ready"
```

- [ ] **Step 4: Create Multistage Dockerfile**

Three stages: `deps` (composer + npm), `frontend` (vite build), `runtime` (FrankenPHP).

```dockerfile
# ============================================================
# Stage 1: PHP Dependencies
# ============================================================
FROM composer:latest AS deps

WORKDIR /build
COPY composer.json composer.lock ./
COPY packages/ packages/
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist

COPY . .
RUN composer dump-autoload --optimize

# ============================================================
# Stage 2: Frontend Build
# ============================================================
FROM node:22-alpine AS frontend

WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
COPY --from=deps /build/vendor vendor
RUN npm run build

# ============================================================
# Stage 3: Runtime
# ============================================================
FROM dunglas/frankenphp:1-php8.5-trixie

RUN install-php-extensions \
    pcntl pdo_mysql redis gd intl zip \
    opcache bcmath exif sockets

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    supervisor curl mariadb-client \
    && rm -rf /var/lib/apt/lists/*

RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"

WORKDIR /app

# Copy built application
COPY --from=deps --chown=www-data:www-data /build /app
COPY --from=frontend /build/public/build /app/public/build

# Config files
COPY docker/config/octane.ini $PHP_INI_DIR/conf.d/octane.ini
COPY docker/config/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Clear build caches
RUN rm -rf bootstrap/cache/*.php \
    storage/framework/cache/data/* \
    storage/framework/sessions/* \
    storage/framework/views/* \
    && php artisan package:discover --ansi

ENV OCTANE_PORT=8088
EXPOSE 8088 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:${OCTANE_PORT}/up || exit 1

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
```

- [ ] **Step 5: Verify Dockerfile syntax**

Run: `docker build --check -f docker/Dockerfile .` (or `docker buildx build --check`)

- [ ] **Step 6: Commit**

```bash
git add docker/Dockerfile docker/config/ docker/scripts/
git commit -m "feat(docker): multistage Dockerfile for local stack

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

### Task 2: Docker Compose

**Files:**
- Create: `docker/docker-compose.yml`
- Create: `docker/.env.example`

- [ ] **Step 1: Create .env.example**

```env
# Core Agent Local Stack
# Copy to .env and adjust as needed

APP_NAME="Core Agent"
APP_ENV=local
APP_DEBUG=true
APP_KEY=
APP_URL=https://lthn.sh
APP_DOMAIN=lthn.sh

# MariaDB
DB_CONNECTION=mariadb
DB_HOST=core-mariadb
DB_PORT=3306
DB_DATABASE=core_agent
DB_USERNAME=core
DB_PASSWORD=core_local_dev

# Redis
REDIS_CLIENT=predis
REDIS_HOST=core-redis
REDIS_PORT=6379
REDIS_PASSWORD=

# Queue
QUEUE_CONNECTION=redis

# Ollama (embeddings)
OLLAMA_URL=http://core-ollama:11434

# Qdrant (vector search)
QDRANT_HOST=core-qdrant
QDRANT_PORT=6334

# Reverb (WebSocket)
REVERB_HOST=0.0.0.0
REVERB_PORT=8080

# Brain API key (agents use this to authenticate)
CORE_BRAIN_KEY=local-dev-key
```

- [ ] **Step 2: Create docker-compose.yml**

```yaml
# Core Agent — Local Development Stack
# Usage: docker compose up -d
# Data: .core/vm/mnt/{config,data,log}

services:
  app:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    container_name: core-app
    env_file: .env
    volumes:
      - ../.core/vm/mnt/log/app:/app/storage/logs
    networks:
      - core-net
    depends_on:
      mariadb:
        condition: service_healthy
      redis:
        condition: service_healthy
      qdrant:
        condition: service_started
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      # Main app
      - "traefik.http.routers.app.rule=Host(`lthn.sh`) || Host(`api.lthn.sh`) || Host(`mcp.lthn.sh`) || Host(`docs.lthn.sh`) || Host(`lab.lthn.sh`)"
      - "traefik.http.routers.app.entrypoints=websecure"
      - "traefik.http.routers.app.tls=true"
      - "traefik.http.routers.app.service=app"
      - "traefik.http.services.app.loadbalancer.server.port=8088"
      # WebSocket (Reverb)
      - "traefik.http.routers.app-ws.rule=Host(`lthn.sh`) && PathPrefix(`/app`)"
      - "traefik.http.routers.app-ws.entrypoints=websecure"
      - "traefik.http.routers.app-ws.tls=true"
      - "traefik.http.routers.app-ws.service=app-ws"
      - "traefik.http.routers.app-ws.priority=10"
      - "traefik.http.services.app-ws.loadbalancer.server.port=8080"

  mariadb:
    image: mariadb:11
    container_name: core-mariadb
    environment:
      MARIADB_ROOT_PASSWORD: ${DB_PASSWORD:-core_local_dev}
      MARIADB_DATABASE: ${DB_DATABASE:-core_agent}
      MARIADB_USER: ${DB_USERNAME:-core}
      MARIADB_PASSWORD: ${DB_PASSWORD:-core_local_dev}
    volumes:
      - ../.core/vm/mnt/data/mariadb:/var/lib/mysql
    networks:
      - core-net
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 10s
      timeout: 5s
      retries: 5

  qdrant:
    image: qdrant/qdrant:v1.17
    container_name: core-qdrant
    volumes:
      - ../.core/vm/mnt/data/qdrant:/qdrant/storage
    networks:
      - core-net
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.qdrant.rule=Host(`qdrant.lthn.sh`)"
      - "traefik.http.routers.qdrant.entrypoints=websecure"
      - "traefik.http.routers.qdrant.tls=true"
      - "traefik.http.services.qdrant.loadbalancer.server.port=6333"

  ollama:
    image: ollama/ollama:latest
    container_name: core-ollama
    volumes:
      - ../.core/vm/mnt/data/ollama:/root/.ollama
    networks:
      - core-net
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.ollama.rule=Host(`ollama.lthn.sh`)"
      - "traefik.http.routers.ollama.entrypoints=websecure"
      - "traefik.http.routers.ollama.tls=true"
      - "traefik.http.services.ollama.loadbalancer.server.port=11434"

  redis:
    image: redis:7-alpine
    container_name: core-redis
    volumes:
      - ../.core/vm/mnt/data/redis:/data
    networks:
      - core-net
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  traefik:
    image: traefik:v3
    container_name: core-traefik
    command:
      - "--api.dashboard=true"
      - "--api.insecure=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.web.http.redirections.entrypoint.to=websecure"
      - "--entrypoints.web.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.websecure.address=:443"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--providers.docker.network=core-net"
      - "--providers.file.directory=/etc/traefik/config"
      - "--providers.file.watch=true"
      - "--log.level=INFO"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ../.core/vm/mnt/config/traefik:/etc/traefik/config
      - ../.core/vm/mnt/log/traefik:/var/log/traefik
    networks:
      - core-net
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.lthn.sh`)"
      - "traefik.http.routers.traefik.entrypoints=websecure"
      - "traefik.http.routers.traefik.tls=true"
      - "traefik.http.routers.traefik.service=api@internal"

networks:
  core-net:
    name: core-net
```

- [ ] **Step 3: Verify compose syntax**

Run: `docker compose -f docker/docker-compose.yml config --quiet`

- [ ] **Step 4: Commit**

```bash
git add docker/docker-compose.yml docker/.env.example
git commit -m "feat(docker): docker-compose with 6 services for local stack

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Chunk 2: Traefik TLS + Setup Script

### Task 3: Traefik TLS Configuration

**Files:**
- Create: `docker/config/traefik-tls.yml`

Traefik needs TLS for `*.lthn.sh`. For local dev, use self-signed certs generated by `mkcert`. The setup script creates them; this config file tells Traefik where to find them.

- [ ] **Step 1: Create Traefik TLS dynamic config**

This goes into `.core/vm/mnt/config/traefik/` at runtime (created by setup.sh). The file in `docker/config/` is the template.

```yaml
# Traefik TLS — local dev (self-signed via mkcert)
tls:
  certificates:
    - certFile: /etc/traefik/config/certs/lthn.sh.crt
      keyFile: /etc/traefik/config/certs/lthn.sh.key
  stores:
    default:
      defaultCertificate:
        certFile: /etc/traefik/config/certs/lthn.sh.crt
        keyFile: /etc/traefik/config/certs/lthn.sh.key
```

- [ ] **Step 2: Commit**

```bash
git add docker/config/traefik-tls.yml
git commit -m "feat(docker): traefik TLS config template for local dev

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

### Task 4: First-Run Setup Script

**Files:**
- Create: `docker/scripts/setup.sh`

- [ ] **Step 1: Create setup.sh**

Handles: directory creation, .env generation, TLS cert generation, Docker build, DB migration, Ollama model pull.

```bash
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
```

- [ ] **Step 2: Make executable and commit**

```bash
chmod +x docker/scripts/setup.sh
git add docker/scripts/setup.sh
git commit -m "feat(docker): first-run setup script with mkcert TLS

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

### Task 5: Update .gitignore

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Ensure .core/ is gitignored**

Check existing `.gitignore` for `.core/` entry. If missing, add:

```
.core/
docker/.env
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: gitignore .core/ and docker/.env

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Summary

**Total: 5 tasks, ~20 steps**

After completion, a community member's workflow is:

```bash
git clone https://github.com/dAppCore/agent.git
cd agent
./docker/scripts/setup.sh
# Add *.lthn.sh to /etc/hosts (or wait for public DNS → 127.0.0.1)
# Done — brain, API, MCP all working on localhost
```

The `.mcp.json` for their Claude Code session:
```json
{
  "mcpServers": {
    "core": {
      "type": "http",
      "url": "https://mcp.lthn.sh",
      "headers": {
        "Authorization": "Bearer $CORE_BRAIN_KEY"
      }
    }
  }
}
```

Same config as the team. DNS determines whether it goes to localhost or the shared infra.
