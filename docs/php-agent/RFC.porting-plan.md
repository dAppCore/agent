# Agentic Task System - Porting Plan

MCP-powered workspace for persistent work plans that survive context limits and enable multi-agent collaboration.

## Why this exists

- **Context persistence** - Work plans persist across Claude sessions, surviving context window limits
- **Multi-agent collaboration** - Handoff support between different agents (Opus, Sonnet, Haiku)
- **Checkpoint verification** - Phase gates ensure work is complete before progressing
- **Workspace state** - Shared key-value storage for agents to communicate findings

## Source Location

```
/Users/snider/Code/lab/upstream/
├── app/Models/
│   ├── AgentPlan.php         (6.1KB, ~200 lines)
│   ├── AgentPhase.php        (7.9KB, ~260 lines)
│   ├── AgentSession.php      (7.5KB, ~250 lines)
│   └── WorkspaceState.php    (2.1KB, ~70 lines)
├── app/Console/Commands/
│   ├── McpAgentServerCommand.php  (42KB, ~1200 lines)
│   ├── PlanCreateCommand.php      (8.5KB)
│   ├── PlanListCommand.php        (1.8KB)
│   ├── PlanShowCommand.php        (4.0KB)
│   ├── PlanStatusCommand.php      (3.7KB)
│   ├── PlanCheckCommand.php       (5.7KB)
│   └── PlanPhaseCommand.php       (5.8KB)
└── database/migrations/
    └── 2025_12_31_000001_create_agent_tables.php
```

## Target Location

```
/Users/snider/Code/lab/dappco.re/
├── app/Models/Agent/              # New subdirectory
│   ├── AgentPlan.php
│   ├── AgentPhase.php
│   ├── AgentSession.php
│   └── WorkspaceState.php
├── app/Console/Commands/Agent/    # New subdirectory
│   ├── McpAgentServerCommand.php
│   ├── PlanCreateCommand.php
│   ├── PlanListCommand.php
│   ├── PlanShowCommand.php
│   ├── PlanStatusCommand.php
│   ├── PlanCheckCommand.php
│   └── PlanPhaseCommand.php
├── database/migrations/
│   └── 2025_12_31_100000_create_agent_tables.php
└── tests/Feature/Agent/           # New subdirectory
    ├── AgentPlanTest.php
    ├── AgentPhaseTest.php
    └── PlanCommandsTest.php
```

---

## Phase 1: Database Migration

Create the migration file with all four tables.

### Tasks

- [ ] Create migration `2025_12_31_100000_create_agent_tables.php`
- [ ] Tables: `agent_plans`, `agent_phases`, `agent_sessions`, `workspace_states`
- [ ] Run migration and verify schema

### Source File

Copy from: `upstream/database/migrations/2025_12_31_000001_create_agent_tables.php`

### Schema Summary

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `agent_plans` | Work plans with phases | slug, title, status, current_phase |
| `agent_phases` | Individual phases | order, name, tasks (JSON), status, dependencies |
| `agent_sessions` | Agent work sessions | session_id, agent_type, work_log, handoff_notes |
| `workspace_states` | Shared key-value state | key, value (JSON), type |

---

## Phase 2: Eloquent Models

Port all four models with namespace adjustment.

### Tasks

- [ ] Create `app/Models/Agent/` directory
- [ ] Port `AgentPlan.php` - update namespace to `App\Models\Agent`
- [ ] Port `AgentPhase.php` - update namespace and relationships
- [ ] Port `AgentSession.php` - update namespace
- [ ] Port `WorkspaceState.php` - update namespace

### Namespace Changes

```php
// From (upstream)
namespace App\Models;

// To (dappco.re)
namespace App\Models\Agent;
```

### Relationship Updates

Update all `use` statements:

```php
use Mod\Agentic\Models\AgentPlan;
use Mod\Agentic\Models\AgentPhase;
use Mod\Agentic\Models\AgentSession;
use Mod\Agentic\Models\WorkspaceState;
```

### Key Methods to Verify

**AgentPlan:**
- `getCurrentPhase()` - proper orWhere scoping with closure
- `generateSlug()` - race-condition safe unique slug generation
- `checkAllPhasesComplete()` - completion verification

**AgentPhase:**
- `complete()` - wrapped in DB::transaction
- `canStart()` - dependency checking
- `isPending()`, `isCompleted()`, `isBlocked()`

---

## Phase 3: CLI Commands

Port all plan management commands.

### Tasks

- [ ] Create `app/Console/Commands/Agent/` directory
- [ ] Port `PlanCreateCommand.php` - markdown import support
- [ ] Port `PlanListCommand.php` - list all plans with stats
- [ ] Port `PlanShowCommand.php` - detailed plan view
- [ ] Port `PlanStatusCommand.php` - status management
- [ ] Port `PlanCheckCommand.php` - checkpoint verification
- [ ] Port `PlanPhaseCommand.php` - phase management

### Namespace Changes

```php
// From
namespace App\Console\Commands;

// To
namespace App\Console\Commands\Agent;
```

### Command Signatures

| Command | Signature | Purpose |
|---------|-----------|---------|
| `plan:create` | `plan:create {slug} {--title=} {--import=} {--activate}` | Create new plan |
| `plan:list` | `plan:list {--status=}` | List all plans |
| `plan:show` | `plan:show {slug} {--markdown}` | Show plan details |
| `plan:status` | `plan:status {slug} {--set=}` | Get/set plan status |
| `plan:check` | `plan:check {slug} {phase?}` | Verify phase completion |
| `plan:phase` | `plan:phase {slug} {phase} {--status=} {--add-task=} {--complete-task=}` | Manage phases |

---

## Phase 4: MCP Agent Server

Port the MCP server command with all tools and resources.

### Tasks

- [ ] Port `McpAgentServerCommand.php` (~1200 lines)
- [ ] Update all model imports to use `Mod\Agentic\Models\*`
- [ ] Register command in `Kernel.php` or auto-discovery
- [ ] Test JSON-RPC protocol over stdio

### MCP Tools (18 total)

| Tool | Purpose |
|------|---------|
| `plan_create` | Create new plan with phases |
| `plan_get` | Get plan by slug with all phases |
| `plan_list` | List plans (optionally filtered) |
| `plan_update` | Update plan status/metadata |
| `phase_update` | Update phase status |
| `phase_check` | **Checkpoint** - verify phase completion |
| `task_add` | Add task to a phase |
| `task_complete` | Mark task done |
| `session_start` | Begin agent session |
| `session_log` | Log action to session |
| `session_artifact` | Log file artifact |
| `session_handoff` | Prepare for agent handoff |
| `session_resume` | Resume from previous session |
| `session_complete` | Mark session completed |
| `state_set` | Store workspace state |
| `state_get` | Retrieve workspace state |
| `state_list` | List all state keys |
| `state_delete` | Delete state key |

### MCP Resources (5 total)

| Resource URI | Purpose |
|--------------|---------|
| `core://plans` | List of all work plans |
| `core://plans/{slug}` | Full plan as markdown |
| `core://plans/{slug}/phase/{n}` | Phase tasks as checklist |
| `core://state/{plan}/{key}` | Specific state value |
| `core://sessions/{id}` | Session handoff context |

---

## Phase 5: Tests ✅

Port and adapt tests for dappco.re conventions.

### Tasks

- [x] Create `app/Mod/Agentic/Tests/Feature/` directory
- [x] Create `AgentPlanTest.php` with factory support
- [x] Create `AgentPhaseTest.php` with factory support
- [x] Create `AgentSessionTest.php` with factory support
- [x] Create model factories (`AgentPlanFactory`, `AgentPhaseFactory`, `AgentSessionFactory`)
- [x] Run full test suite - 67 tests passing

### Test Coverage

- Model CRUD operations
- Relationship integrity
- Status transitions
- Phase dependency checking
- Command input/output
- MCP protocol compliance (optional E2E)

---

## Phase 6: Documentation and Integration

Finalise integration with dappco.re.

### Tasks

- [ ] Add MCP server config to `mcp.json` example
- [ ] Update `CLAUDE.md` with agentic task commands
- [ ] Create feature documentation following `_TEMPLATE.md`
- [ ] Add to route/command discovery if needed

### MCP Configuration

```json
{
  "mcpServers": {
    "core-agent": {
      "command": "php",
      "args": ["artisan", "mcp:agent-server"],
      "cwd": "/Users/snider/Code/lab/dappco.re"
    }
  }
}
```

---

## Verification Checklist

After each phase, verify:

- [ ] No syntax errors (`php artisan list` works)
- [ ] Migrations run cleanly
- [ ] Models can be instantiated
- [ ] Commands appear in `php artisan list`
- [ ] Tests pass (`php artisan test --filter=Agent`)

---

## Files to Copy (Summary)

| Source | Target | Changes Required |
|--------|--------|------------------|
| `upstream/database/migrations/2025_12_31_000001_create_agent_tables.php` | `dappco.re/database/migrations/2025_12_31_100000_create_agent_tables.php` | Rename only |
| `upstream/app/Models/AgentPlan.php` | `dappco.re/app/Models/Agent/AgentPlan.php` | Namespace |
| `upstream/app/Models/AgentPhase.php` | `dappco.re/app/Models/Agent/AgentPhase.php` | Namespace |
| `upstream/app/Models/AgentSession.php` | `dappco.re/app/Models/Agent/AgentSession.php` | Namespace |
| `upstream/app/Models/WorkspaceState.php` | `dappco.re/app/Models/Agent/WorkspaceState.php` | Namespace |
| `upstream/app/Console/Commands/McpAgentServerCommand.php` | `dappco.re/app/Console/Commands/Agent/McpAgentServerCommand.php` | Namespace + imports |
| `upstream/app/Console/Commands/Plan*.php` (6 files) | `dappco.re/app/Console/Commands/Agent/Plan*.php` | Namespace + imports |
| `upstream/tests/Feature/Agent*.php` | `dappco.re/tests/Feature/Agent/*.php` | Namespace |
| `upstream/tests/Feature/PlanCommandsTest.php` | `dappco.re/tests/Feature/Agent/PlanCommandsTest.php` | Namespace |

---

## Estimated Effort

| Phase | Complexity | Notes |
|-------|------------|-------|
| 1. Migration | Low | Direct copy |
| 2. Models | Low | Namespace changes only |
| 3. CLI Commands | Medium | 7 files, namespace + import updates |
| 4. MCP Server | Medium | Large file, many import updates |
| 5. Tests | Low | Namespace changes |
| 6. Documentation | Low | Config and docs |

---

## Related Services

- `ContentProcessingService` - May benefit from agent tracking
- `EntitlementService` - No direct relation
- Existing `Task` model - Different purpose (simple tasks vs agent plans)

See also: `/Users/snider/Code/lab/upstream/CLAUDE.md` for original implementation details.
