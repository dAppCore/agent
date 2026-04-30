---
name: laravel
description: Use when working on Laravel code in core-* PHP packages
---

# Laravel Patterns for Host UK

## Module Structure
All modules follow event-driven loading via Boot class.

## Actions Pattern
Use single-purpose Action classes:
```php
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

## Multi-Tenancy
Always use BelongsToWorkspace trait for tenant-scoped models.

## UI Components
- Use Flux Pro components (not vanilla Alpine)
- Use Font Awesome Pro (not Heroicons)
- UK English spellings (colour, organisation)

## Commands
```bash
core php test              # Run Pest tests
core php fmt --fix         # Format with Pint
core php stan              # PHPStan analysis
```
