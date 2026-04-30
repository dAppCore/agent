# Coolify Skills

Skills for managing Coolify deployments. Coolify is a self-hosted PaaS (Platform as a Service).

## Overview

Coolify provides:
- Docker container orchestration
- Automatic SSL via Traefik/Caddy
- One-click service deployments (90+ services)
- API-driven infrastructure management

**Documentation**: https://coolify.io/docs

## Instance Configuration

| Environment | URL | Purpose |
|-------------|-----|---------|
| **Local (default)** | `http://localhost:8000` | Developer instance |
| **Docker Internal** | `http://host.docker.internal:8000` | From within containers |

Override with environment variable:
```bash
export COOLIFY_URL="http://your-coolify-instance:8000"
```

## Browser Automation (Preferred Method)

Use Claude-in-Chrome MCP tools for Coolify management:

### Workflow

1. **Get tab context**: `mcp__claude-in-chrome__tabs_context_mcp`
2. **Create/navigate tab**: `mcp__claude-in-chrome__tabs_create_mcp` or `navigate`
3. **Read page elements**: `mcp__claude-in-chrome__read_page` with `filter: "interactive"`
4. **Click elements**: `mcp__claude-in-chrome__computer` with `action: "left_click"` and `ref: "ref_XX"`
5. **Type text**: `mcp__claude-in-chrome__computer` with `action: "type"`
6. **Take screenshots**: `mcp__claude-in-chrome__computer` with `action: "screenshot"`

### Common Tasks

#### Deploy a One-Click Service

1. Navigate to project → environment → "+ New"
2. Search for service in search box
3. Click service card to create
4. Click "Deploy" button (top right)
5. Wait for Service Startup modal to show completion

#### Check Deployment Status

- Look for status indicator next to service name:
  - 🟢 Green dot = Running (healthy)
  - 🔴 Red dot = Exited/Failed
  - 🟡 Yellow = Deploying

#### Configure Environment Variables

1. Click service → "Environment Variables" in left sidebar
2. Use "Developer View" for raw text editing
3. Add variables in format: `KEY=value`
4. Click "Save All Environment Variables"
5. Restart service if needed

## API Access

Tokens are team-scoped. "root" permission means full access within that team.

### Permission Levels
- `root` - Full team access (includes all below)
- `write` - Create/update resources
- `deploy` - Trigger deployments
- `read` - View resources
- `read:sensitive` - View secrets/env vars

### API Examples

```bash
# Set your Coolify URL and token
COOLIFY_URL="${COOLIFY_URL:-http://localhost:8000}"
TOKEN="your-api-token"

# List servers
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/servers" | jq

# List projects
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/projects" | jq

# List services
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/services" | jq
```

## Available One-Click Services

Full list: https://coolify.io/docs/services/all

### AI & ML Services

| Service | Search Term | Description |
|---------|-------------|-------------|
| Open WebUI | `ollama` | Ollama chat interface |
| LiteLLM | `litellm` | Universal LLM API proxy (OpenAI format) |
| Flowise | `flowise` | Low-code LLM orchestration |
| LibreChat | `librechat` | Multi-model chat with RAG |
| SearXNG | `searxng` | Private metasearch engine |

### Automation & DevOps

| Service | Description |
|---------|-------------|
| n8n | Workflow automation |
| Activepieces | No-code automation |
| Code Server | VS Code in browser |
| Gitea | Git hosting |

### Databases

| Service | Description |
|---------|-------------|
| PostgreSQL | Relational database |
| MySQL/MariaDB | Relational database |
| MongoDB | Document database |
| Redis | In-memory cache |
| ClickHouse | Analytics database |

### Monitoring

| Service | Description |
|---------|-------------|
| Uptime Kuma | Uptime monitoring |
| Grafana | Dashboards |
| Prometheus | Metrics |

## Environment Variables Magic

Coolify auto-generates these in docker-compose services:

| Variable Pattern | Description |
|------------------|-------------|
| `SERVICE_FQDN_<NAME>` | Auto-generated FQDN |
| `SERVICE_URL_<NAME>` | Full URL with https:// |
| `SERVICE_FQDN_<NAME>_<PORT>` | FQDN for specific port |
| `SERVICE_PASSWORD_<NAME>` | Auto-generated password |
| `SERVICE_USER_<NAME>` | Auto-generated username |

## Connecting Services

### To Local Ollama

```
OLLAMA_BASE_URL=http://host.docker.internal:11434
```

### Between Coolify Services

Use Docker network DNS:
```
DATABASE_URL=postgres://user:pass@postgres-container-name:5432/db
```

## Troubleshooting

### Service Not Found in Search
- Try alternative search terms
- Check "Filter by category" dropdown
- Some services aren't in catalog - use Docker Image deployment

### Deployment Fails
- Check logs in Service Startup modal
- Verify server has enough resources
- Check for port conflicts

### Container Unhealthy
- View container logs via "Logs" tab
- Check environment variables
- Verify dependent services are running

## Related Documentation

- [All Services](https://coolify.io/docs/services/all)
- [API Reference](https://coolify.io/docs/api-reference)
- [Environment Variables](https://coolify.io/docs/knowledge-base/environment-variables)
