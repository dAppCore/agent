---
title: Security
description: Security considerations and audit notes for core-agentic
updated: 2026-01-29
---

# Security Considerations

This document outlines security considerations, known issues, and recommendations for the `core-agentic` package.

## Authentication

### API Key Security

**Current Implementation:**
- Keys generated with `ak_` prefix + 32 random characters
- Stored as SHA-256 hash (no salt)
- Key only visible once at creation time
- Supports expiration dates
- Supports revocation

**Known Issues:**

1. **No salt in hash (SEC-001)**
   - Risk: Rainbow table attacks possible against common key formats
   - Mitigation: Keys are high-entropy (32 random chars), reducing practical risk
   - Recommendation: Migrate to Argon2id with salt

2. **Key prefix visible in hash display**
   - The `getMaskedKey()` method shows first 6 chars of the hash, not the original key
   - This is safe but potentially confusing for users

**Recommendations:**
- Consider key rotation reminders
- Add key compromise detection (unusual usage patterns)
- Implement key versioning for smooth rotation

### IP Whitelisting

**Implementation:**
- Per-key IP restriction toggle
- Supports IPv4 and IPv6
- Supports CIDR notation
- Logged when requests blocked

**Validation:**
- Uses `filter_var()` with `FILTER_VALIDATE_IP`
- CIDR prefix validated against IP version limits (0-32 for IPv4, 0-128 for IPv6)
- Normalises IPs for consistent comparison

**Edge Cases Handled:**
- Empty whitelist with restrictions enabled = deny all
- Invalid IPs/CIDRs rejected during configuration
- IP version mismatch (IPv4 vs IPv6) handled correctly

## Authorisation

### Multi-Tenancy

**Workspace Scoping:**
- All models use `BelongsToWorkspace` trait
- Queries automatically scoped to current workspace context
- Missing workspace throws `MissingWorkspaceContextException`

**Known Issues:**

1. **StateSet tool lacks workspace validation (SEC-003)**
   - Risk: Plan lookup by slug without workspace constraint
   - Impact: Could allow cross-tenant state manipulation if slugs collide
   - Fix: Add workspace_id check to plan query

2. **Some tools have soft dependency on workspace**
   - SessionStart marks workspace as optional if plan_slug provided
   - Could theoretically allow workspace inference attacks

### Permission Model

**Scopes:**
- `plans.read` - List and view plans
- `plans.write` - Create, update, archive plans
- `phases.write` - Update phase status, manage tasks
- `sessions.read` - List and view sessions
- `sessions.write` - Start, update, complete sessions
- `tools.read` - View tool analytics
- `templates.read` - List and view templates
- `templates.instantiate` - Create plans from templates

**Tool Scope Enforcement:**
- Each tool declares required scopes
- `AgentToolRegistry::execute()` validates scopes before execution
- Missing scope throws `RuntimeException`

## Rate Limiting

### Current Implementation

**Global Rate Limiting:**
- ForAgentsController: 60 requests/minute per IP
- Configured via `RateLimiter::for('agentic-api')`

**Per-Key Rate Limiting:**
- Configurable per API key (default: 100/minute)
- Uses cache-based counter with 60-second TTL
- Atomic increment via `Cache::add()` + `Cache::increment()`

**Known Issues:**

1. **No per-tool rate limiting (SEC-004)**
   - Risk: Single key can call expensive tools unlimited times
   - Impact: Resource exhaustion, cost overrun
   - Fix: Add tool-specific rate limits

2. **Rate limit counter not distributed**
   - Multiple app servers may have separate counters
   - Fix: Ensure Redis cache driver in production

### Response Headers

Rate limit status exposed via headers:
- `X-RateLimit-Limit` - Maximum requests allowed
- `X-RateLimit-Remaining` - Requests remaining in window
- `X-RateLimit-Reset` - Seconds until reset
- `Retry-After` - When rate limited

## Input Validation

### MCP Tool Inputs

**Validation Helpers:**
- `requireString()` - Type + optional length validation
- `requireInt()` - Type + optional min/max validation
- `requireEnum()` - Value from allowed set
- `requireArray()` - Type validation

**Known Issues:**

1. **Template variable injection (VAL-001)**
   - JSON escaping added but character validation missing
   - Risk: Specially crafted variables could affect template behaviour
   - Recommendation: Add explicit character whitelist

2. **SQL orderByRaw pattern (SEC-002)**
   - TaskCommand uses raw SQL for FIELD() ordering
   - Currently safe (hardcoded values) but fragile pattern
   - Recommendation: Use parameterised approach

### Content Validation

ContentService validates generated content:
- Minimum word count (600 words)
- UK English spelling checks
- Banned word detection
- Structure validation (headings required)

## Data Protection

### Sensitive Data Handling

**API Keys:**
- Plaintext only available once (at creation)
- Hash stored, never logged
- Excluded from model serialisation via `$hidden`

**Session Data:**
- Work logs may contain sensitive context
- Artifacts track file paths (not contents)
- Context summaries could contain user data

**Recommendations:**
- Add data retention policies for sessions
- Consider encrypting context_summary field
- Audit work_log for sensitive data patterns

### Logging

**Current Logging:**
- IP restriction blocks logged with key metadata
- No API key plaintext ever logged
- No sensitive context logged

**Recommendations:**
- Add audit logging for permission changes
- Log key creation/revocation events
- Consider structured logging for SIEM integration

## Transport Security

**Requirements:**
- All endpoints should be HTTPS-only
- MCP portal at mcp.host.uk.com
- API endpoints under /api/agent/*

**Headers Set:**
- `X-Client-IP` - For debugging/audit
- Rate limit headers

**Recommendations:**
- Add HSTS headers
- Consider mTLS for high-security deployments

## Dependency Security

### External API Calls

AI provider services make external API calls:
- Anthropic API (Claude)
- Google AI API (Gemini)
- OpenAI API

**Security Measures:**
- API keys from environment variables only
- HTTPS connections
- 300-second timeout
- Retry with exponential backoff

**Recommendations:**
- Consider API key vault integration
- Add certificate pinning for provider endpoints
- Monitor for API key exposure in responses

### Internal Dependencies

The package depends on:
- `host-uk/core` - Event system
- `host-uk/core-tenant` - Workspace scoping
- `host-uk/core-mcp` - MCP infrastructure

All are internal packages with shared security posture.

## Audit Checklist

### Pre-Production

- [ ] All SEC-* issues in TODO.md addressed
- [ ] API key hashing upgraded to Argon2id
- [ ] StateSet workspace scoping fixed
- [ ] Per-tool rate limiting implemented
- [ ] Test coverage for auth/permission logic

### Regular Audits

- [ ] Review API key usage patterns
- [ ] Check for expired but not revoked keys
- [ ] Audit workspace scope bypass attempts
- [ ] Review rate limit effectiveness
- [ ] Check for unusual tool call patterns

### Incident Response

1. **Compromised API Key**
   - Immediately revoke via `$key->revoke()`
   - Check usage history in database
   - Notify affected workspace owner
   - Review all actions taken with key

2. **Cross-Tenant Access**
   - Disable affected workspace
   - Audit all data access
   - Review workspace scoping logic
   - Implement additional checks

## Security Contacts

For security issues:
- Create private issue in repository
- Email security@host.uk.com
- Do not disclose publicly until patched

## Changelog

**2026-01-29**
- Initial security documentation
- Documented known issues SEC-001 through SEC-004
- Added audit checklist

**2026-01-21**
- Rate limiting functional (was stub)
- Admin routes now require Hades role
- ForAgentsController rate limited
