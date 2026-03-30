# core/php/agent — Admin UI (Livewire Components)

| Component | Class | Route | Purpose |
|-----------|-------|-------|---------|
| Dashboard | `Dashboard` | `/admin/agentic` | Agent overview (active sessions, plan stats, brain count) |
| Plans | `Plans` | `/admin/agentic/plans` | Plan listing with filters |
| Plan Detail | `PlanDetail` | `/admin/agentic/plans/{id}` | Single plan with phases, tasks, timeline |
| Sessions | `Sessions` | `/admin/agentic/sessions` | Session listing |
| Session Detail | `SessionDetail` | `/admin/agentic/sessions/{id}` | Session work log, artifacts, handoff |
| API Keys | `ApiKeys` | `/admin/agentic/api-keys` | Key listing |
| API Key Manager | `ApiKeyManager` | — | Key CRUD modal (create, revoke, permissions) |
| Templates | `Templates` | `/admin/agentic/templates` | Plan template management |
| Tool Analytics | `ToolAnalytics` | `/admin/agentic/tools` | MCP tool usage stats |
| Tool Calls | `ToolCalls` | `/admin/agentic/tool-calls` | Tool call log (debug) |
| Playground | `Playground` | `/admin/agentic/playground` | AI prompt playground |
| Request Log | `RequestLog` | `/admin/agentic/requests` | API request log |
