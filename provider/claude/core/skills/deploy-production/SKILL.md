---
name: deploy-production
description: This skill should be used when the user asks to "deploy to production", "deploy to de1", "push to prod", "deploy lthn.ai", "deploy host.uk.com", or needs to deploy any website or service to the production fleet. Covers the full Ansible-based deployment pipeline. NEVER ssh directly to production.
---

# Deploy to Production

All production deployments go through Ansible from `~/Code/DevOps`. NEVER ssh directly.

## Quick Reference

```bash
cd ~/Code/DevOps

# Websites
ansible-playbook playbooks/deploy/website/lthn_ai.yml -l primary -e ansible_port=4819
ansible-playbook playbooks/deploy/website/saas.yml -l primary -e ansible_port=4819
ansible-playbook playbooks/deploy/website/core_help.yml -l primary -e ansible_port=4819

# Homelab (different inventory)
ansible-playbook playbooks/deploy/website/lthn_sh.yml -i inventory/linux_snider_dev.yml

# Services
ansible-playbook playbooks/deploy/service/forgejo.yml -l primary -e ansible_port=4819
ansible-playbook playbooks/deploy/service/authentik.yml -l primary -e ansible_port=4819

# Infrastructure
ansible-playbook playbooks/deploy/server/base.yml -l primary -e ansible_port=4819 --tags traefik
```

## Production Fleet

| Host | IP | DC | SSH |
|------|----|----|-----|
| eu-prd-01.lthn.io (de1) | 116.202.82.115 | Falkenstein | Port 4819 |
| eu-prd-noc.lthn.io | 77.42.42.205 | Helsinki | Port 4819 |
| ap-au-syd1.lthn.io | 139.99.131.177 | Sydney | Port 4819 |

Port 22 = Endlessh honeypot. ALWAYS use `-e ansible_port=4819`.

## Website Deploy Pattern (Build + Ship)

For Laravel/CorePHP apps that need local build:

1. **Local build** (needs auth.json for paid packages):
   ```bash
   cd ~/Code/lab/APP_DIR
   composer install --no-dev --optimize-autoloader
   npm ci && npm run build
   docker build --platform linux/amd64 -t IMAGE_NAME:latest .
   docker save IMAGE_NAME:latest | gzip > /tmp/IMAGE_NAME.tar.gz
   ```

2. **Ship to server**:
   ```bash
   scp -P 4819 /tmp/IMAGE_NAME.tar.gz root@116.202.82.115:/tmp/
   ```
   Or let the Ansible playbook handle the transfer.

3. **Deploy via Ansible**:
   ```bash
   cd ~/Code/DevOps
   ansible-playbook playbooks/deploy/website/PLAYBOOK.yml -l primary -e ansible_port=4819
   ```

4. **Verify**:
   ```bash
   curl -sf https://DOMAIN/up
   ```

## Containers on de1

| Website | Container | Port | Domain |
|---------|-----------|------|--------|
| lthn.ai | lthn-ai | 8005/8006 | lthn.ai, api.lthn.ai, mcp.lthn.ai |
| bugseti.app | bugseti-app | 8004 | bugseti.app |
| core.help | core-help | — | core.help |
| SaaS analytics | saas-analytics | 8085 | analytics.host.uk.com |
| SaaS biolinks | saas-biolinks | 8083 | link.host.uk.com |
| SaaS pusher | saas-pusher | 8086 | notify.host.uk.com |
| SaaS socialproof | saas-socialproof | 8087 | trust.host.uk.com |
| SaaS blesta | saas-blesta | 8084 | order.host.uk.com |

## Traefik Routing

De1 uses Docker labels for routing (Traefik Docker provider). Each container declares its own Traefik labels in its docker-compose. Traefik auto-discovers via Docker socket.

Homelab uses file-based routing at `/opt/noc/traefik/config/dynamic.yml`.

## Key Rules

- **NEVER ssh directly** — ALL operations through Ansible or ad-hoc commands
- **Port 4819** — always pass `-e ansible_port=4819` for production hosts
- **Credentials** — stored in `inventory/.credentials/` via Ansible password lookup
- **Dry run** — test with `--check` before applying
- **Existing playbooks** — ALWAYS check `playbooks/deploy/` before creating new ones
- **CLAUDE.md files** — read them at `DevOps/CLAUDE.md`, `playbooks/CLAUDE.md`, `playbooks/deploy/CLAUDE.md`, `playbooks/deploy/website/CLAUDE.md`, `roles/CLAUDE.md`

## Gotchas

- The lthn.ai container on de1 previously ran the FULL host.uk.com app (serving both host.uk.com and lthn.ai domains). Now lthn.ai is a separate split app.
- The host.uk.com SaaS products (analytics, biolinks, pusher, socialproof, blesta) are separate AltumCode containers, NOT part of the CorePHP app.
- host.uk.com itself does NOT have a separate container on de1 yet — it was served by the lthn-ai container. After the split, host.uk.com needs its own container or the lthn-ai playbook needs updating.
- Galera replication: de1 is bootstrap node. Don't run galera playbooks unless you understand the cluster state.
