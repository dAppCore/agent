---
title: API Keys
description: Guide to Agent API key management
updated: 2026-01-29
---

# API Key Management

Agent API keys provide authenticated access to the MCP tools and agentic services. This guide covers key creation, permissions, and security.

## Key Structure

API keys follow the format: `ak_` + 32 random alphanumeric characters.

Example: `ak_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`

The key is only displayed once at creation. Store it securely.

## Creating Keys

### Via Admin Panel

1. Navigate to Workspace Settings > API Keys
2. Click "Create New Key"
3. Enter a descriptive name
4. Select permissions
5. Set expiration (optional)
6. Click Create
7. Copy the displayed key immediately

### Programmatically

```php
use Core\Mod\Agentic\Services\AgentApiKeyService;

$service = app(AgentApiKeyService::class);

$key = $service->create(
    workspace: $workspace,
    name: 'My Agent Key',
    permissions: [
        AgentApiKey::PERM_PLANS_READ,
        AgentApiKey::PERM_PLANS_WRITE,
        AgentApiKey::PERM_SESSIONS_WRITE,
    ],
    rateLimit: 100,
    expiresAt: now()->addYear()
);

// Only available once
$plainKey = $key->plainTextKey;
```

## Permissions

### Available Permissions

| Permission | Constant | Description |
|------------|----------|-------------|
| `plans.read` | `PERM_PLANS_READ` | List and view plans |
| `plans.write` | `PERM_PLANS_WRITE` | Create, update, archive plans |
| `phases.write` | `PERM_PHASES_WRITE` | Update phases, manage tasks |
| `sessions.read` | `PERM_SESSIONS_READ` | List and view sessions |
| `sessions.write` | `PERM_SESSIONS_WRITE` | Start, update, end sessions |
| `tools.read` | `PERM_TOOLS_READ` | View tool analytics |
| `templates.read` | `PERM_TEMPLATES_READ` | List and view templates |
| `templates.instantiate` | `PERM_TEMPLATES_INSTANTIATE` | Create plans from templates |
| `notify:read` | `PERM_NOTIFY_READ` | List push campaigns |
| `notify:write` | `PERM_NOTIFY_WRITE` | Create/update campaigns |
| `notify:send` | `PERM_NOTIFY_SEND` | Send notifications |

### Permission Checking

```php
// Single permission
$key->hasPermission('plans.write');

// Any of several
$key->hasAnyPermission(['plans.read', 'sessions.read']);

// All required
$key->hasAllPermissions(['plans.write', 'phases.write']);
```

### Updating Permissions

```php
$service->updatePermissions($key, [
    AgentApiKey::PERM_PLANS_READ,
    AgentApiKey::PERM_SESSIONS_READ,
]);
```

## Rate Limiting

### Configuration

Each key has a configurable rate limit (requests per minute):

```php
$key = $service->create(
    workspace: $workspace,
    name: 'Limited Key',
    permissions: [...],
    rateLimit: 50  // 50 requests/minute
);

// Update later
$service->updateRateLimit($key, 100);
```

### Checking Status

```php
$status = $service->getRateLimitStatus($key);
// Returns:
// [
//     'limit' => 100,
//     'remaining' => 85,
//     'reset_in_seconds' => 45,
//     'used' => 15
// ]
```

### Response Headers

Rate limit info is included in API responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 85
X-RateLimit-Reset: 45
```

When rate limited (HTTP 429):
```
Retry-After: 45
```

## IP Restrictions

Keys can be restricted to specific IP addresses or ranges.

### Enabling Restrictions

```php
// Enable with whitelist
$service->enableIpRestrictions($key, [
    '192.168.1.0/24',      // CIDR range
    '10.0.0.5',            // Single IPv4
    '2001:db8::1',         // Single IPv6
    '2001:db8::/32',       // IPv6 CIDR
]);

// Disable restrictions
$service->disableIpRestrictions($key);
```

### Managing Whitelist

```php
// Add single entry
$key->addToIpWhitelist('192.168.2.0/24');

// Remove entry
$key->removeFromIpWhitelist('192.168.1.0/24');

// Replace entire list
$key->updateIpWhitelist([
    '10.0.0.0/8',
    '172.16.0.0/12',
]);
```

### Parsing Input

For user-entered whitelists:

```php
$result = $service->parseIpWhitelistInput("
    192.168.1.1
    192.168.2.0/24
    # This is a comment
    invalid-ip
");

// Result:
// [
//     'entries' => ['192.168.1.1', '192.168.2.0/24'],
//     'errors' => ['invalid-ip: Invalid IP address']
// ]
```

## Key Lifecycle

### Expiration

```php
// Set expiration on create
$key = $service->create(
    ...
    expiresAt: now()->addMonths(6)
);

// Extend expiration
$service->extendExpiry($key, now()->addYear());

// Remove expiration (never expires)
$service->removeExpiry($key);
```

### Revocation

```php
// Immediately revoke
$service->revoke($key);

// Check status
$key->isRevoked();  // true
$key->isActive();   // false
```

### Status Helpers

```php
$key->isActive();    // Not revoked, not expired
$key->isRevoked();   // Has been revoked
$key->isExpired();   // Past expiration date
$key->getStatusLabel();  // "Active", "Revoked", or "Expired"
```

## Authentication

### Making Requests

Include the API key as a Bearer token:

```bash
curl -H "Authorization: Bearer ak_your_key_here" \
     https://mcp.host.uk.com/api/agent/plans
```

### Authentication Flow

1. Middleware extracts Bearer token
2. Key looked up by SHA-256 hash
3. Status checked (revoked, expired)
4. IP validated if restrictions enabled
5. Permissions checked against required scopes
6. Rate limit checked and incremented
7. Usage recorded (count, timestamp, IP)

### Error Responses

| HTTP Code | Error | Description |
|-----------|-------|-------------|
| 401 | `unauthorised` | Missing or invalid key |
| 401 | `key_revoked` | Key has been revoked |
| 401 | `key_expired` | Key has expired |
| 403 | `ip_not_allowed` | Request IP not whitelisted |
| 403 | `permission_denied` | Missing required permission |
| 429 | `rate_limited` | Rate limit exceeded |

## Usage Tracking

Each key tracks:
- `call_count` - Total lifetime calls
- `last_used_at` - Timestamp of last use
- `last_used_ip` - IP of last request

Access via:
```php
$key->call_count;
$key->getLastUsedForHumans();  // "2 hours ago"
```

## Best Practices

1. **Use descriptive names** - "Production Agent" not "Key 1"
2. **Minimal permissions** - Only grant needed scopes
3. **Set expiration** - Rotate keys periodically
4. **Enable IP restrictions** - When agents run from known IPs
5. **Monitor usage** - Review call patterns regularly
6. **Revoke promptly** - If key may be compromised
7. **Separate environments** - Different keys for dev/staging/prod

## Example: Complete Setup

```php
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Mod\Agentic\Models\AgentApiKey;

$service = app(AgentApiKeyService::class);

// Create a production key
$key = $service->create(
    workspace: $workspace,
    name: 'Production Agent - Claude',
    permissions: [
        AgentApiKey::PERM_PLANS_READ,
        AgentApiKey::PERM_PLANS_WRITE,
        AgentApiKey::PERM_PHASES_WRITE,
        AgentApiKey::PERM_SESSIONS_WRITE,
        AgentApiKey::PERM_TEMPLATES_READ,
        AgentApiKey::PERM_TEMPLATES_INSTANTIATE,
    ],
    rateLimit: 200,
    expiresAt: now()->addYear()
);

// Restrict to known IPs
$service->enableIpRestrictions($key, [
    '203.0.113.0/24',  // Office network
    '198.51.100.50',   // CI/CD server
]);

// Store the key securely
$plainKey = $key->plainTextKey;  // Only chance to get this!
```
