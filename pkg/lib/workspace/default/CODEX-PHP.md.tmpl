# CODEX.md — PHP / CorePHP

Instructions for Codex when working with PHP code in this workspace.

## CorePHP Framework

This project uses CorePHP (`core/php`) as its foundation. CorePHP is a Laravel package that provides:
- Event-driven module loading (modules only load when their events fire)
- Multi-tenant isolation via `BelongsToWorkspace` trait
- Actions pattern for single-purpose business logic
- Lifecycle events for route/panel/command registration

## Architecture

### Module Pattern

```php
// app/Mod/{Name}/Boot.php
class Boot extends ServiceProvider
{
    public static array $listens = [
        WebRoutesRegistering::class => 'onWebRoutes',
        AdminPanelBooting::class => 'onAdmin',
    ];
}
```

### Website Pattern

```php
// app/Website/{Name}/Boot.php
class Boot extends ServiceProvider
{
    public static array $domains = [
        '/^api\.lthn\.(ai|test|sh)$/',
    ];

    public static array $listens = [
        DomainResolving::class => 'onDomain',
        WebRoutesRegistering::class => 'onWebRoutes',
        ApiRoutesRegistering::class => 'onApiRoutes',
    ];
}
```

### Lifecycle Events

| Event | Purpose |
|-------|---------|
| `DomainResolving` | Match domain → register website module |
| `WebRoutesRegistering` | Public web routes (sessions, CSRF, Vite) |
| `ApiRoutesRegistering` | Stateless API routes |
| `AdminPanelBooting` | Admin panel resources |
| `ClientRoutesRegistering` | Authenticated SaaS client routes |
| `ConsoleBooting` | Artisan commands |
| `McpToolsRegistering` | MCP tool handlers |

### Actions Pattern

```php
use Core\Actions\Action;

class CreateOrder
{
    use Action;

    public function handle(User $user, array $data): Order
    {
        return Order::create($data);
    }
}
// Usage: CreateOrder::run($user, $validated);
```

### Multi-Tenant Isolation

```php
use Core\Mod\Tenant\Concerns\BelongsToWorkspace;

class Memory extends Model
{
    use BelongsToWorkspace;
    // Auto-scopes queries to current workspace
    // Auto-assigns workspace_id on create
}
```

## Mandatory Patterns

### Strict Types — every PHP file

```php
<?php

declare(strict_types=1);
```

### Type Hints — all parameters and return types

```php
// WRONG
function process($data) { }

// CORRECT
function process(array $data): JsonResponse { }
```

### UK English in all comments and strings

```
colour       not color
organisation not organization
initialise   not initialize
serialise    not serialize
centre       not center
```

### Testing — PHPUnit with Orchestra Testbench

```php
class BrainServiceTest extends TestCase
{
    public function test_remember_stores_memory(): void { }
    public function test_remember_validates_input(): void { }
    public function test_recall_returns_ranked_results(): void { }
    public function test_recall_filters_by_org(): void { }
}
```

### Formatting — Laravel Pint (PSR-12)

```bash
./vendor/bin/pint --dirty    # Format changed files only
```

## UI

- **Flux Pro** — component library (NOT vanilla Livewire/Alpine)
- **Font Awesome Pro** — icons (NOT Heroicons)
- **Livewire 3** — server-driven components
- **Alpine.js** — client-side interactivity

## What NOT to do

- Don't use American English spellings
- Don't use Heroicons (use Font Awesome Pro)
- Don't use vanilla Blade components where Flux Pro has an equivalent
- Don't create loose route files — routes belong in Boot.php via lifecycle events
- Don't bypass BelongsToWorkspace — all tenant data must be scoped
- Don't use raw DB queries where Eloquent works

## Build & Test

```bash
composer install
composer test
./vendor/bin/pint --dirty
php artisan test --filter=SpecificTest
```
