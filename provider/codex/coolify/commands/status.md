---
name: status
description: Check Coolify deployment status via browser or API
args: [project-or-service]
flags:
  api:
    description: Use API instead of browser automation
    type: boolean
    default: false
  team:
    description: Team to query (default Agentic)
    type: string
    default: Agentic
---

# Check Coolify Status

Query deployment status for projects, services, and resources.

## Usage

```bash
/coolify:status                     # View all projects
/coolify:status "Software Staging"  # View specific project
/coolify:status --api               # Use API instead of browser
```

## Browser Automation (Preferred)

### 1. Load Tools

```
ToolSearch: select:mcp__claude-in-chrome__tabs_context_mcp
ToolSearch: select:mcp__claude-in-chrome__computer
ToolSearch: select:mcp__claude-in-chrome__read_page
```

### 2. Navigate to Projects

```
# Default to localhost (local dev instance)
COOLIFY_URL="${COOLIFY_URL:-http://localhost:8000}"

mcp__claude-in-chrome__tabs_context_mcp(createIfEmpty: true)
mcp__claude-in-chrome__navigate(tabId, url: "$COOLIFY_URL/projects")
```

### 3. Read Project List

```
mcp__claude-in-chrome__computer(action: "screenshot")
```

### 4. Check Specific Project

1. Click project name
2. Click environment (usually "production")
3. View service cards with status indicators

## Status Indicators

| Indicator | Meaning |
|-----------|---------|
| 🟢 Green dot | Running (healthy) |
| 🔴 Red dot | Exited / Failed |
| 🟡 Yellow dot | Deploying / Starting |
| ⚪ Grey dot | Stopped |

## View Service Details

1. Click service card
2. Check tabs:
   - **Configuration** - General settings
   - **Logs** - Container output
   - **Links** - Access URLs

## API Method

### List All Resources

```bash
# Set Coolify URL and token
COOLIFY_URL="${COOLIFY_URL:-http://localhost:8000}"
TOKEN="your-api-token"

# List servers
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/servers" | jq

# List projects
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/projects" | jq

# List services (one-click apps)
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/services" | jq

# List applications
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/applications" | jq

# List databases
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/databases" | jq
```

### Get Specific Resource

```bash
# Get service by UUID
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/services/{uuid}" | jq

# Get service logs
curl -s -H "Authorization: Bearer $TOKEN" "$COOLIFY_URL/api/v1/services/{uuid}/logs" | jq
```

## SSH Verification (Advanced)

For direct container verification when API/UI insufficient:

```bash
# SSH to Coolify server
ssh user@your-coolify-host

# List all containers
docker ps --format 'table {{.Names}}\t{{.Status}}'
```

## Response Fields (API)

| Field | Description |
|-------|-------------|
| `uuid` | Unique identifier |
| `name` | Resource name |
| `status` | running, stopped, deploying, failed |
| `fqdn` | Fully qualified domain name |
| `created_at` | Creation timestamp |
| `updated_at` | Last update timestamp |

## Team Switching

In browser, use team dropdown in top navigation:
1. Click current team name (e.g., "Agentic")
2. Select target team from dropdown
3. Resources will reload for selected team

API tokens are team-scoped - each token only sees its team's resources.
