# MCP Agent Tools

This directory contains MCP (Model Context Protocol) tool implementations for the agent orchestration system. All tools extend `AgentTool` and integrate with the `ToolDependency` system to declare and validate their execution prerequisites.

## Directory Structure

```
Mcp/Tools/Agent/
├── AgentTool.php              # Base class — extend this for all new tools
├── Contracts/
│   └── AgentToolInterface.php # Tool contract
├── Content/                   # Content generation tools
├── Phase/                     # Plan phase management tools
├── Plan/                      # Work plan CRUD tools
├── Session/                   # Agent session lifecycle tools
├── State/                     # Shared workspace state tools
├── Task/                      # Task status and tracking tools
└── Template/                  # Template listing and application tools
```

## ToolDependency System

`ToolDependency` (from `Core\Mcp\Dependencies\ToolDependency`) lets a tool declare what must be true in the execution context before it runs. The `AgentToolRegistry` validates these automatically — the tool's `handle()` method is never called if a dependency is unmet.

### How It Works

1. A tool declares its dependencies in a `dependencies()` method returning `ToolDependency[]`.
2. When the tool is registered, `AgentToolRegistry::register()` passes those dependencies to `ToolDependencyService`.
3. On each call, `AgentToolRegistry::execute()` calls `ToolDependencyService::validateDependencies()` before invoking `handle()`.
4. If any required dependency fails, a `MissingDependencyException` is thrown and the tool is never called.
5. After a successful call, `ToolDependencyService::recordToolCall()` logs the execution for audit purposes.

### Dependency Types

#### `contextExists` — Require a context field

Validates that a key is present in the `$context` array passed at execution time. Use this for multi-tenant isolation fields like `workspace_id` that come from API key authentication.

```php
ToolDependency::contextExists('workspace_id', 'Workspace context required')
```

Mark a dependency optional with `->asOptional()` when the tool can work without it (e.g. the value can be inferred from another argument):

```php
// SessionStart: workspace can be inferred from the plan if plan_slug is provided
ToolDependency::contextExists('workspace_id', 'Workspace context required (or provide plan_slug)')
    ->asOptional()
```

#### `sessionState` — Require an active session

Validates that a session is active. Use this for tools that must run within an established session context.

```php
ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.')
```

#### `entityExists` — Require a database entity

Validates that an entity exists in the database before the tool runs. The `arg_key` maps to the tool argument that holds the entity identifier.

```php
ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug'])
```

## Context Requirements

The `$context` array is injected into every tool's `handle(array $args, array $context)` call. Context is set by API key authentication middleware — tools should never hardcode or fall back to default values.

| Key | Type | Set by | Used by |
|-----|------|--------|---------|
| `workspace_id` | `string\|int` | API key auth middleware | All workspace-scoped tools |
| `session_id` | `string` | Client (from `session_start` response) | Session-dependent tools |

**Multi-tenant safety:** Always validate `workspace_id` in `handle()` as a defence-in-depth measure, even when a `contextExists` dependency is declared. Use `forWorkspace($workspaceId)` scopes on all queries.

```php
$workspaceId = $context['workspace_id'] ?? null;
if ($workspaceId === null) {
    return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key. See: https://host.uk.com/ai');
}

$plan = AgentPlan::forWorkspace($workspaceId)->where('slug', $slug)->first();
```

## Creating a New Tool

### 1. Create the class

Place the file in the appropriate subdirectory and extend `AgentTool`:

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

class PlanPublish extends AgentTool
{
    protected string $category = 'plan';

    protected array $scopes = ['write']; // 'read' or 'write'

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'plan_publish'; // snake_case; must be unique across all tools
    }

    public function description(): string
    {
        return 'Publish a draft plan, making it active';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Plan slug identifier',
                ],
            ],
            'required' => ['plan_slug'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $planSlug = $this->requireString($args, 'plan_slug', 255);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. See: https://host.uk.com/ai');
        }

        $plan = AgentPlan::forWorkspace($workspaceId)->where('slug', $planSlug)->first();

        if (! $plan) {
            return $this->error("Plan not found: {$planSlug}");
        }

        $plan->update(['status' => 'active']);

        return $this->success(['plan' => ['slug' => $plan->slug, 'status' => $plan->status]]);
    }
}
```

### 2. Register the tool

Add it to the tool registration list in the package boot sequence (see `Boot.php` and the `McpToolsRegistering` event handler).

### 3. Write tests

Add a Pest test file under `Tests/` covering success and failure paths, including missing dependency scenarios.

## AgentTool Base Class Reference

### Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `$category` | `string` | `'general'` | Groups tools in the registry |
| `$scopes` | `string[]` | `['read']` | API key scopes required to call this tool |
| `$timeout` | `?int` | `null` | Per-tool timeout override in seconds (null uses config default of 30s) |

### Argument Helpers

All helpers throw `\InvalidArgumentException` on failure. Catch it in `handle()` and return `$this->error()`.

| Method | Description |
|--------|-------------|
| `requireString($args, $key, $maxLength, $label)` | Required string with optional max length |
| `requireInt($args, $key, $min, $max, $label)` | Required integer with optional bounds |
| `requireArray($args, $key, $label)` | Required array |
| `requireEnum($args, $key, $allowed, $label)` | Required string constrained to allowed values |
| `optionalString($args, $key, $default, $maxLength)` | Optional string |
| `optionalInt($args, $key, $default, $min, $max)` | Optional integer |
| `optionalEnum($args, $key, $allowed, $default)` | Optional enum string |
| `optional($args, $key, $default)` | Optional value of any type |

### Response Helpers

```php
return $this->success(['key' => 'value']); // merges ['success' => true]
return $this->error('Something went wrong');
return $this->error('Resource locked', 'resource_locked'); // with error code
```

### Circuit Breaker

Wrap calls to external services with `withCircuitBreaker()` for fault tolerance:

```php
return $this->withCircuitBreaker(
    'agentic',                                                    // service name
    fn () => $this->doWork(),                                     // operation
    fn () => $this->error('Service unavailable', 'service_unavailable') // fallback
);
```

If no fallback is provided and the circuit is open, `error()` is returned automatically.

### Timeout Override

For long-running tools (e.g. content generation), override the timeout:

```php
protected ?int $timeout = 300; // 5 minutes
```

## Dependency Resolution Order

Dependencies are validated in the order they are returned from `dependencies()`. All required dependencies must pass before the tool runs. Optional dependencies are checked but do not block execution.

Recommended declaration order:

1. `contextExists('workspace_id', ...)` — tenant isolation first
2. `sessionState('session_id', ...)` — session presence second
3. `entityExists(...)` — entity existence last (may query DB)

## Troubleshooting

### "Workspace context required"

The `workspace_id` key is missing from the execution context. This is injected by the API key authentication middleware. Causes:

- Request is unauthenticated or the API key is invalid.
- The API key has no workspace association.
- Dependency validation was bypassed but the tool checks it internally.

**Fix:** Authenticate with a valid API key. See https://host.uk.com/ai.

### "Active session required. Call session_start first."

The `session_id` context key is missing. The tool requires an active session.

**Fix:** Call `session_start` before calling session-dependent tools. Pass the returned `session_id` in the context of all subsequent calls.

### "Plan must exist" / "Plan not found"

The `plan_slug` argument does not match any plan. Either the plan was never created, the slug is misspelled, or the plan belongs to a different workspace.

**Fix:** Call `plan_list` to find valid slugs, then retry.

### "Permission denied: API key missing scope"

The API key does not have the required scope (`read` or `write`) for the tool.

**Fix:** Issue a new API key with the correct scopes, or use an existing key that has the required permissions.

### "Unknown tool: {name}"

The tool name does not match any registered tool.

**Fix:** Check `plan_list` / MCP tool discovery endpoint for the exact tool name. Names are snake_case.

### `MissingDependencyException` in logs

A required dependency was not met and the framework threw before calling `handle()`. The exception message will identify which dependency failed.

**Fix:** Inspect the `context` passed to `execute()`. Ensure required keys are present and the relevant entity exists.
