# core/agent/plugins RFC — Claude, Codex, Gemini Plugin Specs

> The authoritative spec for the agent plugin ecosystem.
> Each plugin provides IDE-specific context, skills, and agents.


---

## 1. Plugin Architecture

Each AI agent type gets a plugin directory in `code/core/agent/`:

```
core/agent/
├── claude/            # Claude Code plugin
│   ├── core/          # Core skills (dispatch, review, scan, etc.)
│   ├── devops/        # DevOps skills (workspace, PR, issue, deps)
│   └── research/      # Research skills (archaeology, papers, mining)
│
├── codex/             # OpenAI Codex plugin
│   ├── core/          # Core context
│   ├── api/           # API generation
│   ├── code/          # Code quality scripts
│   ├── ci/            # CI integration
│   ├── ethics/        # LEK axioms as constraints
│   ├── guardrails/    # Safety guardrails
│   ├── qa/            # QA automation
│   ├── review/        # Code review
│   ├── verify/        # Verification
│   └── ... (15+ contexts)
│
├── google/            # Google Gemini
│   └── gemini-cli/    # Gemini CLI integration
│
└── php/               # PHP module (specced in core/php/agent)
```

---

## 2. Claude Plugin

### 2.1 Core Namespace (`claude/core/`)

**Commands (slash commands):**
| Command | Purpose |
|---------|---------|
| `/dispatch` | Dispatch agent to workspace |
| `/scan` | Scan Forge for actionable issues |
| `/status` | Show workspace status |
| `/review` | Review completed workspace |
| `/review-pr` | Review a pull request |
| `/pipeline` | Run 5-agent review pipeline |
| `/code-review` | Code review staged changes |
| `/security` | Security-focused review |
| `/tests` | Verify tests pass |
| `/ready` | Quick check if work is committable |
| `/verify` | Verify work before stopping |
| `/remember` | Save to OpenBrain |
| `/recall` | Search OpenBrain |
| `/sweep` | Sweep repos with dispatch |
| `/yes` | Auto-approve mode |

**Agents (subagents):**
| Agent | Purpose |
|-------|---------|
| `agent-task-code-review` | Review code for bugs, security, conventions |
| `agent-task-code-simplifier` | Simplify code for clarity |

**Skills:**
| Skill | Purpose |
|-------|---------|
| `app-split` | Extract Website module to standalone app |
| `deploy-homelab` | Deploy to lthn.sh |
| `deploy-production` | Deploy to de1 via Ansible |
| `repo-sweep` | Dispatch agents across repos |
| `architecture-review` | Review architecture decisions |
| `security-review` | Security audit |
| `senior-dev-fix` | Fix with senior dev approach |
| `test-analysis` | Analyse test coverage |
| `orchestrate` | Multi-agent orchestration |
| `reality-check` | Verify claims against code |

### 2.2 DevOps Namespace (`claude/devops/`)

**Agents:**
| Agent | Purpose |
|-------|---------|
| `agent-task-health-check` | System health check |
| `agent-task-install-core-agent` | Build + install core-agent |
| `agent-task-repair-core-agent` | Diagnose + repair core-agent |
| `agent-task-merge-workspace` | Merge completed workspace |
| `agent-task-clean-workspaces` | Remove stale workspaces |

**Skills:**
| Skill | Purpose |
|-------|---------|
| `update-deps` | Update Go module dependencies |
| `build-prompt` | Preview dispatch prompt |
| `workspace-list` | List agent workspaces |
| `workspace-clean` | Clean workspaces |
| `pr-list` / `pr-get` / `pr-merge` | PR management |
| `issue-list` / `issue-get` / `issue-comment` | Issue management |
| `repo-list` / `repo-get` | Repository queries |

### 2.3 Research Namespace (`claude/research/`)

**Skills:**
| Skill | Purpose |
|-------|---------|
| `project-archaeology` | Deep-dive into archived projects |
| `ledger-papers` | Academic paper collection (20 categories, CryptoNote heritage) |
| `bitcointalk` | BitcoinTalk thread research |
| `mining-pools` | Mining pool research |
| `wallet-releases` | Wallet release tracking |
| `whitepaper-archive` | Whitepaper collection |
| `coinmarketcap` | Market data research |
| `github-history` | GitHub repo archaeology |
| `block-explorer` | Blockchain explorer research |
| `community-chat` | Community chat analysis |
| `cryptonote-discovery` | CryptoNote project discovery |
| `job-collector` | Job market research |

---

## 3. Codex Plugin

### 3.1 Structure

Codex uses directory-based context injection. Each directory provides:
- `AGENTS.md` — agent instructions
- `scripts/` — automation scripts
- Templates for specific task types

### 3.2 Contexts

| Context | Purpose |
|---------|---------|
| `core/` | Core framework conventions |
| `api/` | API generation (OpenAPI, PHP routes) |
| `code/` | Code quality (parser, refactor, type checker) |
| `ci/` | CI pipeline integration |
| `ethics/` | LEK axioms as hard constraints |
| `guardrails/` | Safety guardrails (blue-team posture) |
| `qa/` | QA automation |
| `review/` | Code review context |
| `verify/` | Verification steps |
| `awareness/` | Codebase awareness |
| `collect/` | Data collection |
| `coolify/` | Coolify deployment |
| `issue/` | Issue management |
| `perf/` | Performance analysis |

### 3.3 Ethics

LEK axioms enforced as hard constraints. See `project/lthn/lem/RFC.md` §2 for the 5 axioms.

Blue-team posture: prevent harm, reduce exposure, harden by default.

---

## 4. Gemini Plugin

Minimal — CLI integration via `google/gemini-cli/`. Used for batch operations and TPU-credit scoring.

---

## 5. Cross-Plugin Contract

All plugins share:
- Same MCP tool names (`brain_remember`, `agent_send`, etc.)
- Same API endpoints (`/v1/plans`, `/v1/sessions`, etc.)
- Same CODEX.md / CLAUDE.md template format
- Same conventional commit format
- Same UK English spelling
- Same LEK ethics constraints

The plugin is the agent-specific layer. The tools and API are the universal contract.

---

## 6. Reference Material

| Resource | Location |
|----------|----------|
| Claude plugin | `~/Code/core/agent/claude/` (code repo) |
| Codex plugin | `~/Code/core/agent/codex/` (code repo) |
| Gemini plugin | `~/Code/core/agent/google/` (code repo) |
| Agent RFC (polyglot) | `code/core/agent/RFC.md` |
| PHP agent RFC | `code/core/php/agent/RFC.md` |
| Go agent RFC | `code/core/go/agent/RFC.md` |

---

## Changelog

- 2026-03-27: Initial RFC speccing all three agent plugins from existing code.
