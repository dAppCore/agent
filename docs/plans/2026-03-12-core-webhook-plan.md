# Core\Webhook Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add framework-level webhook infrastructure to core/php — append-only inbound log, config-driven outbound cron triggers, replaces 4 AltumCode Docker cron containers.

**Architecture:** `Core\Webhook` namespace in `src/Core/Webhook/`. One migration, one model, one controller, one lifecycle event, one scheduled action for cron triggers, one verifier interface. Modules subscribe to `WebhookReceived` for awareness; they query the table for processing on their own schedule. No inline processing — ever.

**Tech Stack:** Laravel 12, Pest testing, Eloquent (ULID keys), `#[Scheduled]` attribute, lifecycle events (`ApiRoutesRegistering`), HTTP client for outbound triggers.

---

### Task 1: Migration — `webhook_calls` table

**Files:**
- Create: `database/migrations/2026_03_12_000001_create_webhook_calls_table.php`

**Step 1: Create the migration**

```php
<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('webhook_calls', function (Blueprint $table) {
            $table->ulid('id')->primary();
            $table->string('source', 64)->index();
            $table->string('event_type', 128)->nullable();
            $table->json('headers');
            $table->json('payload');
            $table->boolean('signature_valid')->nullable();
            $table->timestamp('processed_at')->nullable();
            $table->timestamp('created_at')->useCurrent();

            $table->index(['source', 'processed_at', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('webhook_calls');
    }
};
```

**Step 2: Verify**

```bash
cd /Users/snider/Code/core/php
head -5 database/migrations/2026_03_12_000001_create_webhook_calls_table.php
```

Expected: `declare(strict_types=1)` and `use Illuminate\Database\Migrations\Migration`.

**Step 3: Commit**

```bash
git add database/migrations/2026_03_12_000001_create_webhook_calls_table.php
git commit -m "feat(webhook): add webhook_calls migration — append-only inbound log"
```

---

### Task 2: Model — `WebhookCall`

**Files:**
- Create: `src/Core/Webhook/WebhookCall.php`
- Test: `tests/Feature/WebhookCallTest.php`

**Step 1: Write the test**

```php
<?php

declare(strict_types=1);

namespace Core\Tests\Feature;

use Core\Webhook\WebhookCall;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Core\Tests\TestCase;

class WebhookCallTest extends TestCase
{
    use RefreshDatabase;

    public function test_create_webhook_call(): void
    {
        $call = WebhookCall::create([
            'source' => 'altum-biolinks',
            'event_type' => 'link.created',
            'headers' => ['webhook-id' => 'abc123'],
            'payload' => ['type' => 'link.created', 'data' => ['id' => 1]],
        ]);

        $this->assertNotNull($call->id);
        $this->assertSame('altum-biolinks', $call->source);
        $this->assertSame('link.created', $call->event_type);
        $this->assertIsArray($call->headers);
        $this->assertIsArray($call->payload);
        $this->assertNull($call->signature_valid);
        $this->assertNull($call->processed_at);
    }

    public function test_unprocessed_scope(): void
    {
        WebhookCall::create([
            'source' => 'stripe',
            'headers' => [],
            'payload' => ['type' => 'invoice.paid'],
        ]);

        WebhookCall::create([
            'source' => 'stripe',
            'headers' => [],
            'payload' => ['type' => 'invoice.created'],
            'processed_at' => now(),
        ]);

        $unprocessed = WebhookCall::unprocessed()->get();
        $this->assertCount(1, $unprocessed);
        $this->assertSame('invoice.paid', $unprocessed->first()->payload['type']);
    }

    public function test_for_source_scope(): void
    {
        WebhookCall::create(['source' => 'stripe', 'headers' => [], 'payload' => []]);
        WebhookCall::create(['source' => 'altum-biolinks', 'headers' => [], 'payload' => []]);

        $this->assertCount(1, WebhookCall::forSource('stripe')->get());
        $this->assertCount(1, WebhookCall::forSource('altum-biolinks')->get());
    }

    public function test_mark_processed(): void
    {
        $call = WebhookCall::create([
            'source' => 'test',
            'headers' => [],
            'payload' => [],
        ]);

        $this->assertNull($call->processed_at);

        $call->markProcessed();

        $this->assertNotNull($call->fresh()->processed_at);
    }

    public function test_signature_valid_is_nullable_boolean(): void
    {
        $call = WebhookCall::create([
            'source' => 'test',
            'headers' => [],
            'payload' => [],
            'signature_valid' => false,
        ]);

        $this->assertFalse($call->signature_valid);
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/snider/Code/core/php
vendor/bin/phpunit tests/Feature/WebhookCallTest.php
```

Expected: FAIL — class `Core\Webhook\WebhookCall` not found.

**Step 3: Create the model**

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Concerns\HasUlids;
use Illuminate\Database\Eloquent\Model;

class WebhookCall extends Model
{
    use HasUlids;

    public $timestamps = false;

    protected $fillable = [
        'source',
        'event_type',
        'headers',
        'payload',
        'signature_valid',
        'processed_at',
    ];

    protected function casts(): array
    {
        return [
            'headers' => 'array',
            'payload' => 'array',
            'signature_valid' => 'boolean',
            'processed_at' => 'datetime',
            'created_at' => 'datetime',
        ];
    }

    public function scopeUnprocessed(Builder $query): Builder
    {
        return $query->whereNull('processed_at');
    }

    public function scopeForSource(Builder $query, string $source): Builder
    {
        return $query->where('source', $source);
    }

    public function markProcessed(): void
    {
        $this->update(['processed_at' => now()]);
    }
}
```

**Step 4: Run tests**

```bash
vendor/bin/phpunit tests/Feature/WebhookCallTest.php
```

Expected: All 5 tests pass.

**Step 5: Commit**

```bash
git add src/Core/Webhook/WebhookCall.php tests/Feature/WebhookCallTest.php
git commit -m "feat(webhook): add WebhookCall model — ULID, scopes, markProcessed"
```

---

### Task 3: Lifecycle event — `WebhookReceived`

**Files:**
- Create: `src/Core/Webhook/WebhookReceived.php`

**Step 1: Create the event**

This is a simple value object. It carries only the source tag and call ID — never the payload. Modules subscribe to it for lightweight awareness only.

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

class WebhookReceived
{
    public function __construct(
        public readonly string $source,
        public readonly string $callId,
    ) {}
}
```

**Step 2: Commit**

```bash
git add src/Core/Webhook/WebhookReceived.php
git commit -m "feat(webhook): add WebhookReceived event — source + callId only"
```

---

### Task 4: Verifier interface — `WebhookVerifier`

**Files:**
- Create: `src/Core/Webhook/WebhookVerifier.php`

**Step 1: Create the interface**

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

use Illuminate\Http\Request;

interface WebhookVerifier
{
    /**
     * Verify the webhook signature.
     *
     * Returns true if valid, false if invalid.
     * Implementations should check headers like webhook-signature against
     * the raw request body using the provided secret.
     */
    public function verify(Request $request, string $secret): bool;
}
```

**Step 2: Commit**

```bash
git add src/Core/Webhook/WebhookVerifier.php
git commit -m "feat(webhook): add WebhookVerifier interface — per-source signature check"
```

---

### Task 5: Controller — `WebhookController`

**Files:**
- Create: `src/Core/Webhook/WebhookController.php`
- Test: `tests/Feature/WebhookControllerTest.php`

**Step 1: Write the test**

```php
<?php

declare(strict_types=1);

namespace Core\Tests\Feature;

use Core\Webhook\WebhookCall;
use Core\Webhook\WebhookReceived;
use Core\Webhook\WebhookVerifier;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Event;
use Core\Tests\TestCase;

class WebhookControllerTest extends TestCase
{
    use RefreshDatabase;

    protected function defineRoutes($router): void
    {
        $router->post('/webhooks/{source}', [\Core\Webhook\WebhookController::class, 'handle']);
    }

    public function test_stores_webhook_call(): void
    {
        $response = $this->postJson('/webhooks/altum-biolinks', [
            'type' => 'link.created',
            'data' => ['id' => 42],
        ]);

        $response->assertOk();

        $call = WebhookCall::first();
        $this->assertNotNull($call);
        $this->assertSame('altum-biolinks', $call->source);
        $this->assertSame(['type' => 'link.created', 'data' => ['id' => 42]], $call->payload);
        $this->assertNull($call->processed_at);
    }

    public function test_captures_headers(): void
    {
        $this->postJson('/webhooks/stripe', ['type' => 'invoice.paid'], [
            'Webhook-Id' => 'msg_abc123',
            'Webhook-Timestamp' => '1234567890',
        ]);

        $call = WebhookCall::first();
        $this->assertArrayHasKey('webhook-id', $call->headers);
    }

    public function test_fires_webhook_received_event(): void
    {
        Event::fake([WebhookReceived::class]);

        $this->postJson('/webhooks/altum-biolinks', ['type' => 'test']);

        Event::assertDispatched(WebhookReceived::class, function ($event) {
            return $event->source === 'altum-biolinks' && ! empty($event->callId);
        });
    }

    public function test_extracts_event_type_from_payload(): void
    {
        $this->postJson('/webhooks/stripe', ['type' => 'invoice.paid']);

        $this->assertSame('invoice.paid', WebhookCall::first()->event_type);
    }

    public function test_handles_empty_payload(): void
    {
        $response = $this->postJson('/webhooks/test', []);

        $response->assertOk();
        $this->assertCount(1, WebhookCall::all());
    }

    public function test_signature_valid_null_when_no_verifier(): void
    {
        $this->postJson('/webhooks/unknown-source', ['data' => 1]);

        $this->assertNull(WebhookCall::first()->signature_valid);
    }

    public function test_signature_verified_when_verifier_registered(): void
    {
        $verifier = new class implements WebhookVerifier {
            public function verify(Request $request, string $secret): bool
            {
                return $request->header('webhook-signature') === 'valid';
            }
        };

        $this->app->instance('webhook.verifier.test-source', $verifier);
        $this->app['config']->set('webhook.secrets.test-source', 'test-secret');

        $this->postJson('/webhooks/test-source', ['data' => 1], [
            'Webhook-Signature' => 'valid',
        ]);

        $this->assertTrue(WebhookCall::first()->signature_valid);
    }

    public function test_signature_invalid_still_stores_call(): void
    {
        $verifier = new class implements WebhookVerifier {
            public function verify(Request $request, string $secret): bool
            {
                return false;
            }
        };

        $this->app->instance('webhook.verifier.test-source', $verifier);
        $this->app['config']->set('webhook.secrets.test-source', 'test-secret');

        $this->postJson('/webhooks/test-source', ['data' => 1]);

        $call = WebhookCall::first();
        $this->assertNotNull($call);
        $this->assertFalse($call->signature_valid);
    }

    public function test_source_is_sanitised(): void
    {
        $response = $this->postJson('/webhooks/valid-source-123', ['data' => 1]);
        $response->assertOk();

        $response = $this->postJson('/webhooks/invalid source!', ['data' => 1]);
        $response->assertStatus(404);
    }
}
```

**Step 2: Run test to verify it fails**

```bash
vendor/bin/phpunit tests/Feature/WebhookControllerTest.php
```

Expected: FAIL — class `Core\Webhook\WebhookController` not found.

**Step 3: Create the controller**

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class WebhookController
{
    public function handle(Request $request, string $source): JsonResponse
    {
        $signatureValid = null;

        // Check for registered verifier
        $verifier = app()->bound("webhook.verifier.{$source}")
            ? app("webhook.verifier.{$source}")
            : null;

        if ($verifier instanceof WebhookVerifier) {
            $secret = config("webhook.secrets.{$source}", '');
            $signatureValid = $verifier->verify($request, $secret);
        }

        // Extract event type from common payload patterns
        $payload = $request->json()->all();
        $eventType = $payload['type'] ?? $payload['event_type'] ?? $payload['event'] ?? null;

        $call = WebhookCall::create([
            'source' => $source,
            'event_type' => is_string($eventType) ? $eventType : null,
            'headers' => $request->headers->all(),
            'payload' => $payload,
            'signature_valid' => $signatureValid,
        ]);

        event(new WebhookReceived($source, $call->id));

        return response()->json(['ok' => true]);
    }
}
```

**Step 4: Run tests**

```bash
vendor/bin/phpunit tests/Feature/WebhookControllerTest.php
```

Expected: All 9 tests pass.

**Step 5: Commit**

```bash
git add src/Core/Webhook/WebhookController.php tests/Feature/WebhookControllerTest.php
git commit -m "feat(webhook): add WebhookController — store, verify, fire event, return 200"
```

---

### Task 6: Config + route registration

**Files:**
- Create: `src/Core/Webhook/config.php`
- Create: `src/Core/Webhook/Boot.php`

**Step 1: Create the config**

```php
<?php

declare(strict_types=1);

return [
    /*
    |--------------------------------------------------------------------------
    | Webhook Secrets
    |--------------------------------------------------------------------------
    |
    | Per-source signing secrets for signature verification.
    | Modules register WebhookVerifier implementations per source.
    |
    */
    'secrets' => [
        'altum-biolinks' => env('ALTUM_BIOLINKS_WEBHOOK_SECRET'),
        'altum-analytics' => env('ALTUM_ANALYTICS_WEBHOOK_SECRET'),
        'altum-pusher' => env('ALTUM_PUSHER_WEBHOOK_SECRET'),
        'altum-socialproof' => env('ALTUM_SOCIALPROOF_WEBHOOK_SECRET'),
    ],

    /*
    |--------------------------------------------------------------------------
    | Cron Triggers
    |--------------------------------------------------------------------------
    |
    | Outbound HTTP triggers that replace Docker cron containers.
    | The CronTrigger action hits these endpoints every minute.
    |
    */
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
];
```

**Step 2: Create Boot.php**

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

use Core\Events\ApiRoutesRegistering;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider
{
    public static array $listens = [
        ApiRoutesRegistering::class => 'onApiRoutes',
    ];

    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__.'/config.php', 'webhook');
    }

    public function onApiRoutes(ApiRoutesRegistering $event): void
    {
        $event->routes(fn () => Route::post(
            '/webhooks/{source}',
            [WebhookController::class, 'handle']
        )->where('source', '[a-z0-9\-]+'));
    }
}
```

**Step 3: Commit**

```bash
git add src/Core/Webhook/config.php src/Core/Webhook/Boot.php
git commit -m "feat(webhook): add config + Boot — route registration, cron trigger config"
```

---

### Task 7: CronTrigger scheduled action

**Files:**
- Create: `src/Core/Webhook/CronTrigger.php`
- Test: `tests/Feature/CronTriggerTest.php`

**Step 1: Write the test**

```php
<?php

declare(strict_types=1);

namespace Core\Tests\Feature;

use Core\Actions\Action;
use Core\Actions\Scheduled;
use Core\Webhook\CronTrigger;
use Illuminate\Support\Facades\Http;
use Core\Tests\TestCase;

class CronTriggerTest extends TestCase
{
    public function test_has_scheduled_attribute(): void
    {
        $ref = new \ReflectionClass(CronTrigger::class);
        $attrs = $ref->getAttributes(Scheduled::class);

        $this->assertCount(1, $attrs);
        $this->assertSame('everyMinute', $attrs[0]->newInstance()->frequency);
    }

    public function test_uses_action_trait(): void
    {
        $this->assertTrue(
            in_array(Action::class, class_uses_recursive(CronTrigger::class), true)
        );
    }

    public function test_hits_configured_endpoints(): void
    {
        Http::fake();

        config(['webhook.cron_triggers' => [
            'test-product' => [
                'base_url' => 'https://example.com',
                'key' => 'secret123',
                'endpoints' => ['/cron', '/cron/reports'],
                'stagger_seconds' => 0,
                'offset_seconds' => 0,
            ],
        ]]);

        CronTrigger::run();

        Http::assertSentCount(2);
        Http::assertSent(fn ($request) => str_contains($request->url(), '/cron?key=secret123'));
        Http::assertSent(fn ($request) => str_contains($request->url(), '/cron/reports?key=secret123'));
    }

    public function test_skips_product_with_no_base_url(): void
    {
        Http::fake();

        config(['webhook.cron_triggers' => [
            'disabled-product' => [
                'base_url' => null,
                'key' => 'secret',
                'endpoints' => ['/cron'],
                'stagger_seconds' => 0,
                'offset_seconds' => 0,
            ],
        ]]);

        CronTrigger::run();

        Http::assertSentCount(0);
    }

    public function test_logs_failures_gracefully(): void
    {
        Http::fake([
            '*' => Http::response('error', 500),
        ]);

        config(['webhook.cron_triggers' => [
            'failing-product' => [
                'base_url' => 'https://broken.example.com',
                'key' => 'key',
                'endpoints' => ['/cron'],
                'stagger_seconds' => 0,
                'offset_seconds' => 0,
            ],
        ]]);

        // Should not throw
        CronTrigger::run();

        Http::assertSentCount(1);
    }

    public function test_handles_empty_config(): void
    {
        Http::fake();
        config(['webhook.cron_triggers' => []]);

        CronTrigger::run();

        Http::assertSentCount(0);
    }
}
```

**Step 2: Run test to verify it fails**

```bash
vendor/bin/phpunit tests/Feature/CronTriggerTest.php
```

Expected: FAIL — class `Core\Webhook\CronTrigger` not found.

**Step 3: Create the action**

```php
<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Webhook;

use Core\Actions\Action;
use Core\Actions\Scheduled;
use Illuminate\Support\Facades\Http;

#[Scheduled(frequency: 'everyMinute', withoutOverlapping: true, runInBackground: true)]
class CronTrigger
{
    use Action;

    public function handle(): void
    {
        $triggers = config('webhook.cron_triggers', []);

        foreach ($triggers as $product => $config) {
            if (empty($config['base_url'])) {
                continue;
            }

            $baseUrl = rtrim($config['base_url'], '/');
            $key = $config['key'] ?? '';
            $stagger = (int) ($config['stagger_seconds'] ?? 0);
            $offset = (int) ($config['offset_seconds'] ?? 0);

            if ($offset > 0) {
                usleep($offset * 1_000_000);
            }

            foreach ($config['endpoints'] ?? [] as $i => $endpoint) {
                if ($i > 0 && $stagger > 0) {
                    usleep($stagger * 1_000_000);
                }

                $url = $baseUrl . $endpoint . '?key=' . $key;

                try {
                    Http::timeout(30)->get($url);
                } catch (\Throwable $e) {
                    logger()->warning("Cron trigger failed for {$product}{$endpoint}: {$e->getMessage()}");
                }
            }
        }
    }
}
```

**Step 4: Run tests**

```bash
vendor/bin/phpunit tests/Feature/CronTriggerTest.php
```

Expected: All 6 tests pass.

**Step 5: Commit**

```bash
git add src/Core/Webhook/CronTrigger.php tests/Feature/CronTriggerTest.php
git commit -m "feat(webhook): add CronTrigger action — replaces 4 Docker cron containers"
```

---

### Task 8: Final verification

**Step 1: Verify file structure**

```bash
cd /Users/snider/Code/core/php
find src/Core/Webhook/ -type f | sort
find database/migrations/ -name '*webhook*' | sort
find tests/Feature/ -name '*Webhook*' -o -name '*CronTrigger*' | sort
```

Expected:
```
src/Core/Webhook/Boot.php
src/Core/Webhook/CronTrigger.php
src/Core/Webhook/WebhookCall.php
src/Core/Webhook/WebhookController.php
src/Core/Webhook/WebhookReceived.php
src/Core/Webhook/WebhookVerifier.php
src/Core/Webhook/config.php
database/migrations/2026_03_12_000001_create_webhook_calls_table.php
tests/Feature/WebhookCallTest.php
tests/Feature/WebhookControllerTest.php
tests/Feature/CronTriggerTest.php
```

**Step 2: Run all webhook tests**

```bash
vendor/bin/phpunit tests/Feature/WebhookCallTest.php tests/Feature/WebhookControllerTest.php tests/Feature/CronTriggerTest.php
```

Expected: All tests pass (5 + 9 + 6 = 20 tests).

**Step 3: Run lint**

```bash
./vendor/bin/pint --test src/Core/Webhook/ tests/Feature/WebhookCallTest.php tests/Feature/WebhookControllerTest.php tests/Feature/CronTriggerTest.php
```

Expected: Clean.

**Step 4: Verify strict types in all files**

```bash
grep -rL 'declare(strict_types=1)' src/Core/Webhook/ database/migrations/*webhook*
```

Expected: No output (all files have strict types).

**Step 5: Final commit if any lint fixes needed**

```bash
git add -A src/Core/Webhook/ tests/Feature/ database/migrations/
git status
```

If clean, done. If fixes needed, commit them.
