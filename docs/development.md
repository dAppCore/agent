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

The module is `dappco.re/go/agent`, rooted at the `go/` subdirectory of this repository. It participates in a Go workspace (`go.work`) that resolves all `dappco.re/go/*` dependencies locally via the submodules under `external/`. Run Go tooling from `go/`:

- Development / default: `cd go && go build ./...`, `cd go && go test ./...`
- CI / reproducibility: add `GOWORK=off` (and optionally `GOFLAGS=-mod=mod`) when running `go test`, `go vet`, and `go mod tidy` from `go/`.

### PHP Dependencies

```bash
composer install
```

The Composer package is `lthn/agent`. It depends on `lthn/php` (the foundation framework) at runtime, and on `orchestra/testbench`, `pestphp/pest`, and `livewire/livewire` for development.


## Building

### The Binary

This module produces a single binary from `go/cmd/core-agent`:

```bash
cd go
go build ./cmd/core-agent/        # build core-agent
go install ./cmd/core-agent/      # install to $GOPATH/bin
go build ./...                    # build all packages
```

The same source ships under two names — `core-agent` and `lthn-agent`. Build the family-consistent name by setting the output:

```bash
go build -o lthn-agent ./cmd/core-agent/
```

The binary detects its invocation name from `argv[0]`, so either name behaves identically.

### MCP + serve modes

The binary is itself the MCP server. The `mcp` (stdio) and `serve` (HTTP) commands are registered by the shared `dappco.re/go/mcp` service the binary mounts:

```bash
core-agent mcp        # MCP server over stdio — what an IDE connects to
core-agent serve      # HTTP MCP daemon — cross-agent communication
```

The tool surface (dispatch, plans, brain, messaging, `lemma_send`, …) is registered by the `agentic`, `brain`, and `lemma` subsystems into that one service. There are no separate per-server binaries.


## Testing

### Go Tests

```bash
cd go

# Run all Go tests
go test ./... -count=1

# Run a single test by name
go test ./pkg/agentic/ -run TestDispatch_Good

# Vet
go vet ./...

# Reproducible run (CI parity)
GOWORK=off go test ./... -count=1
```

Tests use `testify/assert` and `testify/require`, with one test file per source file. The naming convention is `TestFilename_FunctionName_<Category>`:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy-path tests — prove the contract works |
| `_Bad` | Expected error conditions — prove error handling |
| `_Ugly` | Panics and edge cases |

The test suite is substantial — hundreds of tests across the Go packages, covering `agentic` (dispatch, prep, verify, scan, plans, phases, sessions, fleet, platform, mirror), `brain` (direct, provider, messaging, tools), `lemma` (sessions, admin), `monitor` (harvest, sync), `runner` (queue, paths), and `setup` (detect, config, scaffold). Each `*_example_test.go` doubles as an executable usage example.

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
cd go

# Format all Go files
gofmt -w .

# Run the linter
golangci-lint run --timeout=5m --tests=false ./...

# Run go vet
go vet ./...
```

### PHP

```bash
# Fix code style (Laravel Pint, PSR-12)
composer lint

# Format only changed files
./vendor/bin/pint --dirty
```

### Automatic Formatting

The `core` plugin includes PostToolUse hooks (under `provider/claude/core/scripts/`) that auto-format files after every edit:

- **Go files**: `go-format.sh` runs `gofmt` on any edited `.go` file
- **PHP files**: `php-format.sh` runs `pint` on any edited `.php` file
- **Debug check**: `check-debug.sh` warns about `dd()`, `dump()`, `fmt.Println()`, and similar statements left in code


## Provider Integrations

Per-provider integration trees live under `provider/`:

- `provider/claude/` — Claude Code plugin sources (`core`, `core-go`, `core-php`, `devops`, `infra`, `research`, plus the `camofox_mcp` and `hermes_runner_mcp` MCP plugins).
- `provider/codex/` — OpenAI Codex plugin sources (`core`, `code`, `ci`, `qa`, `review`, `verify`, plus `ethics`, `guardrails`, `perf`, `issue`, `coolify`, `awareness`, `api`, `collect`).
- `provider/google/` — Gemini CLI integration.
- `provider/hermes/` — Hermes plugins + skills (including the OpenBrain memory/context Python plugins).

### Claude Code Plugins

The marketplace registry at the repository root (`.claude-plugin/marketplace.json`) publishes the plugins. Locally-sourced plugins point at `./provider/claude/<name>`; some entries are published from URLs. Add the marketplace and install a plugin:

```bash
claude plugin marketplace add https://github.com/dappcore/agent
claude plugin install core
```

Each plugin lives in `provider/claude/<name>/` and contains:

```
provider/claude/<name>/
├── .claude-plugin/plugin.json   # metadata (name, version, description)
├── 000.mcp.json                 # MCP server registration (optional)
├── hooks.json                   # hook declarations (optional)
├── scripts/                     # supporting + hook scripts (optional)
├── commands/                    # slash command definitions (*.md)
├── agents/                      # subagent definitions (optional)
└── skills/                      # skill definitions (optional)
```

### Hook System

The `core` plugin's `hooks.json` fires scripts (from `provider/claude/core/scripts/`) across the Claude Code lifecycle — PreToolUse guards, PostToolUse auto-format + debug warnings + inbox/notify checks, and completion checks. Hook scripts read JSON on stdin and emit a JSON object with a `decision` (`approve` or `block`) and an optional `message`. Test one locally by piping a tool-input fixture into it.

### Adding a New Plugin

1. Create `provider/claude/<name>/.claude-plugin/plugin.json` with `name`, `description`, `version`, `author`, and `license` (EUPL-1.2).
2. Add command files as Markdown in `commands/` — the filename becomes the command name.
3. Register the plugin in `.claude-plugin/marketplace.json` with its `name`, `source` (`./provider/claude/<name>`), `description`, and `version`.


## Adding Go Functionality

### New Package

Create a directory under `go/pkg/`. Follow the existing convention — one test file per source file, with `*_example_test.go` doubling as runnable usage examples. Import the package as `dappco.re/go/agent/pkg/<name>`.

### New CLI Command

CLI commands register against the `core.Core` via `c.Command(name, core.Command{...})`. Binary-level commands are registered in `go/cmd/core-agent/commands.go`; subsystem commands are registered by the owning package (for example `pkg/agentic/commands_plan.go`). Actions return a `core.Result`:

```go
c.Command("my-command", core.Command{
    Description: "What it does",
    Action: func(opts core.Options) core.Result {
        // read opts.String("flag") etc.
        return core.Result{OK: true}
    },
})
```

### New MCP Tool

MCP tools are registered into the shared `dappco.re/go/mcp` service by a subsystem, via `coremcp.AddToolRecorded`:

```go
coremcp.AddToolRecorded(svc, svc.Server(), "<subsystem>", &mcp.Tool{
    Name:        "my_tool",
    Description: "What the tool does and when to use it.",
}, func(ctx context.Context, req *mcp.CallToolRequest, in MyInput) (*mcp.CallToolResult, MyOutput, error) {
    // implementation
    return nil, MyOutput{...}, nil
})
```

Wire the registration from the subsystem's `RegisterTools` (see `pkg/agentic/dispatch.go` or `cmd/core-agent/lemma_mcp.go` for working examples). The same service serves both the stdio (`mcp`) and HTTP (`serve`) transports — there is no separate per-server binary.


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
