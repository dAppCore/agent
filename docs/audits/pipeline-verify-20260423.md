# Pipeline, Plugin, and Session Lifecycle Verification - 2026-04-23

## Audit basis

- Ticket scope: audit-only verification for MetaReader pipeline, plugin restructure, and session lifecycle; this report is the only created file.
- The cross-cutting RFC links the pipeline and plugin restructure sub-specs as `RFC.pipeline.md` and `RFC.plugin-restructure.md` from `docs/RFC-AGENT.md:25`.
- In this checkout, the matching RFC bodies are present as `docs/RFC-AGENT-PIPELINE.md` and `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md`, with pipeline scope at `docs/RFC-AGENT-PIPELINE.md:1` and plugin scope at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:1`.
- The PHP RFC names `AgentSession` as work sessions with `work_log`, artefacts, and handoff at `docs/php-agent/RFC.md:19`.
- The PHP RFC names `WorkspaceState` as typed, shared state per plan at `docs/php-agent/RFC.md:30`.
- Session lifecycle is section 7 in `docs/php-agent/RFC.md:253`, while the cross-cutting RFC has session lifecycle as section 13 at `docs/RFC-AGENT.md:726`.
- Negative search basis: `rg -n "MetaReader|PRMeta|EpicMeta|ReactionMeta|GetPRMeta|GetEpicMeta|GetIssueState|GetCommentReactions" php` returned no PHP implementation hits.
- Negative search basis: `find php -maxdepth 3 -type d` returned no `php/Pipeline`, `php/Plugin`, `php/Session`, `php/Workspace`, or `php/Fleet` directories; related implementation lives under `php/Actions`, `php/Services`, `php/Mcp`, `php/Models`, and `php/Controllers`.
- Negative search basis: `find . -maxdepth 4 -name marketplace.yaml -o -name marketplace.yml` returned no YAML marketplace files.

## Verification 1 - MetaReader stage

**Verdict: MISSING**

### RFC expectation

- The pipeline RFC defines issue-to-merge flow before the MetaReader section, including issue pickup, workspace prep, agent dispatch, QA, PR, review, fix loop, merge, training data, and issue close at `docs/RFC-AGENT-PIPELINE.md:8`.
- The RFC says every pipeline decision comes through `MetaReader` at `docs/RFC-AGENT-PIPELINE.md:93`.
- The RFC says `MetaReader` must never read comment bodies, commit messages, PR descriptions, or review content at `docs/RFC-AGENT-PIPELINE.md:95`.
- The RFC interface includes `GetPRMeta`, `GetEpicMeta`, `GetIssueState`, and `GetCommentReactions` at `docs/RFC-AGENT-PIPELINE.md:97`.
- `PRMeta` is structural metadata: state, mergeability, head SHA/date, branches, checks, review thread counts, and an eyes reaction flag at `docs/RFC-AGENT-PIPELINE.md:106`.
- `EpicMeta` is structural metadata: issue state and child issue checked/open/PR linkage at `docs/RFC-AGENT-PIPELINE.md:130`.
- The RFC explicitly excludes comment bodies, commit messages, PR descriptions, and review thread content from the MetaReader surface at `docs/RFC-AGENT-PIPELINE.md:146`.
- The RFC says content stripping should happen at query level, before content enters the process, at `docs/RFC-AGENT-PIPELINE.md:154`.
- The RFC defines the three stages as audit, organise, and execute at `docs/RFC-AGENT-PIPELINE.md:156`.
- Stage 3 expects dispatch, monitor CI/reviews/conflicts/merges, intervention, phase completion, and epic merge at `docs/RFC-AGENT-PIPELINE.md:173`.

### Implementation evidence

- The PHP module schedules `agentic:scan`, `agentic:dispatch`, and `agentic:pr-manage` when a Forge token is present at `php/Boot.php:50`.
- The scheduled PHP pipeline is command-based rather than a `MetaReader` precondition surface, because the registered commands are scan, dispatch, and PR management at `php/Boot.php:52`.
- `ScanForWork` describes itself as scanning Forgejo for epic issues and unchecked children at `php/Actions/Forge/ScanForWork.php:17`.
- `ScanForWork` says it parses epic issue bodies for checklist syntax at `php/Actions/Forge/ScanForWork.php:20`.
- `ScanForWork` fetches epic issues through `listIssues()` at `php/Actions/Forge/ScanForWork.php:50`.
- `ScanForWork` fetches PRs through `listPullRequests()` at `php/Actions/Forge/ScanForWork.php:56`.
- `ScanForWork` parses the epic body directly with `$epic['body']` at `php/Actions/Forge/ScanForWork.php:62`.
- `ScanForWork` returns each child issue body as `issue_body` at `php/Actions/Forge/ScanForWork.php:84`.
- `ScanForWork` uses a regex over checklist body text in `parseChecklist()` at `php/Actions/Forge/ScanForWork.php:104`.
- `ScanForWork` extracts linked issues from PR bodies by reading `$pr['body']` at `php/Actions/Forge/ScanForWork.php:133`.
- `ScanForWork` uses a regex over PR body text to discover `#N` references at `php/Actions/Forge/ScanForWork.php:136`.
- This body parsing conflicts with the RFC exclusion for issue/comment/PR content at `docs/RFC-AGENT-PIPELINE.md:146`.
- `ManagePullRequest` directly calls `getPullRequest()` at `php/Actions/Forge/ManagePullRequest.php:38`.
- `ManagePullRequest` checks open state at `php/Actions/Forge/ManagePullRequest.php:40`.
- `ManagePullRequest` checks mergeability at `php/Actions/Forge/ManagePullRequest.php:44`.
- `ManagePullRequest` checks combined commit status at `php/Actions/Forge/ManagePullRequest.php:48`.
- `ManagePullRequest` merges the PR directly after status checks at `php/Actions/Forge/ManagePullRequest.php:55`.
- `ManagePullRequest` implements some PR structural checks, but not behind the `MetaReader` interface required by `docs/RFC-AGENT-PIPELINE.md:97`.
- `ForgejoService::listIssues()` returns raw decoded issue payloads from `/issues` at `php/Services/ForgejoService.php:34`.
- `ForgejoService::getIssue()` returns raw decoded issue payloads from `/issues/{number}` at `php/Services/ForgejoService.php:50`.
- `ForgejoService::listPullRequests()` returns raw decoded pull payloads from `/pulls` at `php/Services/ForgejoService.php:85`.
- `ForgejoService::getPullRequest()` returns raw decoded pull payloads from `/pulls/{number}` at `php/Services/ForgejoService.php:95`.
- `ForgejoService::getCombinedStatus()` returns raw combined status payloads at `php/Services/ForgejoService.php:105`.
- `ForgejoService` adds JSON accept headers and timeout at `php/Services/ForgejoService.php:147`, but it does not filter fields to structural metadata before callers receive the payloads at `php/Services/ForgejoService.php:170`.
- The only PHP `pipeline` search hits in MCP content tooling are content generation, not dispatch verification, at `php/Mcp/Tools/Agent/Content/ContentGenerate.php:13`.
- `ContentGenerate` supports Gemini draft, Claude refine, or full content modes at `php/Mcp/Tools/Agent/Content/ContentGenerate.php:15`.
- `GenerateCommand` describes a content pipeline, not the MetaReader dispatch pipeline, at `php/Console/Commands/GenerateCommand.php:28`.
- `ReportToIssue` calls itself a standalone action within the orchestration pipeline at `php/Actions/Forge/ReportToIssue.php:20`, but it only posts comments through `ForgejoService::createComment()` at `php/Actions/Forge/ReportToIssue.php:30`.

### Gap assessment

- There is no PHP `MetaReader` class, interface, or equivalent named abstraction in the audited source, based on the negative search basis above and the direct Forgejo callers at `php/Actions/Forge/ScanForWork.php:48` and `php/Actions/Forge/ManagePullRequest.php:36`.
- There is no precondition stage that strips body/description/review content before pipeline decisions, based on body parsing in `ScanForWork` at `php/Actions/Forge/ScanForWork.php:62` and `php/Actions/Forge/ScanForWork.php:133`.
- The PHP implementation has partial structural PR checks through `ManagePullRequest`, but those checks are local to that action and do not satisfy "every pipeline decision comes through this interface" at `docs/RFC-AGENT-PIPELINE.md:95`.
- The content-generation pipeline is implemented separately and should not be counted as the MetaReader pipeline because its subject is brief generation at `php/Mcp/Tools/Agent/Content/ContentGenerate.php:36`.

### Follow-up ticket scope

- Add a PHP MetaReader contract and Forgejo-backed implementation that returns only PR, epic, issue, reaction, and check metadata matching `docs/RFC-AGENT-PIPELINE.md:97`.
- Refactor `ScanForWork` and `ManagePullRequest` to depend on MetaReader outputs instead of raw Forgejo payloads; remove direct PR/issue body parsing from pipeline decisions at `php/Actions/Forge/ScanForWork.php:62` and `php/Actions/Forge/ScanForWork.php:133`.
- Add tests proving body, description, comment, commit, and review-thread content do not enter the pipeline decision layer, matching `docs/RFC-AGENT-PIPELINE.md:146`.

## Verification 2 - Plugin family restructure

**Verdict: PARTIAL**

### RFC expectation

- The plugin RFC says three skeleton plugins need building out, and names the source families as core-go, core-php, and infra at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:5`.
- Step 1 requires `dappcore-go` to be renamed to `core-go` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:7`.
- Step 1 requires adding `README.md` and `marketplace.yaml` for core-go at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:27`.
- Step 2 requires `dappcore-php` to be renamed to `core-php` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:31`.
- Step 2 requires adding `README.md` and `marketplace.yaml` for core-php at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:50`.
- Step 3 requires an infra plugin update and adding `marketplace.yaml` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:54`.
- Step 4 requires endpoint documentation for `api.lthn.sh`, `mcp.lthn.sh`, JSON Accept, JSON Content-Type, bearer auth, and `/v1/{resource}` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:75`.
- Step 4 requires `.mcp.json` in core-go and core-php to reference `core mcp serve` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:90`.
- Step 5 requires `marketplace.yaml` for all three plugins, with registry `forge.lthn.ai`, organisation `core`, repository name, auto-update, and 24h check interval at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:92`.
- The verification checklist requires root `.claude-plugin/plugin.json`, root-level commands/agents/skills, valid frontmatter, no hardcoded paths, and `core mcp serve` validation at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:104`.
- The RFC explicitly marks Codex and Gemini plugins out of scope for that RFC at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:112`.

### Implementation evidence

- The repository has a Claude marketplace JSON named `dappcore-agent`, not a YAML marketplace, at `.claude-plugin/marketplace.json:2`.
- The Claude marketplace includes a local `core` plugin at `.claude-plugin/marketplace.json:10`.
- The Claude marketplace includes a `core-php` entry sourced from `https://forge.lthn.ai/core/php.git` at `.claude-plugin/marketplace.json:22`.
- The Claude marketplace includes a `core-build` entry sourced from `https://forge.lthn.ai/core/go-build.git` at `.claude-plugin/marketplace.json:31`.
- The Claude marketplace includes a `core-devops` entry sourced from `https://forge.lthn.ai/core/go-devops.git` at `.claude-plugin/marketplace.json:40`.
- The Claude marketplace is JSON, while the RFC requires `marketplace.yaml` at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:92`.
- The root Claude package metadata is a Claude Code plugin marketplace package at `.claude-plugin/package.json:2`.
- The `claude/core` plugin manifest is named `agent`, not `core-go`, `core-php`, or `infra`, at `claude/core/.claude-plugin/plugin.json:2`.
- The `claude/core` plugin homepage remains `https://dappco.re/agent/claude` at `claude/core/.claude-plugin/plugin.json:9`.
- The `claude/core` plugin repository remains `https://github.com/dAppCore/agent.git` at `claude/core/.claude-plugin/plugin.json:10`.
- The `claude/research` plugin homepage remains `https://dappco.re/agent/claude` at `claude/research/.claude-plugin/plugin.json:9`.
- The `claude/research` plugin repository remains `https://github.com/dAppCore/agent.git` at `claude/research/.claude-plugin/plugin.json:10`.
- The `claude/devops` plugin exists as `devops` at `claude/devops/.claude-plugin/plugin.json:2`, but it is not named `infra` as described by the RFC step at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:54`.
- The root `.mcp.json` runs `core-agent mcp` at `.mcp.json:5`.
- `claude/core/.mcp.json` also runs `core-agent mcp` at `claude/core/.mcp.json:4`.
- The RFC requested `.mcp.json` to reference `core mcp serve`, not `core-agent mcp`, at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:90`.
- Claude scripts document the API endpoint default as `https://api.lthn.sh` at `claude/core/scripts/session-start.sh:8`.
- `session-start.sh` sends `Content-Type: application/json` at `claude/core/scripts/session-start.sh:29`.
- `session-start.sh` sends `Accept: application/json` at `claude/core/scripts/session-start.sh:30`.
- `session-start.sh` sends bearer auth at `claude/core/scripts/session-start.sh:31`.
- `session-save.sh` sends `Content-Type: application/json` at `claude/core/scripts/session-save.sh:59`.
- `session-save.sh` sends `Accept: application/json` at `claude/core/scripts/session-save.sh:60`.
- `session-save.sh` sends bearer auth at `claude/core/scripts/session-save.sh:61`.
- These scripts partially satisfy the endpoint convention, but the RFC asked for a shared skill or pattern file at `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:77`.
- The Codex marketplace JSON is present at `codex/.codex-plugin/marketplace.json:2`.
- The Codex marketplace lists a root Codex plugin at `codex/.codex-plugin/marketplace.json:10`.
- The Codex marketplace lists plugin families such as `api`, `ci`, `code`, `core`, `qa`, `review`, and `verify` at `codex/.codex-plugin/marketplace.json:34`.
- The Codex root plugin manifest is named `codex` at `codex/.codex-plugin/plugin.json:2`.
- The Codex code plugin manifest is named `code` at `codex/code/.codex-plugin/plugin.json:2`.
- The Codex code plugin contains a `core-go` skill frontmatter name at `codex/code/skills/go/SKILL.md:2`.
- The Codex code plugin contains a `core-php` skill frontmatter name at `codex/code/skills/php/SKILL.md:2`.
- The Codex README says the Codex plugin mirrors key behaviours from the Claude plugin suite at `codex/README.md:3`.
- The Codex README lists `.codex-plugin/marketplace.json` as the Codex marketplace registry at `codex/README.md:40`.
- The Codex AGENTS file says `claude/` contains Claude Code plugins at `codex/AGENTS.md:44`.
- The Codex AGENTS file says `google/gemini-cli/` contains the Gemini CLI extension at `codex/AGENTS.md:45`.
- The audited tree has only `scripts/gemini-batch-runner.sh` as a Gemini-named file under the max-depth plugin scan, while no `google/gemini-cli` plugin metadata appeared in the negative search basis.

### Gap assessment

- Claude and Codex plugin families exist, but the RFC's specific `core-go`, `core-php`, and infra restructure is only partially represented by marketplace entries and skills rather than first-class plugin directories with YAML marketplaces.
- Marketplace integration is partial because JSON registries exist at `.claude-plugin/marketplace.json:1` and `codex/.codex-plugin/marketplace.json:1`, but the RFC-required `marketplace.yaml` files are absent by negative search basis.
- The namespace rename is incomplete because Claude manifests still contain `dappcore-agent`, `dappco.re`, and `dAppCore` identifiers at `.claude-plugin/marketplace.json:2`, `claude/core/.claude-plugin/plugin.json:9`, and `claude/core/.claude-plugin/plugin.json:10`.
- API endpoint behaviour is partially documented in executable Claude scripts at `claude/core/scripts/session-start.sh:27`, but no shared `api-endpoints/SKILL.md` equivalent was found in the plugin families covered by the negative search basis.
- Codex has a richer plugin family than the plugin RFC expected, but that family is named by workflow (`code`, `qa`, `review`, `verify`) rather than by `core-go`, `core-php`, and `infra` at `codex/.codex-plugin/marketplace.json:46`.
- Gemini plugin integration is not implemented as a plugin family in this checkout, despite `codex/AGENTS.md:45` documenting a `google/gemini-cli` location.

### Follow-up ticket scope

- Decide whether the canonical marketplace format is YAML or JSON; if YAML remains required, add `marketplace.yaml` to core-go, core-php, and infra equivalents using the RFC template from `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:95`.
- Finish the `dappcore` to `core` rename across Claude metadata, or explicitly document why legacy `dappcore-agent` and `dAppCore` identifiers remain at `.claude-plugin/marketplace.json:2` and `claude/core/.claude-plugin/plugin.json:10`.
- Add a shared API/MCP endpoint skill or pattern file and align `.mcp.json` commands with the canonical command chosen for `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md:90`.

## Verification 3 - Session lifecycle and cross-session state

**Verdict: PARTIAL**

### RFC expectation

- The cross-cutting RFC says sessions belong to a plan and an agent, track `work_log`, and produce artefacts at `docs/RFC-AGENT.md:58`.
- The cross-cutting RFC says `WorkspaceState` is key-value state per plan, typed, and shared across sessions at `docs/RFC-AGENT.md:54`.
- The PHP RFC names `AgentSession` as work sessions with context, `work_log`, artefacts, and handoff at `docs/php-agent/RFC.md:19`.
- The PHP RFC names `WorkspaceState` as key-value state per plan, typed and shared across sessions at `docs/php-agent/RFC.md:30`.
- The PHP lifecycle flow is start session, append to `work_log`, continue from last state, end with summary and handoff notes, handoff, and replay at `docs/php-agent/RFC.md:253`.
- The PHP RFC says WorkspaceState is shared between sessions within a plan at `docs/php-agent/RFC.md:264`.
- The cross-cutting API surface says Go is local workspace state, PHP is persistent database state, and sync connects local dispatch history/findings to fleet context at `docs/RFC-AGENT.md:198`.
- The remote state sync RFC says dispatch history should create BrainMemory records, update WorkspaceState workflow progress, and notify subscribers at `docs/RFC-AGENT.md:981`.
- The PHP sync endpoint table says `/v1/agent/sync` should receive dispatch history/findings and write to BrainMemory plus WorkspaceState at `docs/RFC-AGENT.md:1127`.

### Implementation evidence

- `AgentSession` declares context, `work_log`, artefacts, handoff notes, final summary, and lifecycle timestamps in properties at `php/Models/AgentSession.php:28`.
- `AgentSession` marks those columns fillable at `php/Models/AgentSession.php:51`.
- `AgentSession` casts `context_summary`, `work_log`, `artifacts`, and `handoff_notes` as arrays at `php/Models/AgentSession.php:68`.
- The session table migration stores `context_summary`, `work_log`, `artifacts`, `handoff_notes`, and final summary at `php/Migrations/0001_01_01_000001_create_agentic_tables.php:48`.
- `AgentSession::start()` creates an active session with empty `work_log` and `artifacts` at `php/Models/AgentSession.php:126`.
- `AgentSession::logAction()` appends action, details, and timestamp to `work_log` at `php/Models/AgentSession.php:206`.
- `AgentSession::addWorkLogEntry()` appends message, type, data, and timestamp to `work_log` at `php/Models/AgentSession.php:223`.
- `AgentSession::end()` records terminal status, final summary, handoff notes, and end time at `php/Models/AgentSession.php:243`.
- `AgentSession::addArtifact()` records path, action, metadata, and timestamp at `php/Models/AgentSession.php:271`.
- `AgentSession::prepareHandoff()` stores summary, next steps, blockers, and context for next agent at `php/Models/AgentSession.php:310`.
- `AgentSession::getHandoffContext()` returns session identity, agent type, timestamps, context, recent actions, artefacts, and handoff notes at `php/Models/AgentSession.php:330`.
- `AgentSession::getReplayContext()` reconstructs checkpoints, decisions, errors, progress summary, artefacts, recent actions, handoff notes, and final summary from the stored session at `php/Models/AgentSession.php:355`.
- `AgentSession::createReplaySession()` creates a new active session with inherited context from the old session at `php/Models/AgentSession.php:464`.
- `AgentSessionService::start()` starts and caches sessions at `php/Services/AgentSessionService.php:33`.
- `AgentSessionService::resume()` reactivates paused or handed-off sessions at `php/Services/AgentSessionService.php:67`.
- `AgentSessionService::continueFrom()` creates a new session with previous handoff and inherited context at `php/Services/AgentSessionService.php:200`.
- `AgentSessionService::continueFrom()` marks the previous session handed off at `php/Services/AgentSessionService.php:227`.
- `AgentSessionService::getReplayContext()` returns reconstructed state from the session work log at `php/Services/AgentSessionService.php:299`.
- `AgentSessionService::replay()` creates and caches a replay session at `php/Services/AgentSessionService.php:316`.
- REST routes expose session list/show under `sessions.read` at `php/Routes/api.php:83`.
- REST routes expose session start/continue/end under `sessions.write` at `php/Routes/api.php:88`.
- `SessionController::store()` validates `agent_type`, `plan_slug`, and initial context at `php/Controllers/Api/SessionController.php:83`.
- `SessionController::continue()` creates a continuation session with a new `agent_type` at `php/Controllers/Api/SessionController.php:153`.
- `SessionController::end()` validates terminal status, summary, and handoff notes at `php/Controllers/Api/SessionController.php:120`.
- MCP tool registration includes `SessionStart`, `SessionEnd`, `SessionLog`, `SessionHandoff`, `SessionResume`, `SessionReplay`, `SessionContinue`, `SessionArtifact`, and `SessionList` at `php/Boot.php:218`.
- `SessionLog` requires active session state at `php/Mcp/Tools/Agent/Session/SessionLog.php:25`.
- `SessionLog` writes through `addWorkLogEntry()` at `php/Mcp/Tools/Agent/Session/SessionLog.php:85`.
- `SessionHandoff` prepares handoff with summary, next steps, blockers, and context at `php/Mcp/Tools/Agent/Session/SessionHandoff.php:77`.
- `SessionContinue` exposes inherited context, previous agent, and handoff notes in its result at `php/Mcp/Tools/Agent/Session/SessionContinue.php:55`.
- `SessionReplay` says it reconstructs state from work log for resume/handoff at `php/Mcp/Tools/Agent/Session/SessionReplay.php:10`.
- `SessionReplay` delegates to `AgentSessionService::getReplayContext()` at `php/Mcp/Tools/Agent/Session/SessionReplay.php:54`.
- `SessionArtifact` declares it records artefacts at `php/Mcp/Tools/Agent/Session/SessionArtifact.php:10`.
- `SessionArtifact` passes optional `description` into `addArtifact()` as the third argument at `php/Mcp/Tools/Agent/Session/SessionArtifact.php:73`.
- `addArtifact()` expects the third argument to be `?array $metadata` at `php/Models/AgentSession.php:272`, so the `SessionArtifact` MCP path can type-error when `description` is a string.
- `AgentPlan` has many sessions at `php/Models/AgentPlan.php:99`.
- `AgentPlan` has many workspace states at `php/Models/AgentPlan.php:104`.
- `AgentPlan::getState()` reads a state value by key at `php/Models/AgentPlan.php:236`.
- `AgentPlan::setState()` writes a state value by key, type, and description at `php/Models/AgentPlan.php:243`.
- `WorkspaceState` persists to `agent_workspace_states` at `php/Models/WorkspaceState.php:16`.
- `WorkspaceState` defines `TYPE_JSON`, `TYPE_MARKDOWN`, `TYPE_CODE`, and `TYPE_REFERENCE` at `php/Models/WorkspaceState.php:20`.
- `WorkspaceState` stores `agent_plan_id`, key, category, value, type, and description at `php/Models/WorkspaceState.php:28`.
- `WorkspaceState::forPlan()` scopes state to a plan at `php/Models/WorkspaceState.php:46`.
- `WorkspaceState::setValue()` updates or creates a key per plan at `php/Models/WorkspaceState.php:115`.
- `WorkspaceState::set()` and `WorkspaceState::get()` implement the RFC example shape at `php/Models/WorkspaceState.php:129`.
- The `agent_workspace_states` migration creates unique `(agent_plan_id, key)` values at `php/Migrations/0001_01_01_000003_create_agent_plans_tables.php:62`.
- The category migration adds a category column and plan/category index at `php/Migrations/2026_03_31_000002_add_category_to_agent_workspace_states.php:17`.
- MCP `StateSet` requires workspace context for tenant isolation at `php/Mcp/Tools/Agent/State/StateSet.php:21`.
- MCP `StateSet` writes state with plan slug, key, value, and category at `php/Mcp/Tools/Agent/State/StateSet.php:96`.
- MCP `StateGet` reads state by plan slug and key at `php/Mcp/Tools/Agent/State/StateGet.php:87`.
- MCP `StateList` lists all states for a plan and optional category at `php/Mcp/Tools/Agent/State/StateList.php:86`.
- Fleet routes expose register, heartbeat, deregister, assign, complete, next, events, and stats at `php/Routes/api.php:138`.
- Sync routes expose push, context pull, and sync status at `php/Routes/api.php:153`.
- `PushDispatchHistory` creates or finds a fleet node at `php/Actions/Sync/PushDispatchHistory.php:28`.
- `PushDispatchHistory` writes dispatch observations into `BrainMemory` at `php/Actions/Sync/PushDispatchHistory.php:51`.
- `PushDispatchHistory` records a sync record at `php/Actions/Sync/PushDispatchHistory.php:69`.
- `PushDispatchHistory` does not import or call `WorkspaceState`; its imports are `BrainMemory`, `FleetNode`, and `SyncRecord` at `php/Actions/Sync/PushDispatchHistory.php:7`.
- `PullFleetContext` reads latest active `BrainMemory` rows for a workspace at `php/Actions/Sync/PullFleetContext.php:28`.
- `PullFleetContext` returns memory MCP context values at `php/Actions/Sync/PullFleetContext.php:54`.
- `CompleteTask` persists fleet task result, findings, changes, report, and completion timestamp at `php/Actions/Fleet/CompleteTask.php:50`.
- `CompleteTask` awards credits for a completed fleet task at `php/Actions/Fleet/CompleteTask.php:65`.

### Gap assessment

- Core session lifecycle is implemented for local PHP persistence, REST, and MCP: start, log, artefact recording, handoff, continue, replay, and end are present in model/service/controller/tool code.
- WorkspaceState is implemented as plan-scoped typed state and exposed through MCP tools, satisfying the shared-per-plan state shape in `docs/php-agent/RFC.md:264`.
- End-to-end local-vs-fleet inheritance is incomplete because sync push writes BrainMemory but does not update WorkspaceState workflow progress, despite the RFC requirement at `docs/RFC-AGENT.md:994`.
- Fleet task lifecycle is implemented as task assignment/completion, but it is not linked to AgentSession records or session replay/handoff state in the audited fleet actions at `php/Actions/Fleet/AssignTask.php:40` and `php/Actions/Fleet/CompleteTask.php:50`.
- `SessionArtifact` likely has a runtime defect because it passes a string `description` to an `?array $metadata` parameter at `php/Mcp/Tools/Agent/Session/SessionArtifact.php:73` and `php/Models/AgentSession.php:272`.
- Test coverage confirms session start/log/artifact/handoff helpers at `php/tests/Feature/AgentSessionTest.php:38`, `php/tests/Feature/AgentSessionTest.php:152`, `php/tests/Feature/AgentSessionTest.php:201`, and `php/tests/Feature/AgentSessionTest.php:261`.
- Test coverage confirms replay context at `php/tests/Feature/SessionReplayTest.php:16`.
- Test coverage confirms WorkspaceState table, types, set/get helpers, and plan integration at `php/tests/Feature/WorkspaceStateTest.php:37`, `php/tests/Feature/WorkspaceStateTest.php:85`, `php/tests/Feature/WorkspaceStateTest.php:219`, and `php/tests/Feature/WorkspaceStateTest.php:291`.
- No inspected test covers sync writing WorkspaceState because `PushDispatchHistory` has no `WorkspaceState` dependency at `php/Actions/Sync/PushDispatchHistory.php:7`.

### Follow-up ticket scope

- Extend `/v1/agent/sync` so dispatch history updates both `BrainMemory` and `WorkspaceState` workflow progress, matching `docs/RFC-AGENT.md:994` and `docs/RFC-AGENT.md:1129`.
- Link fleet task assignment/completion to `AgentSession` creation, work log entries, artefacts, and replayable handoff context, or document fleet tasks as intentionally separate from session lifecycle.
- Fix `SessionArtifact` metadata typing and add a feature test for the MCP artefact tool path, using `php/Mcp/Tools/Agent/Session/SessionArtifact.php:73` as the regression point.

## Raised tickets

1. Implement PHP MetaReader and structural-signal pipeline precondition.
2. Refactor Forge scan and PR management away from body parsing.
3. Complete plugin restructure metadata: core-go/core-php/infra, marketplace YAML, and MCP command convention.
4. Resolve Claude/Codex/Gemini plugin family scope mismatch and missing Gemini plugin metadata.
5. Complete `/v1/agent/sync` WorkspaceState updates for fleet-shared workflow progress.
6. Connect fleet task lifecycle to AgentSession lifecycle or formalise the separation.
7. Fix `session_artifact` MCP metadata typing and add regression coverage.
