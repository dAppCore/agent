# Module Classification Guide

When splitting an app from host.uk.com, classify each module as **keep** or **remove** based on domain ownership.

## Website Modules

Website modules have `$domains` arrays that define which domains they respond to. Check the regex patterns to determine ownership.

| Module | Domains | Classification |
|--------|---------|----------------|
| Host | `host.uk.com`, `host.test` | host.uk.com only |
| Lthn | `lthn.ai`, `lthn.test`, `lthn.sh` | lthn.ai only |
| App | `app.lthn.*`, `hub.lthn.*` | lthn.ai (client dashboard) |
| Api | `api.lthn.*`, `api.host.*` | Shared — check domain patterns |
| Mcp | `mcp.lthn.*`, `mcp.host.*` | Shared — check domain patterns |
| Docs | `docs.lthn.*`, `docs.host.*` | Shared — check domain patterns |
| Html | Static HTML pages | host.uk.com only |
| Lab | `lab.host.*` | host.uk.com only |
| Service | `*.host.uk.com` service subdomains | host.uk.com only |

**Rule**: If the module's `$domains` patterns match the target domain, keep it. If they only match host.uk.com patterns, remove it. For shared modules (Api, Mcp, Docs), strip the host.uk.com domain patterns.

## Mod Modules (Products)

Mod modules are product-level features. Classify by which platform they serve.

### host.uk.com Products (Remove for lthn.ai)

| Module | Product | Why Remove |
|--------|---------|------------|
| Links | BioHost (link-in-bio) | host.uk.com SaaS product |
| Social | SocialHost (scheduling) | host.uk.com SaaS product |
| Front | Frontend chrome/nav | host.uk.com-specific UI |
| Hub | Admin dashboard | host.uk.com admin panel |
| Trees | Trees for Agents | host.uk.com feature |

### lthn.ai Products (Keep for lthn.ai)

| Module | Product | Why Keep |
|--------|---------|----------|
| Agentic | AI agent orchestration | Core lthn.ai feature |
| Lem | LEM model management | Core lthn.ai feature |
| Mcp | MCP tool registry | Core lthn.ai feature |
| Studio | Multimedia pipeline | lthn.ai content creation |
| Uptelligence | Server monitoring | Cross-platform, lthn.ai relevant |

## Service Providers

Service providers in `app/Service/` are the product layer — they register ServiceDefinition contracts. They depend on their corresponding Mod module.

**Rule**: If the Mod module is removed, the Service provider MUST also be removed. Otherwise the app crashes on boot when it tries to resolve the missing module's classes.

| Service | Depends On | Action |
|---------|-----------|--------|
| Hub | Mod/Hub | Remove with Hub |
| Commerce | Core\Mod\Commerce (package) | Keep — it's a core package |
| Agentic | Core\Mod\Agentic (package) | Keep — it's a core package |

## Core Framework Providers

These are from CorePHP packages (`core/php`, `core/php-admin`, etc.) and should always be kept — they're the framework itself.

- `Core\Storage\CacheResilienceProvider`
- `Core\LifecycleEventProvider`
- `Core\Website\Boot`
- `Core\Bouncer\Boot`
- `Core\Config\Boot`
- `Core\Tenant\Boot`
- `Core\Cdn\Boot`, `Core\Mail\Boot`, `Core\Front\Boot`
- `Core\Headers\Boot`, `Core\Helpers\Boot`
- `Core\Media\Boot`, `Core\Search\Boot`, `Core\Seo\Boot`
- `Core\Webhook\Boot`
- `Core\Api\Boot`
- `Core\Mod\Agentic\Boot`, `Core\Mod\Commerce\Boot`
- `Core\Mod\Uptelligence\Boot`, `Core\Mod\Content\Boot`

## Shared Infrastructure

Some host.uk.com references are shared infrastructure that ALL apps use. These should NOT be changed during the split:

| Reference | Why Keep |
|-----------|----------|
| `analytics.host.uk.com` | Shared analytics service (CSP headers, tracking pixel) |
| `cdn.host.uk.com` | Shared CDN delivery URL |
| Hetzner S3 bucket names (`hostuk`, `host-uk`) | Shared storage |
| BunnyCDN storage zones | Shared CDN zones |
| Footer link to host.uk.com | Legitimate external link |

## Composer Dependencies

After removing modules, review composer.json for packages only needed by removed modules:

| Package | Used By | Action |
|---------|---------|--------|
| `core/php-plug-social` | Mod/Social | Remove |
| `core/php-plug-stock` | Stock photo integration | Keep if any module uses it |
| `webklex/php-imap` | Mod/Support (if removed) | Safe to remove |
| `minishlink/web-push` | Mod/Notify (if removed) | Safe to remove |

**Conservative approach**: Leave deps in place. They don't hurt — they're just unused. Remove later during a cleanup pass.
