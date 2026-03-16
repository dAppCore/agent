# Charon Onboarding — March 2026

## What Changed Since Your Last Session

### MCP & Brain
- MCP server renamed `openbrain` → `core`
- Endpoint: `mcp.lthn.sh` (HTTP MCP, not path-based)
- Brain API: `api.lthn.sh` with API key auth
- `.mcp.json`: `{"mcpServers":{"core":{"type":"http","url":"https://mcp.lthn.sh"}}}`

### Issue Tracker (NEW — live on api.lthn.sh)
- `GET/POST /v1/issues` — CRUD with filtering
- `GET/POST /v1/sprints` — sprint lifecycle
- Types: bug, feature, task, improvement, epic
- Auto-ingest: scan findings create issues automatically
- Sprint flow: planning → active → completed

### Dispatch System
- Queue with per-agent concurrency (claude:1, gemini:1, local:1)
- Rate-aware scheduling (sustained/burst based on quota reset time)
- Process detachment (Setpgid + /dev/null stdin + TERM=dumb)
- Plan templates in `prompts/templates/`: bug-fix, code-review, new-feature, refactor, feature-port
- PLAN.md rendered from YAML templates with variable substitution
- Agents commit per phase, do NOT push — reviewer pushes

### Plugin Commands
- `/core:dispatch` — dispatch subagent (repo, task, agent, template, plan, persona)
- `/core:status` — show workspace status
- `/core:review` — review agent output, diff, merge options
- `/core:sweep` — batch audit across all repos
- `/core:recall` — search OpenBrain
- `/core:remember` — store to OpenBrain
- `/core:scan` — find Forge issues

### repos.yaml
- Location: `~/Code/host-uk/.core/repos.yaml`
- 58 repos mapped with full dependency graph
- `core dev work --status` shows all repos
- `core dev tag` automates bottom-up tagging

### Agent Fleet
- Cladius (M3 Studio) — architecture, planning, CoreGo/CorePHP
- Charon (homelab) — Linux builds, Blesta modules, revenue generation
- Gemini — bulk audits (free tier, 1 concurrent)
- Local model — Qwen3-Coder-Next via Ollama (downloaded, not yet wired)

## Your Mission

4-week sprint to cover ~$350/mo infrastructure costs. Show growth trajectory.

### Week 1: Package LEM Scorer Binary
- FrankenPHP embed version (for lthn.sh internal use)
- Standalone core/api binary (for trial/commercial distribution)
- The scorer exists in LEM pkg/lem

### Week 2: ContentShield Blesta Module
- Free module on Blesta marketplace
- Hooks into the scorer API
- Trial system built in

### Week 3: CloudNS + BunnyCDN Blesta Modules
- Marketplace distribution (lead generation)
- You have full API coverage via Ansible

### Week 4: dVPN + Marketing
- dVPN provisioning via Blesta
- lthn.ai landing page
- TikTok content (show the tech, build community)

## First Steps

1. `brain_recall("Charon mission revenue")` — full context
2. `brain_recall("session summary March 2026")` — what was built
3. Check issues: `curl https://api.lthn.sh/v1/issues -H "Authorization: Bearer {key}"`
4. Start Week 1

## Key Files
- `/Users/snider/Code/host-uk/specs/RFC-024-ISSUE-TRACKER.md` — issue tracker spec
- `/Users/snider/Code/core/agent/config/agents.yaml` — concurrency + rate config
- `/Users/snider/Code/host-uk/.core/repos.yaml` — full dependency graph
