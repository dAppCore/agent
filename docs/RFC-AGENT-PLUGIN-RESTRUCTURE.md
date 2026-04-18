# Plugin Restructure: dappcore → core + API/MCP Integration

## Context

3 skeleton plugins (core-go, core-php, infra) need building out. The go-agent repo has 67 commands across 11 plugins that can enrich them. Plugins need configuring to work with `{api,mcp}.lthn.sh` endpoints (JSON via `Accept` header, default returns HTML).

## Step 1: Rename dappcore-go → core-go

**Files to modify:**
- `plugins/dappcore-go/.claude-plugin/plugin.json` — change name, update metadata
- Rename directory: `dappcore-go/` → `core-go/`

**Keep existing skills** (they're solid):
- `core/SKILL.md` — CLI reference & decision tree
- `core-go/SKILL.md` — Go framework patterns (pkg structure, CLI helpers, i18n, test naming)
- `go-agent/SKILL.md` — Autonomous dev workflow (7-step loop, PR management, CodeRabbit)

**Add from go-agent/claude/code:**
- `commands/qa.md` — QA fix loop (from code plugin, Go-specific)
- `commands/commit.md` — Smart conventional commit
- `commands/review.md` — Code review (from review plugin)
- `commands/verify.md` — Verification gate (from verify plugin)

**Add agents:**
- `agents/go-developer.md` — Go dev agent persona (derived from go-agent skill)

**Add:**
- `README.md`
- `marketplace.yaml` (template from agentic-flows)

## Step 2: Rename dappcore-php → core-php

**Files to modify:**
- `plugins/dappcore-php/.claude-plugin/plugin.json` — change name, update metadata
- Rename directory: `dappcore-php/` → `core-php/`

**Keep existing skills:**
- `core-php/SKILL.md` — Module structure, Boot class, Action pattern, multi-tenant
- `php-agent/SKILL.md` — Autonomous PHP dev workflow (TDD, CodeRabbit, issue loop)

**Add from go-agent/claude/code:**
- `commands/qa.md` — QA fix loop (PHP-specific: pest, pint, analyse)
- `commands/commit.md` — Smart conventional commit
- `commands/review.md` — Code review
- `commands/verify.md` — Verification gate

**Add agents:**
- `agents/php-developer.md` — PHP/Laravel dev agent persona

**Add:**
- `README.md`
- `marketplace.yaml`

## Step 3: Update infra plugin

**Keep existing skills** (content is detailed and good):
- `infra/SKILL.md` — Machine inventory, NOC services, network config
- `gitea/SKILL.md` — Forge/Forgejo CLI commands, org structure, mirrors

**Rename skill:** `agents/` → `brand/` (it's about Vi mascot & brand voice, not agent definitions)

**Add agents:**
- `agents/infra-ops.md` — Infrastructure operations agent

**Add from go-agent/claude/coolify:**
- `commands/deploy.md` — Service deployment
- `commands/status.md` — Deployment status check

**Add:**
- `README.md`
- `marketplace.yaml`

**Fix plugin.json:** Update skill references after rename

## Step 4: API/MCP endpoint configuration

Add a shared skill or pattern file that documents the endpoint convention for all plugins:

**Create `core-go/skills/api-endpoints/SKILL.md`** (and symlink or copy to core-php, infra):

Content covers:
- `api.lthn.sh` — REST API
- `mcp.lthn.sh` — MCP bridge endpoint
- **Must send `Accept: application/json`** — default returns HTML
- **Must send `Content-Type: application/json`** for POST bodies
- Auth: Bearer token in `Authorization` header
- REST convention: `/v1/{resource}`
- This is both OSS (people run their own lthn.sh) and production

**Update `.mcp.json`** in core-go and core-php to reference `core mcp serve` (same pattern as agentic-flows).

## Step 5: Add marketplace.yaml to all 3 plugins

Template from agentic-flows, adjusted per plugin:
```yaml
marketplace:
  registry: forge.lthn.ai
  organization: core
  repository: {plugin-name}
  auto_update: true
  check_interval: 24h
```

## Verification

1. Check plugin structure matches convention: `.claude-plugin/plugin.json` at root, commands/agents/skills at root level
2. Validate all SKILL.md files have proper YAML frontmatter
3. Validate all command .md files have proper frontmatter with name/description
4. Confirm no hardcoded paths (use `${CLAUDE_PLUGIN_ROOT}` where needed)
5. Test that `core mcp serve` still works with updated .mcp.json configs

## Out of Scope

- lethean & cryptonote-archive plugins (reference material)
- go-agent/claude/ plugins (stay in Go repo, not merged into shared plugins)
- EaaS subsystem references (stripped for OSS release)
- Codex/Gemini plugins (stay in go-agent)
