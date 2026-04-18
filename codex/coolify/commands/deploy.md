---
name: deploy
description: Deploy a service to Coolify via browser automation
args: [service-name]
flags:
  project:
    description: Target project name (default Software Staging)
    type: string
    default: Software Staging
  search:
    description: Search term if different from service name
    type: string
---

# Deploy Service to Coolify

Deploy applications, databases, or one-click services to Coolify using browser automation.

## Usage

```bash
/coolify:deploy open-webui
/coolify:deploy litellm
/coolify:deploy flowise --search "flowise with databases"
/coolify:deploy n8n --project "My first project"
```

## Browser Automation Workflow

### 1. Load Required Tools

```
ToolSearch: select:mcp__claude-in-chrome__tabs_context_mcp
ToolSearch: select:mcp__claude-in-chrome__computer
ToolSearch: select:mcp__claude-in-chrome__read_page
```

### 2. Get Tab Context

```
mcp__claude-in-chrome__tabs_context_mcp(createIfEmpty: true)
```

### 3. Navigate to New Resource Page

```
# Default to localhost (local dev instance)
COOLIFY_URL="${COOLIFY_URL:-http://localhost:8000}"

mcp__claude-in-chrome__navigate(
  tabId: <from context>,
  url: "$COOLIFY_URL/project/<project-uuid>/environment/<env-uuid>/new"
)
```

Or navigate via UI:
1. Click "Projects" in sidebar
2. Click target project
3. Click target environment
4. Click "+ New" button

### 4. Search for Service

```
mcp__claude-in-chrome__read_page(tabId, filter: "interactive")
# Find search textbox ref (usually "Type / to search...")
mcp__claude-in-chrome__computer(action: "left_click", ref: "ref_XX")
mcp__claude-in-chrome__computer(action: "type", text: "<service-name>")
```

### 5. Select Service

```
mcp__claude-in-chrome__computer(action: "screenshot")
# Find service card in results
mcp__claude-in-chrome__computer(action: "left_click", coordinate: [x, y])
```

### 6. Deploy

```
mcp__claude-in-chrome__computer(action: "screenshot")
# Click Deploy button (usually top right)
mcp__claude-in-chrome__computer(action: "left_click", coordinate: [1246, 115])
```

### 7. Wait for Completion

```
mcp__claude-in-chrome__computer(action: "wait", duration: 5)
mcp__claude-in-chrome__computer(action: "screenshot")
# Check logs in Service Startup modal
# Close modal when complete
```

## Available AI Services

| Service | Search Term | Components |
|---------|-------------|------------|
| Open WebUI | `ollama` or `openwebui` | open-webui |
| LiteLLM | `litellm` | litellm, postgres, redis |
| Flowise | `flowise` | flowise |
| Flowise With Databases | `flowise` (second option) | flowise, qdrant, postgres, redis |
| LibreChat | `librechat` | librechat, rag-api, meilisearch, mongodb, vectordb |
| SearXNG | `searxng` | searxng, redis |

## Post-Deploy Configuration

### Connect to Ollama

For services needing Ollama access, add environment variable:
```
OLLAMA_BASE_URL=http://host.docker.internal:11434
```

### View Environment Variables

1. Click service in breadcrumb
2. Click "Environment Variables" in left sidebar
3. **Use "Developer View"** for raw text editing
4. Save and restart if needed

## Service Types

### Databases
- `postgresql` - PostgreSQL 16
- `mysql` - MySQL 8.0
- `redis` - Redis 7
- `mongodb` - MongoDB 8
- `mariadb` - MariaDB 11
- `clickhouse` - ClickHouse

### One-Click Services (90+)
- `n8n` - Workflow automation
- `code-server` - VS Code in browser
- `uptime-kuma` - Uptime monitoring
- `grafana` - Dashboards
- `minio` - S3-compatible storage

### Applications
- **Docker Image** - Deploy from any registry
- **Public Repository** - Deploy from public git
- **Private Repository** - Deploy with GitHub App or deploy key
- **Dockerfile** - Build from Dockerfile
- **Docker Compose** - Multi-container apps

## Troubleshooting

### Service Not Found
- Try alternative search terms
- Check "Filter by category" dropdown
- Some services like Langflow aren't in catalog - use Docker Image

### Deployment Fails
- Check logs in Service Startup modal
- Verify server has enough resources
- Check for port conflicts

### Container Unhealthy
- View container logs via "Logs" tab
- Check environment variables
- Verify dependent services are running
