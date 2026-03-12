# Core\Webhook Design

**Date**: 2026-03-12
**Status**: Approved
**Location**: `core/php/src/Core/Webhook/`

## Goal

Framework-level webhook infrastructure: append-only inbound log + config-driven outbound cron triggers. Replaces 4 AltumCode `*-cron` Docker containers. No inline processing — ever.

## Hard Rule

**No inline processing.** The webhook endpoint stores data and returns 200. `WebhookReceived` is for lightweight awareness (increment a counter, set a flag) — never for HTTP calls, DB writes beyond the log, or queue dispatch. Actual work happens when the background worker reads the table on its own schedule. This prevents DDoS through the webhook endpoint.

## Architecture

### Inbound: Record Everything

**Table: `webhook_calls`**

| Column | Type | Purpose |
|--------|------|---------|
| `id` | `ulid` | Primary key |
| `source` | `string(64)` | Tag: `altum-biolinks`, `stripe`, `blesta` |
| `event_type` | `string(128)` nullable | From payload if parseable (e.g. `link.created`) |
| `headers` | `json` | Raw request headers |
| `payload` | `json` | Raw request body |
| `signature_valid` | `boolean` nullable | null = no verifier registered |
| `processed_at` | `timestamp` nullable | Set by consumer when they've handled it |
| `created_at` | `timestamp` | |

Indexed on `(source, processed_at, created_at)` — modules query unprocessed rows by source.

**Route:** `POST /webhooks/{source}` — no auth middleware, rate-limited by IP. Stores the raw request, fires `WebhookReceived` (source + call ID only, no payload in the event), returns `200 OK`.

**Signature verification:** `WebhookVerifier` interface with a single `verify(Request, string $secret): bool` method. Modules register their verifier per source. If no verifier registered, `signature_valid` stays null. If verification fails, still store the row (for debugging) but mark `signature_valid = false`.

### Outbound: Cron Triggers

**`CronTrigger` scheduled action** — runs every minute via the `#[Scheduled]` system. Config-driven:

```php
// config/webhook.php
'cron_triggers' => [
    'altum-biolinks' => [
        'base_url' => env('ALTUM_BIOLINKS_URL'),
        'key' => env('ALTUM_BIOLINKS_CRON_KEY'),
        'endpoints' => ['/cron', '/cron/email_reports', '/cron/broadcasts', '/cron/push_notifications'],
        'stagger_seconds' => 15,
        'offset_seconds' => 5,
    ],
    'altum-analytics' => [
        'base_url' => env('ALTUM_ANALYTICS_URL'),
        'key' => env('ALTUM_ANALYTICS_CRON_KEY'),
        'endpoints' => ['/cron', '/cron/email_reports', '/cron/broadcasts', '/cron/push_notifications'],
        'stagger_seconds' => 15,
        'offset_seconds' => 0,
    ],
    'altum-pusher' => [
        'base_url' => env('ALTUM_PUSHER_URL'),
        'key' => env('ALTUM_PUSHER_CRON_KEY'),
        'endpoints' => [
            '/cron/reset', '/cron/broadcasts', '/cron/campaigns',
            '/cron/flows', '/cron/flows_notifications', '/cron/personal_notifications',
            '/cron/rss_automations', '/cron/recurring_campaigns', '/cron/push_notifications',
        ],
        'stagger_seconds' => 7,
        'offset_seconds' => 7,
    ],
    'altum-socialproof' => [
        'base_url' => env('ALTUM_SOCIALPROOF_URL'),
        'key' => env('ALTUM_SOCIALPROOF_CRON_KEY'),
        'endpoints' => ['/cron', '/cron/email_reports', '/cron/broadcasts', '/cron/push_notifications'],
        'stagger_seconds' => 15,
        'offset_seconds' => 10,
    ],
],
```

The action iterates products with their offset/stagger, hits each endpoint with `GET ?key={key}`. Fire-and-forget HTTP (short timeout, no retries). Logs failures to `webhook_calls` with source `cron-trigger-{product}` for health visibility.

### Lifecycle Event

`WebhookReceived` carries only `source` and `call_id`. Modules subscribe via `$listens` for lightweight awareness — never for processing.

Consuming module pattern (e.g. Mod/Links Boot.php):
```php
public static array $listens = [
    WebhookReceived::class => 'onWebhook',  // lightweight flag only
];
```

Actual processing: a scheduled action or queue job in the module queries `webhook_calls` where `source = 'altum-biolinks' and processed_at is null`.

## File Structure

```
src/Core/Webhook/
├── WebhookCall.php              # Eloquent model (ULID, append-only)
├── WebhookReceived.php          # Lifecycle event (source + call_id only)
├── WebhookController.php        # POST /webhooks/{source}
├── WebhookVerifier.php          # Interface: verify(Request, secret): bool
├── CronTrigger.php              # #[Scheduled('everyMinute')] Action
├── config.php                   # cron_triggers config
└── Migrations/
    └── create_webhook_calls_table.php
```

## What This Replaces

| Before | After |
|--------|-------|
| `saas-biolinks-cron` container (wget loop) | CronTrigger scheduled action |
| `saas-analytics-cron` container (wget loop) | CronTrigger scheduled action |
| `saas-pusher-cron` container (wget loop) | CronTrigger scheduled action |
| `saas-socialproof-cron` container (wget loop) | CronTrigger scheduled action |
| No inbound webhook handling | `POST /webhooks/{source}` → append-only log |

## Scope Boundaries (YAGNI)

- No retry/backoff for outbound cron triggers (fire-and-forget)
- No webhook delivery (outbound webhooks to external consumers) — that's a separate feature
- No payload parsing or transformation — raw JSON storage
- No admin UI — query the table directly or via MCP
- No automatic cleanup/archival — add later if needed

## Success Criteria

- `POST /webhooks/{source}` stores raw request and returns 200 in <10ms
- CronTrigger hits all 4 AltumCode products per-minute with correct stagger
- `signature_valid` correctly set when a verifier is registered
- No inline processing triggered by inbound webhooks
- 4 Docker cron containers can be removed from docker-compose.prod.yml
