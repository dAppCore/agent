---
name: deploy-homelab
description: This skill should be used when the user asks to "deploy to homelab", "deploy to lthn.sh", "ship to homelab", "build and deploy", "push image to homelab", or needs to build a Docker image locally and transfer it to the homelab server at 10.69.69.165. Covers the full build-locally → transfer-tarball → deploy pipeline for CorePHP apps.
---

# Deploy to Homelab

Build a CorePHP app Docker image locally (required for paid package auth), transfer via tarball to the homelab (no registry), and deploy.

## When to Use

- Deploying any CorePHP app to the homelab (*.lthn.sh)
- Building images that need `auth.json` for Flux Pro or other paid packages
- Shipping a new version of an app to 10.69.69.165

## Prerequisites

- Docker Desktop running locally
- `auth.json` in the app root (for Flux Pro licence)
- Homelab accessible at 10.69.69.165 (SSH: claude/claude)
- **NEVER ssh directly** — use the deploy script or Ansible from `~/Code/DevOps`

## Process

### 1. Build Locally

Run from the app directory (e.g. `/Users/snider/Code/lab/lthn.ai`):

```bash
# Install deps (auth.json provides paid package access)
composer install --no-dev --optimize-autoloader
npm ci
npm run build

# Build the Docker image for linux/amd64 (homelab is x86_64)
docker build --platform linux/amd64 -t IMAGE_NAME:latest .
```

The image name follows the pattern: `lthn-sh`, `lthn-ai`, etc.

### 2. Transfer to Homelab

```bash
# Save image as compressed tarball
docker save IMAGE_NAME:latest | gzip > /tmp/IMAGE_NAME.tar.gz

# SCP to homelab
sshpass -p claude scp -P 22 /tmp/IMAGE_NAME.tar.gz claude@10.69.69.165:/tmp/

# Load image on homelab
sshpass -p claude ssh -p 22 claude@10.69.69.165 'echo claude | sudo -S docker load < /tmp/IMAGE_NAME.tar.gz'
```

**Note:** Homelab SSH is port 22 (NOT port 4819 — that's production servers). Credentials: claude/claude.

### 3. Deploy on Homelab

```bash
# Restart container with new image
sshpass -p claude ssh -p 22 claude@10.69.69.165 'echo claude | sudo -S docker restart CONTAINER_NAME'

# Or if using docker-compose
sshpass -p claude ssh -p 22 claude@10.69.69.165 'cd /opt/services/APP_DIR && echo claude | sudo -S docker compose up -d'
```

### 4. Post-Deploy Checks

```bash
# Run migrations
sshpass -p claude ssh -p 22 claude@10.69.69.165 'echo claude | sudo -S docker exec CONTAINER_NAME php artisan migrate --force'

# Clear and rebuild caches
sshpass -p claude ssh -p 22 claude@10.69.69.165 'echo claude | sudo -S docker exec CONTAINER_NAME php artisan config:cache && sudo docker exec CONTAINER_NAME php artisan route:cache && sudo docker exec CONTAINER_NAME php artisan view:cache && sudo docker exec CONTAINER_NAME php artisan event:cache'

# Health check
curl -sf https://APP_DOMAIN/up && echo "OK" || echo "FAILED"
```

### One-Shot Script

Use the bundled script for the full pipeline:

```bash
scripts/build-and-ship.sh APP_DIR IMAGE_NAME CONTAINER_NAME
```

Example:
```bash
scripts/build-and-ship.sh /Users/snider/Code/lab/host.uk.com lthn-sh lthn-sh-hub
scripts/build-and-ship.sh /Users/snider/Code/lab/lthn.ai lthn-ai lthn-ai
```

## Or Use Ansible (Preferred)

The Ansible playbooks handle all of this automatically:

```bash
cd ~/Code/DevOps
ansible-playbook playbooks/deploy/website/lthn_sh.yml -i inventory/linux_snider_dev.yml
```

Available playbooks:
- `lthn_sh.yml` — host.uk.com app to homelab
- `lthn_ai.yml` — lthn.ai app to homelab/prod

## Known Apps on Homelab

| App | Image | Container | Port | Data Dir |
|-----|-------|-----------|------|----------|
| host.uk.com | lthn-sh:latest | lthn-sh-hub | 8088 | /opt/services/lthn-lan |
| lthn.ai | lthn-ai:latest | lthn-ai | 80 | /opt/services/lthn-ai |

## Gotchas

- **Platform flag required**: Mac builds ARM images by default. Always use `--platform linux/amd64` — homelab is x86_64 Ryzen 9.
- **auth.json stays local**: The Dockerfile copies the entire app directory. The `.dockerignore` should exclude `auth.json` to avoid leaking licence keys into the image. If it doesn't, add it.
- **Tarball size**: Full images are 500MB–1GB compressed. Ensure `/tmp` has space on both ends.
- **Homelab SSH is port 22**: Unlike production servers (port 4819 + Endlessh on 22), the homelab uses standard port 22.
- **No `sudo` password prompt**: Use `echo claude | sudo -S` pattern for sudo commands over SSH.
- **Redis is embedded**: The FrankenPHP image includes supervisord running Redis. No separate Redis container needed on homelab.
- **GPU services**: The homelab has Ollama (11434), Whisper (9150), TTS (9200), ComfyUI (8188) running natively — the app container connects to them via `127.0.0.1` with `--network host`.

## Consult References

- `references/environments.md` — Environment variables and service mapping for each deployment target
