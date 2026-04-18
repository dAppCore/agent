---
title: Development Guide
description: How to build, test, and contribute to core/agent — covering Go packages, PHP tests, MCP servers, Claude Code plugins, and coding standards.
---

# Development Guide

Core Agent is a polyglot repository. Go and PHP live side by side, each with their own toolchain. The `core` CLI wraps both and is the primary interface for all development tasks.


## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26+ | Go packages, CLI commands, MCP servers |
| PHP | 8.2+ | Laravel package, Pest tests |
| Composer | 2.x | PHP dependency management |
| `core` CLI | latest | Wraps Go and PHP toolchains; enforced by plugin hooks |
| `jq` | any | Used by shell hooks for JSON parsing |

### Go Workspace

The module is `forge.lthn.ai/core/agent`. It participates in a Go workspace (`go.work`) that resolves all `forge.lthn.ai/core/*` dependencies locally. After cloning, ensure the workspace file includes a `use` entry for this module:

```
use ./core/agent
```

Then run `go work sync` from the workspace root.

### PHP Dependencies

```bash
composer install
```

The Composer package is `lthn/agent`. It depends on `lthn/php` (the foundation framework) at runtime, and on `orchestra/testbench`, `pestphp/pest`, and `livewire/livewire` for development.


## Building

### Go Packages

There is no standalone binary produced by this module. The Go packages (`pkg/lifecycle/`, `pkg/loop/`, `pkg/orchestrator/`, `pkg/jobrunner/`) are libraries imported by the `core` CLI binary (built from `forge.lthn.ai/core/cli`).

To verify the packages compile:

```bash
core go build
```

### MCP Servers

Two MCP servers live in this repository:

**Stdio server** (`cmd/mcp/`) — a standalone binary using `mcp-go`:

```bash
cd cmd/mcp && go build -o agent-mcp .
```

It exposes four tools (`marketplace_list`, `marketplace_plugin_info`, `core_cli`, `ethics_check`) and is invoked by Claude Code over stdio.

**HTTP server** (`google/mcp/`) — a plain `net/http` server on port 8080:

```bash
cd google/mcp && go build -o google-mcp .
./google-mcp
```

It exposes `core_go_test`, `core_dev_health`, and `core_dev_commit` as POST endpoints.


## Testing

### Go Tests

```bash
# Run all Go tests
core go test

# Run a single test by name
core go test --run TestMemoryRegistry_Register_Good

# Full QA pipeline (fmt + vet + lint + test)
core go qa

# QA with race detector, vulnerability scan, and security checks
core go qa full

# Generate and view test coverage
core go cov
core go cov --open
```

Tests use `testify/assert` and `testify/require`. The naming convention is:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy-path tests |
| `_Bad` | Expected error conditions |
| `_Ugly` | Panic and edge cases |

The test suite is substantial: ~65 test files across the Go packages, covering lifecycle (registry, allowance, dispatcher, router, events, client, brain, context), jobrunner (poller, journal, handlers, Forgejo source), loop (engine, parsing, prompts, tools), and orchestrator (Clotho, config, security).

### PHP Tests

```bash
# Run the full Pest suite
composer test

# Run a specific test file
./vendor/bin/pest --filter=AgenticManagerTest

# Fix code style
composer lint
```

The PHP test suite uses Pest with Orchestra Testbench for package testing. Feature tests use `RefreshDatabase` for clean database state. The test configuration lives in `src/php/tests/Pest.php`:

```php
uses(TestCase::class)->in('Feature', 'Unit', 'UseCase');
uses(RefreshDatabase::class)->in('Feature');
```

Helper functions for test setup:

```php
// Create a workspace for testing
$workspace = createWorkspace();

// Create an API key for testing
$key = createApiKey($workspace, 'Test Key', ['plan:read'], 100);
```

The test suite includes:

- **Unit tests** (`src/php/tests/Unit/`): ClaudeService, GeminiService, OpenAIService, AgenticManager, AgentToolRegistry, AgentDetection, stream parsing, retry logic
- **Feature tests** (`src/php/tests/Feature/`): AgentPlan, AgentPhase, AgentSession, AgentApiKey, ForgejoService, security, workspace state, plan retention, prompt versioning, content service, Forgejo actions, scan-for-work
- **Livewire tests** (`src/php/tests/Feature/Livewire/`): Dashboard, Plans, PlanDetail, Sessions, SessionDetail, ApiKeys, Templates, ToolAnalytics, ToolCalls, Playground, RequestLog
- **Use-case tests** (`src/php/tests/UseCase/`): AdminPanelBasic


## Formatting and Linting

### Go

```bash
# Format all Go files
core go fmt

# Run the linter
core go lint

# Run go vet
core go vet
```

### PHP

```bash
# Fix code style (Laravel Pint, PSR-12)
composer lint

# Format only changed files
./vendor/bin/pint --dirty
```

### Automatic Formatting

The `code` plugin includes PostToolUse hooks that auto-format files after every edit:

- **Go files**: `scripts/go-format.sh` runs `gofmt` on any edited `.go` file
- **PHP files**: `scripts/php-format.sh` runs `pint` on any edited `.php` file
- **Debug check**: `scripts/check-debug.sh` warns about `dd()`, `dump()`, `fmt.Println()`, and similar statements left in code


## Claude Code Plugins

### Installing

Install all five plugins at once:

```bash
claude plugin add host-uk/core-agent
```

Or install individual plugins:

```bash
claude plugin add host-uk/core-agent/claude/code
claude plugin add host-uk/core-agent/claude/review
claude plugin add host-uk/core-agent/claude/verify
claude plugin add host-uk/core-agent/claude/qa
claude plugin add host-uk/core-agent/claude/ci
```

### Plugin Architecture

Each plugin lives in `claude/<name>/` and contains:

```
claude/<name>/
├── .claude-plugin/
│   └── plugin.json          # Plugin metadata (name, version, description)
├── hooks.json                # Hook declarations (optional)
├── hooks/                    # Hook scripts (optional)
├── scripts/                  # Supporting scripts (optional)
├── commands/                 # Slash command definitions (*.md files)
└── skills/                   # Skill definitions (optional)
```

The marketplace registry at `.claude-plugin/marketplace.json` lists all five plugins with their source paths and versions.

### Available Commands

| Plugin | Command | Purpose |
|--------|---------|---------|
| code | `/code:remember <fact>` | Save context that persists across compaction |
| code | `/code:yes <task>` | Auto-approve mode with commit requirement |
| code | `/code:qa` | Run QA pipeline |
| review | `/review:review [range]` | Code review on staged changes or commits |
| review | `/review:security` | Security-focused review |
| review | `/review:pr` | Pull request review |
| verify | `/verify:verify [--quick\|--full]` | Verify work is complete |
| verify | `/verify:ready` | Check if work is ready to ship |
| verify | `/verify:tests` | Verify test coverage |
| qa | `/qa:qa` | Iterative QA fix loop (runs until all checks pass) |
| qa | `/qa:fix <issue>` | Fix a specific QA issue |
| qa | `/qa:check` | Run checks without fixing |
| qa | `/qa:lint` | Lint check only |
| ci | `/ci:ci [status\|run\|logs\|fix]` | CI status and management |
| ci | `/ci:workflow <type>` | Generate GitHub Actions workflows |
| ci | `/ci:fix` | Fix CI failures |
| ci | `/ci:run` | Trigger a CI run |
| ci | `/ci:status` | Show CI status |

### Hook System

The `code` plugin defines hooks in `claude/code/hooks.json` that fire at different points in the Claude Code lifecycle:

**PreToolUse** (before a tool runs):
- `prefer-core.sh` on `Bash` tool: blocks destructive commands (`rm -rf`, `sed -i`, `xargs rm`, `find -exec rm`, `grep -l | ...`) and enforces `core` CLI usage (blocks raw `go test`, `go build`, `composer test`, `golangci-lint`)
- `block-docs.sh` on `Write` tool: prevents creation of random `.md` files

**PostToolUse** (after a tool completes):
- `go-format.sh` on `Edit` for `.go` files: auto-runs `gofmt`
- `php-format.sh` on `Edit` for `.php` files: auto-runs `pint`
- `check-debug.sh` on `Edit`: warns about debug statements
- `post-commit-check.sh` on `Bash` for `git commit`: warns about uncommitted work

**PreCompact** (before context compaction):
- `pre-compact.sh`: saves session state to prevent amnesia

**SessionStart** (when a session begins):
- `session-start.sh`: restores recent session context

### Testing Hooks Locally

```bash
echo '{"tool_input": {"command": "rm -rf /"}}' | bash ./claude/code/hooks/prefer-core.sh
# Output: {"decision": "block", "message": "BLOCKED: Recursive delete is not allowed..."}

echo '{"tool_input": {"command": "core go test"}}' | bash ./claude/code/hooks/prefer-core.sh
# Output: {"decision": "approve"}
```

Hook scripts read JSON on stdin and output a JSON object with `decision` (`approve` or `block`) and an optional `message`.

### Adding a New Plugin

1. Create the directory structure:
   ```
   claude/<name>/
   ├── .claude-plugin/
   │   └── plugin.json
   └── commands/
       └── <command>.md
   ```

2. Write `plugin.json`:
   ```json
   {
     "name": "<name>",
     "description": "What this plugin does",
     "version": "0.1.0",
     "author": {
       "name": "Host UK",
       "email": "hello@host.uk.com"
     },
     "license": "EUPL-1.2"
   }
   ```

3. Add command files as Markdown (`.md`) in `commands/`. The filename becomes the command name.

4. Register the plugin in `.claude-plugin/marketplace.json`:
   ```json
   {
     "name": "<name>",
     "source": "./claude/<name>",
     "description": "Short description",
     "version": "0.1.0"
   }
   ```

### Codex Plugins

The `codex/` directory mirrors the Claude plugin structure for OpenAI Codex. It contains additional plugins beyond the Claude five: `ethics`, `guardrails`, `perf`, `issue`, `coolify`, `awareness`, `api`, and `collect`. Each follows the same pattern with `.codex-plugin/plugin.json` and optional hooks, commands, and skills.


## Adding Go Functionality

### New Package

Create a directory under `pkg/`. Follow the existing convention:

```
pkg/<name>/
├── types.go           # Public types and interfaces
├── <implementation>.go
└── <implementation>_test.go
```

Import the package from other modules as `forge.lthn.ai/core/agent/pkg/<name>`.

### New CLI Command

Commands live in `cmd/`. Each command directory registers itself into the `core` binary via the CLI framework:

```go
package mycmd

import (
    "forge.lthn.ai/core/cli"
    "github.com/spf13/cobra"
)

func AddCommands(parent *cobra.Command) {
    parent.AddCommand(&cobra.Command{
        Use:   "mycommand",
        Short: "What it does",
        RunE: func(cmd *cobra.Command, args []string) error {
            // implementation
            return nil
        },
    })
}
```

Registration into the `core` binary happens in the CLI module, not here. This module exports the `AddCommands` function and the CLI module calls it.

### New MCP Tool (stdio server)

Tools are added in `cmd/mcp/server.go`. Each tool needs:

1. A `mcp.Tool` definition with name, description, and input schema
2. A handler function with signature `func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)`
3. Registration via `s.AddTool(tool, handler)` in the `newServer()` function

### New MCP Tool (HTTP server)

Tools for the Google MCP server are plain HTTP handlers in `google/mcp/main.go`. Add a handler function and register it with `http.HandleFunc`.


## Adding PHP Functionality

### New Model

Create in `src/php/Models/`. All models use the `Core\Mod\Agentic\Models` namespace:

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Model;

class MyModel extends Model
{
    protected $fillable = ['name', 'status'];
}
```

### New Action

Actions follow the single-purpose pattern in `src/php/Actions/`:

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions;

use Core\Mod\Agentic\Concerns\Action;

class DoSomething
{
    use Action;

    public function handle(string $input): string
    {
        return strtoupper($input);
    }
}

// Usage: DoSomething::run('hello');
```

### New Controller

API controllers go in `src/php/Controllers/`. Routes are registered in `src/php/Routes/api.php`, which is loaded by the service provider's `onApiRoutes` handler.

### New Artisan Command

Console commands go in `src/php/Console/Commands/`. Register them in `Boot::onConsole()`:

```php
public function onConsole(ConsoleBooting $event): void
{
    $event->command(Console\Commands\MyCommand::class);
    // ...existing commands...
}
```

### New Livewire Component

Admin panel components go in `src/php/View/Modal/Admin/`. Blade views go in `src/php/View/Blade/admin/`. Register the component in `Boot::onAdminPanel()`:

```php
$event->livewire('agentic.admin.my-component', View\Modal\Admin\MyComponent::class);
```


## Writing Tests

### Go Test Conventions

Use the `_Good` / `_Bad` / `_Ugly` suffix pattern:

```go
func TestMyFunction_Good(t *testing.T) {
    // Happy path — expected input produces expected output
    result := MyFunction("valid")
    assert.Equal(t, "expected", result)
}

func TestMyFunction_Bad_EmptyInput(t *testing.T) {
    // Expected failure — invalid input returns error
    _, err := MyFunction("")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "input required")
}

func TestMyFunction_Ugly_NilPointer(t *testing.T) {
    // Edge case — nil receiver, concurrent access, etc.
    assert.Panics(t, func() { MyFunction(nil) })
}
```

Always use `require` for preconditions (stops test immediately on failure) and `assert` for verifications (continues to report all failures).

### PHP Test Conventions

Use Pest syntax:

```php
it('creates a plan with phases', function () {
    $workspace = createWorkspace();
    $plan = AgentPlan::factory()->create(['workspace_id' => $workspace->id]);

    expect($plan)->toBeInstanceOf(AgentPlan::class);
    expect($plan->workspace_id)->toBe($workspace->id);
});

it('rejects invalid input', function () {
    $this->postJson('/v1/plans', [])
        ->assertStatus(422);
});
```

Feature tests get `RefreshDatabase` automatically. Unit tests should not touch the database.


## Coding Standards

### Language

Use **UK English** throughout: colour, organisation, centre, licence, behaviour, catalogue. Never American spellings.

### PHP

- `declare(strict_types=1);` in every file
- All parameters and return types must have type hints
- PSR-12 formatting via Laravel Pint
- Pest syntax for tests (not PHPUnit)

### Go

- Standard `gofmt` formatting
- Errors via `core.E("scope.Method", "what failed", err)` pattern where the core framework is used
- Exported types get doc comments
- Test files co-locate with their source files

### Shell Scripts

- Shebang: `#!/bin/bash`
- Read JSON input with `jq`
- Hook output: JSON with `decision` and optional `message` fields

### Commits

Use conventional commits: `type(scope): description`

```
feat(lifecycle): add exponential backoff to dispatcher
fix(brain): handle empty embedding vectors
docs(architecture): update data flow diagram
test(registry): add concurrent access tests
```


## Project Configuration

### Go Client Config (`~/.core/agentic.yaml`)

```yaml
base_url: https://api.lthn.sh
token: your-api-token
default_project: my-project
agent_id: cladius
```

Environment variables `AGENTIC_BASE_URL`, `AGENTIC_TOKEN`, `AGENTIC_PROJECT`, and `AGENTIC_AGENT_ID` override the YAML values.

### PHP Config

The service provider merges two config files on boot:

- `src/php/config.php` into the `mcp` config key (brain database, Ollama URL, Qdrant URL)
- `src/php/agentic.php` into the `agentic` config key (Forgejo URL, token, general settings)

Environment variables:

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Claude API key |
| `GOOGLE_AI_API_KEY` | Gemini API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `BRAIN_DB_HOST` | Dedicated brain database host |
| `BRAIN_DB_DATABASE` | Dedicated brain database name |

### Workspace Config (`.core/workspace.yaml`)

Controls `core` CLI behaviour when running from the repository root:

```yaml
version: 1
active: core-php
packages_dir: ./packages
settings:
  suggest_core_commands: true
  show_active_in_prompt: true
```


## Licence

EUPL-1.2
