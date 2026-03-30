# core/php/agent — Console Commands

| Command | Artisan | Schedule | Purpose |
|---------|---------|----------|---------|
| `TaskCommand` | `agentic:task` | — | Manage tasks (create, update, toggle) |
| `PlanCommand` | `agentic:plan` | — | Manage plans (create from template, status) |
| `GenerateCommand` | `agentic:generate` | — | AI content generation |
| `PlanRetentionCommand` | `agentic:plan-cleanup` | Daily | Archive old completed plans |
| `BrainSeedMemoryCommand` | `brain:seed-memory` | — | Seed brain from file/directory |
| `BrainIngestCommand` | `brain:ingest` | — | Bulk ingest memories |
| `ScanCommand` | `agentic:scan` | Every 5 min | Scan Forge for actionable issues |
| `DispatchCommand` | `agentic:dispatch` | Every 2 min | Dispatch queued agents |
| `PrManageCommand` | `agentic:pr-manage` | Every 5 min | Manage open PRs (merge/close/review) |
| `PrepWorkspaceCommand` | `agentic:prep-workspace` | — | Prepare sandboxed workspace for agent |
