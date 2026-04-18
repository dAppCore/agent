# Environment Reference

## Homelab (lthn.sh)

**Host:** 10.69.69.165 (Ryzen 9 + 128GB RAM + RX 7800 XT)
**SSH:** claude:claude (port 22)
**Domains:** *.lthn.sh → 10.69.69.165

### host.uk.com on homelab

| Setting | Value |
|---------|-------|
| Container | lthn-sh-hub |
| Image | lthn-sh:latest |
| Port | 8088 (Octane/FrankenPHP) |
| Network | --network host |
| Data | /opt/services/lthn-lan |
| DB | MariaDB 127.0.0.1:3306, db=lthn_sh |
| Redis | Embedded (supervisord in container) |
| APP_URL | https://lthn.sh |
| SESSION_DOMAIN | .lthn.sh |

### lthn.ai on homelab

| Setting | Value |
|---------|-------|
| Container | lthn-ai |
| Image | lthn-ai:latest |
| Port | 80 (via docker-compose) |
| Network | proxy + lthn-ai bridge |
| Data | /opt/services/lthn-ai |
| DB | MariaDB lthn-ai-db:3306, db=lthn_ai |
| Redis | Embedded |
| APP_URL | https://lthn.sh (homelab) |
| SESSION_DOMAIN | .lthn.sh |

### GPU Services (native on homelab)

| Service | Port | Used By |
|---------|------|---------|
| Ollama | 11434 | LEM scoring (lem-4b model) |
| Whisper | 9150 | Studio speech-to-text |
| Kokoro TTS | 9200 | Studio text-to-speech |
| ComfyUI | 8188 | Studio image generation |
| InfluxDB | via https://influx.infra.lthn.sh | LEM metrics |

### Key .env differences from production

```env
# Homelab-specific
APP_ENV=production
APP_URL=https://lthn.sh
SESSION_DOMAIN=.lthn.sh

# Local GPU services (--network host)
STUDIO_WHISPER_URL=http://127.0.0.1:9150
STUDIO_OLLAMA_URL=http://127.0.0.1:11434
STUDIO_TTS_URL=http://127.0.0.1:9200
STUDIO_COMFYUI_URL=http://127.0.0.1:8188

# Local Redis (embedded in container via supervisord)
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
```

## Production (de1 — Falkenstein)

**Host:** eu-prd-01.lthn.io (Hetzner AX102)
**SSH:** Port 4819 only (port 22 = Endlessh tarpit)
**Deploy:** ONLY via Ansible from ~/Code/DevOps

### Port Map

| Port | Service |
|------|---------|
| 80/443 | Traefik (TLS termination) |
| 2223/3000 | Forgejo |
| 3306 | Galera (MariaDB cluster) |
| 5432 | PostgreSQL |
| 6379 | Dragonfly (Redis-compatible) |
| 8000-8001 | host.uk.com |
| 8003 | lthn.io |
| 8004 | bugseti.app |
| 8005-8006 | lthn.ai |
| 8007 | api.lthn.ai |
| 8008 | mcp.lthn.ai |
| 8083 | 66Biolinks |
| 8084 | Blesta |
| 8085 | Analytics |
| 8086 | Push Notifications |
| 8087 | Social Proof |
| 3900 | Garage S3 |
| 9000/9443 | Authentik |

### Ansible Playbooks

```bash
cd ~/Code/DevOps

# Homelab
ansible-playbook playbooks/deploy/website/lthn_sh.yml -i inventory/linux_snider_dev.yml

# Production (de1)
ansible-playbook playbooks/deploy/website/lthn_ai.yml -i inventory/production.yml
```

## Dockerfile Base

All CorePHP apps use the same Dockerfile pattern:

- Base: `dunglas/frankenphp:1-php8.5-trixie`
- PHP extensions: pcntl, pdo_mysql, redis, gd, intl, zip, opcache, bcmath, exif, sockets
- System packages: redis-server, supervisor, curl, mariadb-client
- Runtime: Supervisord (FrankenPHP + Redis + Horizon + Scheduler)
- Healthcheck: `curl -f http://localhost:${OCTANE_PORT}/up`
