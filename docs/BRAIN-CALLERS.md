<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Brain API Callers

Date: 2026-04-25  
Ticket: Mantis #179  
Companion audit: `docs/brain-callers-audit.md` (broad sweep), this file is the focused living map for Brain callers and contracts.

## Purpose

This document records who calls Brain APIs in this workspace, which endpoint or in-process action they use, what protections sit on that path, and what request/response shape each caller expects.

Future Brain call sites should be added here in the same change that introduces them.

## Recent Hardening To Preserve

- `#312` org auth: `php/Services/BrainService.php` rejects unauthorised org scopes before remember, recall, and forget proceed.
- `#1006` org bounds: PHP service and MCP schemas bound org values to 128 characters.
- `#998` `brain.key` perms: the shared Go client writes `~/.claude/brain.key` as `0600` and rejects insecure existing permissions.
- `#1052` bearer SSRF: the shared Go client rejects absolute request URLs and only appends relative `/v1/brain/*` paths to the configured API base URL.
- `#1055` retry jitter: the shared Go client retries `408`, `429`, and `5xx` with `Retry-After` support plus full-jitter backoff and a circuit breaker.

## Canonical Contract Surfaces

### HTTP `/v1/brain/*`

| Endpoint | Current request shape | Current success shape | Current error shape | Notes |
| --- | --- | --- | --- | --- |
| `POST /v1/brain/remember` | `content`, `type`, `tags?`, `project?`, `confidence?`, `supersedes?`, `expires_in?` | `201 {"data": <memory>}` | `422 {"error":"validation_error","message":...}`, `503 {"error":"service_error","message":...}` | The controller currently does not validate or forward `org`, so external HTTP callers cannot rely on org-scoped remember yet. |
| `POST /v1/brain/recall` | `query`, `limit?`, `top_k?`, `org?`, `project?`, `type?`, `keywords?`, `boost_keywords?`, `filter?` | `200 {"data":{"memories":[...],"scores":{...},"count":n}}` | `422 {"error":"validation_error","message":...}`, `503 {"error":"service_error","message":...}` | This is the current HTTP route that actually models org-aware recall. |
| `DELETE /v1/brain/forget/{id}` | path `id`, optional JSON `reason` | `200 {"data": {...}}` | `404 {"error":"not_found","message":...}`, `503 {"error":"service_error","message":...}` | Forget runs through workspace and org checks in `ForgetKnowledge` and `BrainService`. |
| `GET /v1/brain/list` | `project?`, `type?`, `agent_id?`, `limit?` | `200 {"data":{"memories":[...],"count":n}}` | `422 {"error":"validation_error","message":...}` | The controller currently does not validate `org`, even though the PHP MCP tool and shared Go client both model org-filtered list calls. |
| `GET /v1/brain/search` | `q`, `org?`, `project?`, `limit?` | `200 {"data":{"memories":[...],"count":n}}` | `503 {"error":"service_error","message":...}` | Search is PHP-only in this repo; no Go caller was found here. |
| `GET /v1/brain/tags` | none | `200 {"data": {"tag": count}}` | `503 {"error":"service_error","message":...}` | PHP-only read endpoint over Elasticsearch aggregates. |
| `GET /v1/brain/scopes` | none | `200 {"data": {"org":{"project":count}}}` | `503 {"error":"service_error","message":...}` | PHP-only read endpoint over Elasticsearch aggregates. |

### In-process PHP service and actions

`php/Services/BrainService.php` is the canonical PHP write/read implementation behind the controller, MCP tools, console commands, and Livewire explorer.

- Remember input shape:
  `workspace_id`, `agent_id`, `type`, `content`, `tags?`, `org?`, `project?`, `confidence?`, `supersedes_id?`, `expires_at?`, `source?`
- Recall input shape:
  `query`, `topK`, `filter`, `workspaceId`, optional `keywords`, optional `boostKeywords`
- Remember/recall protections:
  workspace-scoped persistence, org-scope authorisation, bounded retry against Qdrant and Elasticsearch, `Retry-After` support, capped retry delay (`30s`), and queue-based embedding/index cleanup
- Forget protections:
  workspace ownership check, org-scope authorisation, DB delete plus async index cleanup

### Shared Go client

The canonical Go client lives in module `dappco.re/go/mcp/pkg/mcp/brain/client`, which resolves to `../mcp/pkg/mcp/brain/client/client.go` in this workspace via `go.mod`.

- Typed inputs:
  `RememberInput`, `RecallInput`, `ForgetInput`, `ListInput`
- Success surface:
  parsed `map[string]any`
- Error surface:
  non-2xx becomes an error; the upstream JSON body is preserved inside the error text, not returned as a parsed map
- Protections:
  Bearer auth, default org and agent injection on typed calls, `https` enforcement unless `CORE_BRAIN_INSECURE=true`, `brain.key` `0600` enforcement, absolute-URL rejection, `408/429/5xx` retry, `Retry-After` handling, full-jitter backoff, circuit breaker

## PHP Callers

### HTTP controller

| Call site | Endpoint(s) | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `php/Controllers/Api/BrainController.php` | `remember`, `recall`, `forget`, `list`, `search`, `tags`, `scopes` | `AgentApiAuth` permission checks (`brain.read` or `brain.write`), Bearer auth, workspace binding from API key, rate-limit headers, downstream org auth in `BrainService` | Route-specific JSON and query validation; see HTTP contract table above | Returns wrapped JSON under `data` on success. `remember` and `list` are not yet fully aligned with the org-aware service/client contract. |

### MCP tools

| Call site | Endpoint / action | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `php/Mcp/Tools/Agent/Brain/BrainRemember.php` | in-process `RememberKnowledge::run()` | workspace context dependency, MCP circuit breaker wrapper, org max length `128`, service-level org auth and retry | `content`, `type`, `tags?`, `org?`, `project?`, `confidence?`, `supersedes?`, `expires_in?` | `{"success":true,"memory":{...}}` |
| `php/Mcp/Tools/Agent/Brain/BrainRecall.php` | in-process `RecallKnowledge::run()` | workspace context dependency, MCP circuit breaker wrapper, org max length `128`, service-level org auth and retry | `query`, `top_k?`, `filter?` where filter can include `org`, `project`, `type`, `agent_id`, `min_confidence` | `{"success":true,"count":n,"memories":[...],"scores":{...}}` |
| `php/Mcp/Tools/Agent/Brain/BrainList.php` | in-process `BrainMemory` query | workspace context dependency, no external HTTP, org max length `128` | `org?`, `project?`, `type?`, `agent_id?`, `limit?` | `{"success":true,"memories":[...],"count":n}` |
| `php/Mcp/Tools/Agent/Brain/BrainForget.php` | in-process `ForgetKnowledge::run()` | workspace context dependency, MCP circuit breaker wrapper, service-level org auth | `id`, `reason?` | `{"success":true,...}` |

### Console and UI callers

| Call site | Endpoint / action | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `php/Console/Commands/BrainSeedMemoryCommand.php` | direct `BrainService::remember()` | service validation, org auth, retry, queued indexing | `workspace_id`, `agent_id`, `type`, `content`, `tags`, `project`, `confidence` | Side-effect only; logs imported/skipped counts |
| `php/Console/Commands/BrainIngestCommand.php` | direct `BrainService::remember()` | same as above | same remember shape plus optional `source` | Side-effect only; truncates oversized text before embedding |
| `php/Console/Commands/PrepWorkspaceCommand.php` | direct `BrainService::recall()` | service validation, org auth, retry | repo query plus optional issue query, `topK`, filters, `workspaceId` | Builds `CONTEXT.md` from `memories` and `scores` |
| `php/Agentic/Livewire/BrainExplorer.php` | `RecallKnowledge::run()`, `ListKnowledge::run()`, `ForgetKnowledge::run()` | workspace validation, service/org checks, DB fallback on recall failure | query plus list/recall filters | UI-normalised memory cards; fallback search if vector recall fails |

## Go Callers

### Shared client consumers in this repo

| Call site | Endpoint(s) | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `pkg/agentic/brain_client.go` | arbitrary relative `/v1/brain/*` via shared client `Call()` | shared Go client protections plus helper-level `CORE_BRAIN_ORG` injection when body omits `org`; subsystem-scoped shared circuit breaker | raw `map[string]any` body | Returns `core.Result`; callers use `brainPayloadMap()` to unwrap `data` when present |
| `pkg/agentic/prep.go` | `POST /v1/brain/recall` | helper above | `query`, `top_k`, `project`, `agent_id`; helper can inject `org` | Reads `data.memories` to build workspace context |
| `pkg/agentic/session.go` | `POST /v1/brain/remember` | helper above | `content`, `agent_id`, `type`, `tags`, `confidence`, optional `project` | Best-effort session handoff persistence; failures are logged and ignored |
| `pkg/agentic/brain_seed_memory.go` | `POST /v1/brain/remember` | helper above | `workspace_id`, `agent_id`, `type`, `content`, `tags`, `project`, `confidence` | Best-effort import; failures are counted as skipped |

### Go MCP brain subsystem

| Call site | Endpoint(s) | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `pkg/brain/direct.go` | `POST /v1/brain/remember`, `POST /v1/brain/recall`, `DELETE /v1/brain/forget/{id}`, `GET /v1/brain/list` | shared Go client protections, env-based default org, `~/.claude/brain.key` fallback, Bearer auth | remember: `content`, `type`, `tags?`, `org?`, `project?`, `confidence?`, `supersedes?`, `expires_in?`, `agent_id`; recall: `query`, `top_k`, optional org/project/type/agent_id/min_confidence`; list: query params `org`, `project`, `type`, `agent_id`, `limit` | Exposes MCP tool responses such as `RememberOutput{Success, MemoryID}` and `RecallOutput{Count, Memories}` |

## Scripts And Plugin Callers

### Python plugins

| Call site | Endpoint(s) | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `hermes/plugins/openbrain_memory.py` | `remember`, `recall`, `forget`, `list` | Bearer auth header, optional default `org`, optional default `workspace_id`, async background write dispatch for turn sync | remember/list/recall/forget payloads are forwarded largely as-is after empty-value cleanup | Returns decoded JSON plus `status`; no shared breaker, no shared retry/jitter, no absolute-URL guard |
| `hermes/plugins/openbrain_context.py` | `POST /v1/brain/recall` | Bearer auth header, default `workspace_id`, default `org` in `filter` | `{"query":..., "top_k":..., "filter":{"workspace_id":...,"org":...}}` | Accepts several response layouts (`data.memories`, `results`, `items`, `matches`) and normalises candidates locally; no shared breaker or retry |

### Shell scripts

| Call site | Endpoint(s) | Protections | Input shape | Output shape / notes |
| --- | --- | --- | --- | --- |
| `claude/core/scripts/session-start.sh` | `POST /v1/brain/recall` | Bearer auth header, loads `~/.claude/brain.key`, short `curl --max-time` | raw JSON body with `query`, `top_k`, `agent_id`, optional inline `project` or `type` fragments | Parses JSON on stdout; no shared org injection, no retry, no breaker, no SSRF guard |
| `claude/core/scripts/session-save.sh` | `POST /v1/brain/remember` | Bearer auth header, `brain.key` fallback, debounce before write | raw JSON body with `content`, `type`, `project`, `agent_id`, `tags` | Fire-and-forget autosave; no org, no retry, no breaker |
| `claude/core/scripts/pre-compact.sh` | `POST /v1/brain/remember` | Bearer auth header, `brain.key` fallback | raw JSON body with `content`, `type`, `project`, `agent_id`, `tags` | Fire-and-forget compaction snapshot; no org, no retry, no breaker |

## Non-runtime References

- `plugins/core-go/skills/api-endpoints/SKILL.md`
- `plugins/core-php/skills/api-endpoints/SKILL.md`

These are documentation/examples only. They are not runtime callers, but they can still become copy-paste bypasses if they drift away from the hardened shared-client path.

## Contract-Test Follow-up For Part B

Part B was not implemented in this lane because the current HTTP controller surface is not yet fully aligned with the service and shared-client contract that the test needs to lock down.

- `POST /v1/brain/remember` currently drops `org` at controller validation time, so a PHP endpoint test cannot truthfully assert the same org-aware remember contract that the service and Go client model.
- `GET /v1/brain/list` currently omits `org` from controller validation even though the PHP MCP tool and shared Go client both model org-filtered list requests.
- The shared Go client correctly preserves upstream error JSON inside the error text, but it does not currently expose non-2xx bodies as parsed structured data, so an "identical error shape" assertion needs either a small shared wrapper or a raw HTTP harness.

Recommended follow-up before adding the cross-runtime contract test:

1. Align `BrainController::remember()` with the org-aware remember contract.
2. Align `BrainController::list()` with the org-aware list contract.
3. Add a PHP route-level Pest test and a Go shared-client integration test that both use the same `remember(core)` and `remember(evil)` fixtures once the HTTP contract is aligned.
