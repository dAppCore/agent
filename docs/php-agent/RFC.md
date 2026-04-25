# core/php/agent RFC — Agentic Module (PHP Implementation)

> The PHP implementation of the agent system, specced from existing code.
> Implements `code/core/agent/RFC.md` contract in PHP.
> An agent should be able to build agent features from this document alone.

**Module:** `dappco.re/php/agent`
**Namespace:** `Core\Mod\Agentic\*`
**Sub-specs:** [Actions](RFC.actions.md) | [Architecture](RFC.architecture.md) | [Commands](RFC.commands.md) | [Endpoints](RFC.endpoints.md) | [MCP Tools](RFC.mcp-tools.md) | [Models](RFC.models.md) | [OpenBrain Design](RFC.openbrain-design.md) | [OpenBrain Impl](RFC.openbrain-impl.md) | [Porting Plan](RFC.porting-plan.md) | [Security](RFC.security.md) | [UI](RFC.ui.md)

---

## 1. Domain Model

| Model | Table | Purpose |
|-------|-------|---------|
| `AgentPlan` | `agent_plans` | Structured work plans with phases, soft-deleted, activity-logged |
| `AgentPhase` | `agent_phases` | Individual phase within a plan (tasks, deps, status) |
| `AgentSession` | `agent_sessions` | Agent work sessions (context, work_log, artefacts, handoff) |
| `AgentMessage` | `agent_messages` | Direct agent-to-agent messaging (chronological, not semantic) |
| `AgentApiKey` | `agent_api_keys` | External agent access keys (hashed, scoped, rate-limited) |
| `BrainMemory` | `brain_memories` | Semantic knowledge store (tags, confidence, vector-indexed) |
| `Issue` | `issues` | Bug/feature/task tracking (labels, priority, sprint) |
| `IssueComment` | `issue_comments` | Comments on issues |
| `Sprint` | `sprints` | Time-boxed iterations grouping issues |
| `Task` | `tasks` | Simple tasks (title, status, file/line ref) |
| `Prompt` | `prompts` | Reusable AI prompt templates (system + user template) |
| `PromptVersion` | `prompt_versions` | Immutable prompt snapshots |
| `PlanTemplateVersion` | `plan_template_versions` | Immutable YAML template snapshots |
| `WorkspaceState` | `workspace_states` | Key-value state per plan (typed, shared across sessions) |

---

## 2. Actions

Single-responsibility action classes in `Actions/`:

### Brain
| Action | Method | Purpose |
|--------|--------|---------|
| `ForgetKnowledge` | `execute(id)` | Delete a memory |
| `ListKnowledge` | `execute(filters)` | List memories with filtering |
| `RecallKnowledge` | `execute(query)` | Semantic search via Qdrant |
| `RememberKnowledge` | `execute(content, tags)` | Store + embed memory |

### Forge
| Action | Method | Purpose |
|--------|--------|---------|
| `AssignAgent` | `execute(issue, agent)` | Assign agent to Forge issue |
| `CreatePlanFromIssue` | `execute(issue)` | Generate plan from issue description |
| `ManagePullRequest` | `execute(pr)` | Review/merge/close PRs |
| `ReportToIssue` | `execute(issue, report)` | Post agent findings to issue |
| `ScanForWork` | `execute()` | Scan Forge repos for actionable issues |

### Issue
| Action | Method | Purpose |
|--------|--------|---------|
| `CreateIssue` | `execute(data)` | Create issue |
| `GetIssue` | `execute(id)` | Get issue by ID |
| `ListIssues` | `execute(filters)` | List with filtering |
| `UpdateIssue` | `execute(id, data)` | Update fields |
| `AddIssueComment` | `execute(id, body)` | Add comment |
| `ArchiveIssue` | `execute(id)` | Soft delete |

### Plan
| Action | Method | Purpose |
|--------|--------|---------|
| `CreatePlan` | `execute(data)` | Create plan with phases |
| `GetPlan` | `execute(id)` | Get plan by ID/slug |
| `ListPlans` | `execute(filters)` | List plans |
| `UpdatePlanStatus` | `execute(id, status)` | Update plan status |
| `ArchivePlan` | `execute(id)` | Soft delete plan |

### Phase
| Action | Method | Purpose |
|--------|--------|---------|
| `GetPhase` | `execute(id)` | Get phase details |
| `UpdatePhaseStatus` | `execute(id, status)` | Update phase status |
| `AddCheckpoint` | `execute(id, checkpoint)` | Record checkpoint |

### Session
| Action | Method | Purpose |
|--------|--------|---------|
| `StartSession` | `execute(data)` | Start agent session |
| `ContinueSession` | `execute(id, data)` | Resume session |
| `EndSession` | `execute(id, summary)` | End session with summary |
| `GetSession` | `execute(id)` | Get session details |
| `ListSessions` | `execute(filters)` | List sessions |

### Sprint
| Action | Method | Purpose |
|--------|--------|---------|
| `CreateSprint` | `execute(data)` | Create sprint |
| `GetSprint` | `execute(id)` | Get sprint |
| `ListSprints` | `execute(filters)` | List sprints |
| `UpdateSprint` | `execute(id, data)` | Update sprint |
| `ArchiveSprint` | `execute(id)` | Soft delete |

### Task
| Action | Method | Purpose |
|--------|--------|---------|
| `ToggleTask` | `execute(id)` | Toggle task completion |
| `UpdateTask` | `execute(id, data)` | Update task fields |

---

## 3. API Endpoints

Routes in `Routes/api.php`, auth via `AgentApiAuth` middleware:

### Brain (`/v1/brain/*`)
| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/brain/remember` | RememberKnowledge |
| POST | `/v1/brain/recall` | RecallKnowledge |
| DELETE | `/v1/brain/forget/{id}` | ForgetKnowledge |
| GET | `/v1/brain/list` | ListKnowledge |

### Plans (`/v1/plans/*`)
| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/plans` | CreatePlan |
| GET | `/v1/plans` | ListPlans |
| GET | `/v1/plans/{id}` | GetPlan |
| PATCH | `/v1/plans/{id}/status` | UpdatePlanStatus |
| DELETE | `/v1/plans/{id}` | ArchivePlan |

### Sessions (`/v1/sessions/*`)
| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/sessions` | StartSession |
| GET | `/v1/sessions` | ListSessions |
| GET | `/v1/sessions/{id}` | GetSession |
| POST | `/v1/sessions/{id}/continue` | ContinueSession |
| POST | `/v1/sessions/{id}/end` | EndSession |

### Messages (`/v1/messages/*`)
| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/messages/send` | AgentSend |
| GET | `/v1/messages/inbox` | AgentInbox |
| GET | `/v1/messages/conversation/{agent}` | AgentConversation |

### Issues, Sprints, Tasks, Phases — similar CRUD patterns.

### Auth (`/v1/agent/auth/*`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| POST | `/v1/agent/auth/provision` | ProvisionAgentKey | OAuth (Authentik) |
| DELETE | `/v1/agent/auth/revoke/{key_id}` | RevokeAgentKey | AgentApiKey |

### Fleet (`/v1/fleet/*`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| POST | `/v1/fleet/register` | RegisterNode | AgentApiKey |
| POST | `/v1/fleet/heartbeat` | NodeHeartbeat | AgentApiKey |
| POST | `/v1/fleet/deregister` | DeregisterNode | AgentApiKey |
| GET | `/v1/fleet/nodes` | ListNodes | AgentApiKey |
| POST | `/v1/fleet/task/assign` | AssignTask | AgentApiKey |
| POST | `/v1/fleet/task/complete` | CompleteTask | AgentApiKey |
| GET | `/v1/fleet/task/next` | GetNextTask | AgentApiKey |

### Fleet Events (SSE)

| Method | Path | Purpose | Auth |
|--------|------|---------|------|
| GET | `/v1/fleet/events` | SSE stream — pushes task assignments to connected nodes | AgentApiKey |

The SSE connection stays open. When the scheduler assigns a task to a node, it pushes a `task.assigned` event. Nodes that can't hold SSE connections fall back to polling `GET /v1/fleet/task/next`.

### Fleet Stats (`/v1/fleet/stats`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| GET | `/v1/fleet/stats` | GetFleetStats | AgentApiKey |

Returns: nodes_online, tasks_today, tasks_week, repos_touched, findings_total, compute_hours.

### Sync (`/v1/agent/sync/*`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| POST | `/v1/agent/sync` | PushDispatchHistory | AgentApiKey |
| GET | `/v1/agent/context` | PullFleetContext | AgentApiKey |
| GET | `/v1/agent/status` | GetAgentSyncStatus | AgentApiKey |

### Credits (`/v1/credits/*`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| POST | `/v1/credits/award` | AwardCredits | Internal |
| GET | `/v1/credits/balance/{agent_id}` | GetBalance | AgentApiKey |
| GET | `/v1/credits/history/{agent_id}` | GetCreditHistory | AgentApiKey |

### Subscription (`/v1/subscription/*`)

| Method | Path | Action | Auth |
|--------|------|--------|------|
| POST | `/v1/subscription/detect` | DetectCapabilities | AgentApiKey |
| GET | `/v1/subscription/budget/{agent_id}` | GetNodeBudget | AgentApiKey |
| PUT | `/v1/subscription/budget/{agent_id}` | UpdateBudget | AgentApiKey |

---

## 4. MCP Tools

Registered via `AgentToolRegistry` in `onMcpTools()`:

### Brain Tools
| Tool | MCP Name | Maps To |
|------|----------|---------|
| `BrainRemember` | `brain_remember` | RememberKnowledge action |
| `BrainRecall` | `brain_recall` | RecallKnowledge action |
| `BrainForget` | `brain_forget` | ForgetKnowledge action |
| `BrainList` | `brain_list` | ListKnowledge action |

### Messaging Tools
| Tool | MCP Name | Maps To |
|------|----------|---------|
| `AgentSend` | `agent_send` | POST /v1/messages/send |
| `AgentInbox` | `agent_inbox` | GET /v1/messages/inbox |
| `AgentConversation` | `agent_conversation` | GET /v1/messages/conversation |

### Plan/Session/Phase/Task/Template tools — same pattern.

---

## 5. OpenBrain

OpenBrain architecture (storage layers, schema, flow, lifecycle) is defined in `code/core/agent/RFC.md` section "OpenBrain Architecture". PHP provides the MariaDB persistence layer, Qdrant integration, and Ollama embedding via `BrainService`.

---

## 6. Provider Abstraction

```php
interface AgenticProviderInterface
{
    public function generate(string $prompt, array $options = []): string;
    public function stream(string $prompt, array $options = [], callable $onToken): void;
    public function name(): string;
    public function defaultModel(): string;
    public function isAvailable(): bool;
}
```

`AgenticManager` registers providers (Claude, Gemini, OpenAI) with retry + exponential backoff.

---

## 7. Session Lifecycle

```
StartSession(plan_id, agent) -> active session with context
  -> Agent works, appends to work_log
  -> ContinueSession(id, work) -> resume from last state
  -> EndSession(id, summary, handoff_notes) -> closed
  -> session_handoff tool: {summary, next_steps, blockers, context_for_next}
  -> session_replay tool: recover context from completed session
```

### Workspace State

Key-value store shared between sessions within a plan:

```php
// Agent A discovers something
WorkspaceState::set($planId, 'discovered_pattern', 'observer');

// Agent B reads it later
$pattern = WorkspaceState::get($planId, 'discovered_pattern');
```

### 7.x Fleet tasks vs sessions

Fleet tasks (AssignTask / CompleteTask) are deliberately out-of-session. AgentSession's work_log, artefacts, handoff, and replay semantics are designed for interactive / MCP-driven flows, not for the atomic assign→complete shape of fleet distribution. If a fleet task's handler needs session-style replay, that handler should start its own AgentSession via AgentSessionService when it begins the work.

---

## 8. API Key Security

- **Hashing**: Argon2id (upgraded from SHA-256 Jan 2026)
- **Scoping**: Permission strings (`plans:read`, `plans:write`, `sessions:write`, `brain:recall`)
- **IP restriction**: IPv4/IPv6/CIDR whitelist via `IpRestrictionService`
- **Rate limiting**: Per-key configurable limits
- **Display**: Key shown once on creation, stored hashed, prefix `ak_` for identification

---

## 9. Services

| Service | Purpose |
|---------|---------|
| `AgenticManager` | Provider registry (claude, gemini, openai) |
| `AgentSessionService` | Session lifecycle management |
| `AgentApiKeyService` | API key CRUD + hashing |
| `AgentToolRegistry` | MCP tool registration |
| `BrainService` | Qdrant + Ollama integration (embed, search, store) |
| `ClaudeService` | Anthropic API client |
| `GeminiService` | Google Gemini API client |
| `OpenAIService` | OpenAI API client |
| `ForgejoService` | Forgejo API client (issues, PRs, repos) |
| `ContentService` | AI content generation pipeline |
| `PlanTemplateService` | YAML template loading + versioning |
| `IpRestrictionService` | IP whitelist enforcement |
| `AgentDetection` | Detect agent type from request headers |

---

## 10. Console Commands

| Command | Artisan | Purpose |
|---------|---------|---------|
| `TaskCommand` | `agentic:task` | Manage tasks |
| `PlanCommand` | `agentic:plan` | Manage plans |
| `GenerateCommand` | `agentic:generate` | AI content generation |
| `PlanRetentionCommand` | `agentic:plan-cleanup` | Clean old plans (scheduled daily) |
| `BrainSeedMemoryCommand` | `brain:seed-memory` | Seed brain from files |
| `BrainIngestCommand` | `brain:ingest` | Bulk ingest into brain |
| `ScanCommand` | `agentic:scan` | Scan Forge for work (every 5 min) |
| `DispatchCommand` | `agentic:dispatch` | Dispatch agents (every 2 min) |
| `PrManageCommand` | `agentic:pr-manage` | Manage PRs (every 5 min) |
| `PrepWorkspaceCommand` | `agentic:prep-workspace` | Prepare agent workspace |

---

## 11. Admin UI (Livewire)

| Component | Route | Purpose |
|-----------|-------|---------|
| `Dashboard` | `/admin/agentic` | Overview stats |
| `Plans` | `/admin/agentic/plans` | Plan listing |
| `PlanDetail` | `/admin/agentic/plans/{id}` | Single plan view |
| `Sessions` | `/admin/agentic/sessions` | Session listing |
| `SessionDetail` | `/admin/agentic/sessions/{id}` | Single session view |
| `ApiKeys` | `/admin/agentic/api-keys` | Key management |
| `ApiKeyManager` | — | Key CRUD modal |
| `Templates` | `/admin/agentic/templates` | Template management |
| `ToolAnalytics` | `/admin/agentic/tools` | Tool usage stats |
| `ToolCalls` | `/admin/agentic/tool-calls` | Tool call log |
| `Playground` | `/admin/agentic/playground` | AI playground |
| `RequestLog` | `/admin/agentic/requests` | API request log |

---

## 12. Content Generation Pipeline

The agentic module was originally built for AI-driven content generation. This is the PHP side's primary product — the Go agent inherited dispatch/workspace/brain but content generation stays PHP.

### Pipeline

```
Product Briefs (per service)
  -> Prompt Templates (system + user, versioned)
    -> AI Generation (Claude/Gemini via AgenticManager)
      -> Drafts (blog posts, help articles, social media)
        -> Quality Refinement (scoring, rewriting)
          -> Publication (CMS, social scheduler, help desk)
```

### Product Briefs

Each service has a brief (`Resources/briefs/`) that gives AI the product context.

| Brief | Product |
|-------|---------|
| `host-link.md` | LinkHost |
| `host-social.md` | SocialHost |
| `host-analytics.md` | AnalyticsHost |
| `host-trust.md` | TrustHost |
| `host-notify.md` | NotifyHost |

### Prompt Templates

Versioned prompt templates in `Resources/prompts/`:

| Category | Templates |
|----------|-----------|
| **Content** | blog-post, help-article, landing-page, social-media, quality-refinement |
| **Development** | architecture-review, code-review, debug-session, test-generation |
| **Visual** | infographic, logo-generation, social-graphics |
| **System** | dappcore-writer (brand voice) |

### Natural Progression SEO

Content changes create **future revisions** (scheduled posts with no date). When Googlebot visits a page with pending revisions, the system schedules publication 8-62 minutes later — making updates appear as natural content evolution rather than bulk changes.

### MCP Content Tools

```
content_generate     — Generate content from brief + prompt template
content_batch        — Batch generation across services
content_brief_create — Create new product brief
```

### SEO Schema Generation

Structured data templates for generated content:
- Article (BlogPosting, TechArticle)
- FAQ (FAQPage)
- HowTo (step-by-step guides)

---

## 13. Reference Material

| Resource | Location |
|----------|----------|
| Agent contract (cross-cutting) | `code/core/agent/RFC.md` |
| Go implementation | `code/core/go/agent/RFC.md` |
| lthn.sh platform | `project/lthn/ai/RFC.md` |

---

## Changelog

- 2026-03-29: Restructured as PHP implementation spec. OpenBrain architecture and polyglot mapping moved to `code/core/agent/RFC.md`. Added contract reference. Kept all PHP-specific detail (Eloquent, Livewire, actions, services, commands, admin UI, content pipeline).
- 2026-03-27: Initial RFC specced from existing PHP codebase. 14 models, 30+ actions, 20+ API endpoints, 12 MCP tools, 10 console commands, 12 admin UI components.
