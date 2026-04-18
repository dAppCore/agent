---
title: Architecture
description: Technical architecture of the core-agentic package
updated: 2026-01-29
---

# Architecture

The `core-agentic` package provides AI agent orchestration infrastructure for the Host UK platform. It enables multi-agent collaboration, persistent task tracking, and unified access to multiple AI providers.

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCP Protocol Layer                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Plan     │ │ Phase    │ │ Session  │ │ State    │ ... tools │
│  │ Tools    │ │ Tools    │ │ Tools    │ │ Tools    │           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
└───────┼────────────┼────────────┼────────────┼──────────────────┘
        │            │            │            │
┌───────┴────────────┴────────────┴────────────┴──────────────────┐
│                      AgentToolRegistry                           │
│  - Tool registration and discovery                               │
│  - Permission checking (API key scopes)                          │
│  - Dependency validation                                         │
│  - Circuit breaker integration                                   │
└──────────────────────────────────────────────────────────────────┘
        │
┌───────┴──────────────────────────────────────────────────────────┐
│                         Core Services                             │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │ AgenticManager │  │ AgentApiKey    │  │ PlanTemplate   │     │
│  │ (AI Providers) │  │ Service        │  │ Service        │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │ IpRestriction  │  │ Content        │  │ AgentSession   │     │
│  │ Service        │  │ Service        │  │ Service        │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
└──────────────────────────────────────────────────────────────────┘
        │
┌───────┴──────────────────────────────────────────────────────────┐
│                         Data Layer                                │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐│
│  │ AgentPlan   │ │ AgentPhase  │ │ AgentSession│ │ AgentApiKey ││
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘│
│  ┌─────────────┐ ┌─────────────┐                                 │
│  │ Workspace   │ │ Task        │                                 │
│  │ State       │ │             │                                 │
│  └─────────────┘ └─────────────┘                                 │
└──────────────────────────────────────────────────────────────────┘
```

## Core Concepts

### Agent Plans

Plans represent structured work with phases, tasks, and progress tracking. They persist across agent sessions, enabling handoff between different AI models or instances.

```
AgentPlan
├── slug (unique identifier)
├── title
├── status (draft → active → completed/archived)
├── current_phase
└── phases[] (AgentPhase)
    ├── name
    ├── tasks[]
    │   ├── name
    │   └── status
    ├── dependencies[]
    └── checkpoints[]
```

**Lifecycle:**
1. Created via MCP tool or template
2. Activated to begin work
3. Phases started/completed in order
4. Plan auto-completes when all phases done
5. Archived for historical reference

### Agent Sessions

Sessions track individual work periods. They enable context recovery when an agent's context window resets or when handing off to another agent.

```
AgentSession
├── session_id (prefixed unique ID)
├── agent_type (opus/sonnet/haiku)
├── status (active/paused/completed/failed)
├── work_log[] (chronological actions)
├── artifacts[] (files created/modified)
├── context_summary (current state)
└── handoff_notes (for next agent)
```

**Handoff Flow:**
1. Session logs work as it progresses
2. Before context ends, agent calls `session_handoff`
3. Handoff notes capture summary, next steps, blockers
4. Next agent calls `session_resume` to continue
5. Resume session inherits context from previous

### Workspace State

Key-value state storage shared between sessions and plans. Enables agents to persist decisions, configurations, and intermediate results.

```
WorkspaceState
├── key (namespaced identifier)
├── value (any JSON-serialisable data)
├── type (json/markdown/code/reference)
└── category (for organisation)
```

## MCP Tool Architecture

All MCP tools extend the `AgentTool` base class which provides:

### Input Validation

```php
protected function requireString(array $args, string $key, ?int $maxLength = null): string
protected function optionalInt(array $args, string $key, ?int $default = null): ?int
protected function requireEnum(array $args, string $key, array $allowed): string
```

### Circuit Breaker Protection

```php
return $this->withCircuitBreaker('agentic', function () {
    // Database operations that could fail
    return AgentPlan::where('slug', $slug)->first();
}, fn () => $this->error('Service unavailable', 'circuit_open'));
```

### Dependency Declaration

```php
public function dependencies(): array
{
    return [
        ToolDependency::contextExists('workspace_id', 'Workspace required'),
        ToolDependency::toolCalled('session_start', 'Start session first'),
    ];
}
```

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| `plan` | plan_create, plan_get, plan_list, plan_update_status, plan_archive | Work plan management |
| `phase` | phase_get, phase_update_status, phase_add_checkpoint | Phase operations |
| `session` | session_start, session_end, session_log, session_handoff, session_resume, session_replay | Session tracking |
| `state` | state_get, state_set, state_list | Persistent state |
| `task` | task_update, task_toggle | Task completion |
| `template` | template_list, template_preview, template_create_plan | Plan templates |
| `content` | content_generate, content_batch_generate, content_brief_create | Content generation |

## AI Provider Abstraction

The `AgenticManager` provides unified access to multiple AI providers:

```php
$ai = app(AgenticManager::class);

// Use specific provider
$response = $ai->claude()->generate($system, $user);
$response = $ai->gemini()->generate($system, $user);
$response = $ai->openai()->generate($system, $user);

// Use by name (for configuration-driven selection)
$response = $ai->provider('gemini')->generate($system, $user);
```

### Provider Interface

All providers implement `AgenticProviderInterface`:

```php
interface AgenticProviderInterface
{
    public function generate(string $systemPrompt, string $userPrompt, array $config = []): AgenticResponse;
    public function stream(string $systemPrompt, string $userPrompt, array $config = []): Generator;
    public function name(): string;
    public function defaultModel(): string;
    public function isAvailable(): bool;
}
```

### Response Object

```php
class AgenticResponse
{
    public string $content;
    public string $model;
    public int $inputTokens;
    public int $outputTokens;
    public int $durationMs;
    public ?string $stopReason;
    public array $raw;

    public function estimateCost(): float;
}
```

## Authentication

### API Key Flow

```
Request → AgentApiAuth Middleware → AgentApiKeyService::authenticate()
                                            │
                                            ├── Validate key (SHA-256 hash lookup)
                                            ├── Check revoked/expired
                                            ├── Validate IP whitelist
                                            ├── Check permissions
                                            ├── Check rate limit
                                            └── Record usage
```

### Permission Model

```php
// Permission constants
AgentApiKey::PERM_PLANS_READ      // 'plans.read'
AgentApiKey::PERM_PLANS_WRITE     // 'plans.write'
AgentApiKey::PERM_SESSIONS_WRITE  // 'sessions.write'
// etc.

// Check permissions
$key->hasPermission('plans.write');
$key->hasAllPermissions(['plans.read', 'sessions.write']);
```

### IP Restrictions

API keys can optionally restrict access by IP:

- Individual IPv4/IPv6 addresses
- CIDR notation (e.g., `192.168.1.0/24`)
- Mixed whitelist

## Event-Driven Boot

The module uses the Core framework's event-driven lazy loading:

```php
class Boot extends ServiceProvider
{
    public static array $listens = [
        AdminPanelBooting::class => 'onAdminPanel',
        ConsoleBooting::class => 'onConsole',
        McpToolsRegistering::class => 'onMcpTools',
    ];
}
```

This ensures:
- Views only loaded when admin panel boots
- Commands only registered when console boots
- MCP tools only registered when MCP module initialises

## Multi-Tenancy

All data is workspace-scoped via the `BelongsToWorkspace` trait:

- Queries auto-scoped to current workspace
- Creates auto-assign workspace_id
- Cross-tenant queries throw `MissingWorkspaceContextException`

## File Structure

```
core-agentic/
├── Boot.php                    # Service provider with event handlers
├── config.php                  # Module configuration
├── Migrations/                 # Database schema
├── Models/                     # Eloquent models
│   ├── AgentPlan.php
│   ├── AgentPhase.php
│   ├── AgentSession.php
│   ├── AgentApiKey.php
│   └── WorkspaceState.php
├── Services/                   # Business logic
│   ├── AgenticManager.php      # AI provider orchestration
│   ├── AgentApiKeyService.php  # API key management
│   ├── IpRestrictionService.php
│   ├── PlanTemplateService.php
│   ├── ContentService.php
│   ├── ClaudeService.php
│   ├── GeminiService.php
│   └── OpenAIService.php
├── Mcp/
│   ├── Tools/Agent/            # MCP tool implementations
│   │   ├── AgentTool.php       # Base class
│   │   ├── Plan/
│   │   ├── Phase/
│   │   ├── Session/
│   │   ├── State/
│   │   └── ...
│   ├── Prompts/                # MCP prompt definitions
│   └── Servers/                # MCP server configurations
├── Middleware/
│   └── AgentApiAuth.php        # API authentication
├── Controllers/
│   └── ForAgentsController.php # Agent discovery endpoint
├── View/
│   ├── Blade/admin/            # Admin panel views
│   └── Modal/Admin/            # Livewire components
├── Jobs/                       # Queue jobs
├── Console/Commands/           # Artisan commands
└── Tests/                      # Pest test suites
```

## Dependencies

- `host-uk/core` - Event system, base classes
- `host-uk/core-tenant` - Workspace, BelongsToWorkspace trait
- `host-uk/core-mcp` - MCP infrastructure, CircuitBreaker
