---
name: flow-issue-orchestrator
description: Use when onboarding a repo into the agentic pipeline. End-to-end flow covering audit → epic → execute for a complete repository transformation.
---

# Flow: Issue Orchestrator

End-to-end pipeline that takes a repo from raw audit findings to running epics with agents. Sequences three flows: **audit-issues** → **create-epic** → **issue-epic**.

---

## When to Use

- Onboarding a new repo into the agentic pipeline
- Processing accumulated audit issues across the org
- Bootstrapping epics for repos that have open issues but no structure

## Pipeline Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  STAGE 1: AUDIT                          flow: audit-issues     │
│  ───────────────                                                │
│  Input:  Repo with [Audit] issues                               │
│  Output: Implementation issues (1 per finding)                  │
│                                                                 │
│  - Read each audit issue                                        │
│  - Classify findings (severity, type, scope, complexity)        │
│  - Create one issue per finding                                 │
│  - Detect patterns (3+ similar → framework issue)               │
│  - Close audit issues, link to children                         │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  STAGE 2: ORGANISE                       flow: create-epic      │
│  ─────────────────                                              │
│  Input:  Repo with implementation issues (from Stage 1)         │
│  Output: Epic issues with children, branches, phase ordering    │
│                                                                 │
│  - Group issues by theme (security, quality, testing, etc.)     │
│  - Order into phases (blockers → parallel → cleanup)            │
│  - Create epic parent issue with checklist                      │
│  - Link children to parent                                      │
│  - Create epic branch off default branch                        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  STAGE 3: EXECUTE                        flow: issue-epic       │
│  ────────────────                                               │
│  Input:  Epic with children, branch, phase ordering             │
│  Output: Merged PRs, closed issues, training data               │
│                                                                 │
│  - Dispatch Phase 1 blockers to agents (add label)              │
│  - Monitor: CI, reviews, conflicts, merges                      │
│  - Intervene: "fix code reviews" / "fix merge conflict"         │
│  - Resolve threads, update branches, tick parent checklist      │
│  - When phase complete → dispatch next phase                    │
│  - When epic complete → merge epic branch to dev                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Running the Pipeline

### Prerequisites

- `gh` CLI authenticated with org access
- Agent label exists in the repo (e.g. `jules`)
- Repo has CI configured (or agent handles it)
- CODEOWNERS configured for auto-review requests

### Stage 1: Audit → Implementation Issues

For each repo with `[Audit]` issues:

```bash
# 1. List audit issues
gh issue list --repo dappcore/REPO --state open \
  --json number,title --jq '.[] | select(.title | test("\\[Audit\\]|audit:"))'

# 2. For each audit issue, run the audit-issues flow:
#    - Read the audit body
#    - Classify each finding
#    - Create implementation issues
#    - Detect patterns → create framework issues
#    - Close audit, link to children

# 3. Verify: count new issues created
gh issue list --repo dappcore/REPO --state open --label audit \
  --json number --jq 'length'
```

**Agent execution:** This stage can be delegated to a subagent with the audit-issues flow as instructions. The subagent reads audit content (allowed — it's creating issues, not orchestrating PRs) and creates structured issues.

```bash
# Example: task a subagent to process all audits in a repo
# Prompt: "Run RFC.flow-audit-issues.md on dappcore/REPO.
#          Process all [Audit] issues. Create implementation issues.
#          Detect patterns. Create framework issues if 3+ similar."
```

### Stage 2: Group into Epics

After Stage 1 produces implementation issues:

```bash
# 1. List all open issues (implementation issues from Stage 1 + any pre-existing)
gh issue list --repo dappcore/REPO --state open \
  --json number,title,labels --jq 'sort_by(.number) | .[]'

# 2. Check for existing epics
gh search issues --repo dappcore/REPO --state open --json number,title,body \
  --jq '.[] | select(.body | test("- \\[[ x]\\] #\\d+")) | {number, title}'

# 3. Group issues by theme, create epics per create-epic flow:
#    - Create epic parent issue with checklist
#    - Link children to parent (comment "Parent: #EPIC")
#    - Create epic branch: epic/<number>-<slug>

# 4. Verify: epic exists with children
gh issue view EPIC_NUMBER --repo dappcore/REPO
```

**Grouping heuristics:**

| Signal | Grouping |
|--------|----------|
| Same `audit` label + security theme | → Security epic |
| Same `audit` label + quality theme | → Quality epic |
| Same `audit` label + testing theme | → Testing epic |
| Same `audit` label + docs theme | → Documentation epic |
| All audit in small repo (< 5 issues) | → Single audit epic |
| Feature issues sharing a subsystem | → Feature epic |

**Small repos (< 5 audit issues):** Create one epic per repo covering all audit findings. No need to split by theme.

**Large repos (10+ audit issues):** Split into themed epics (security, quality, testing, docs). Each epic should have 3-10 children.

### Stage 3: Dispatch and Execute

After Stage 2 creates epics:

```bash
# 1. For each epic, dispatch Phase 1 blockers:
gh issue edit CHILD_NUM --repo dappcore/REPO --add-label jules
gh issue comment CHILD_NUM --repo dappcore/REPO \
  --body "Target branch: \`epic/EPIC_NUMBER-SLUG\` (epic #EPIC_NUMBER)"

# 2. Monitor and intervene per issue-epic flow
# 3. When Phase 1 complete → dispatch Phase 2
# 4. When all phases complete → merge epic branch to dev
```

**IMPORTANT:** Adding the `jules` label costs 1 daily task (300/day). Calculate total dispatch cost before starting:

```bash
# Count total children across all epics about to be dispatched
TOTAL=0
for EPIC in NUM1 NUM2 NUM3; do
  COUNT=$(gh issue view $EPIC --repo dappcore/REPO --json body --jq \
    '[.body | split("\n")[] | select(test("^- \\[ \\] #"))] | length')
  TOTAL=$((TOTAL + COUNT))
  echo "Epic #$EPIC: $COUNT children"
done
echo "Total dispatch cost: $TOTAL tasks"
```

---

## Repo Inventory

Current state of repos needing orchestration (as of 2026-02-04):

| Repo | Open | Audit | Epics | Default Branch | Stage |
|------|------|-------|-------|----------------|-------|
| `core` | 40+ | 0 | 8 (#101,#118,#127,#133,#299-#302) | `dev` | Stage 3 (executing) |
| `core-php` | 28 | 15 | 0 | `dev` | **Stage 1 ready** |
| `core-claude` | 30 | 0 | 0 | `dev` | Stage 2 (features, no audits) |
| `core-api` | 22 | 3 | 0 | `dev` | **Stage 1 ready** |
| `core-admin` | 14 | 2 | 0 | `dev` | **Stage 1 ready** |
| `core-mcp` | 24 | 5 | 0 | `dev` | **Stage 1 ready** |
| `core-tenant` | 14 | 2 | 0 | `dev` | **Stage 1 ready** |
| `core-developer` | 19 | 2 | 0 | `dev` | **Stage 1 ready** |
| `core-service-commerce` | 30 | 2 | 0 | `dev` | **Stage 1 ready** |
| `core-devops` | 3 | 1 | 0 | `dev` | **Stage 1 ready** |
| `core-agent` | 14 | 0 | 0 | `dev` | Stage 2 (features, no audits) |
| `core-template` | 12 | 1 | 0 | `dev` | **Stage 1 ready** |
| `build` | 9 | 1 | 0 | `dev` | **Stage 1 ready** |
| `ansible-coolify` | 1 | 1 | 0 | `main` | **Stage 1 ready** |
| `docker-server-php` | 1 | 1 | 0 | `main` | **Stage 1 ready** |
| `docker-server-blockchain` | 1 | 1 | 0 | `main` | **Stage 1 ready** |

### Priority Order

Process repos in this order (most issues = most value from epic structure):

```
Tier 1 — High issue count, audit-ready:
  1. core-php          (28 open, 15 audit → 1-2 audit epics)
  2. core-mcp          (24 open, 5 audit → 1 audit epic)
  3. core-api          (22 open, 3 audit → 1 audit epic)

Tier 2 — Medium issue count:
  4. core-developer    (19 open, 2 audit → 1 small epic)
  5. core-admin        (14 open, 2 audit → 1 small epic)
  6. core-tenant       (14 open, 2 audit → 1 small epic)

Tier 3 — Feature repos (no audits, skip Stage 1):
  7. core-claude       (30 open, 0 audit → feature epics via Stage 2)
  8. core-agent        (14 open, 0 audit → feature epics via Stage 2)

Tier 4 — Small repos (1-2 audit issues, single epic each):
  9. core-service-commerce (30 open, 2 audit)
  10. core-template     (12 open, 1 audit)
  11. build             (9 open, 1 audit)
  12. core-devops       (3 open, 1 audit)
  13. ansible-coolify   (1 open, 1 audit)
  14. docker-server-php (1 open, 1 audit)
  15. docker-server-blockchain (1 open, 1 audit)
```

---

## Full Repo Onboarding Sequence

Step-by-step for onboarding a single repo:

```bash
REPO="dappcore/REPO_NAME"
ORG="dappcore"

# ─── STAGE 1: Process Audits ───

# List audit issues
AUDITS=$(gh issue list --repo $REPO --state open \
  --json number,title --jq '.[] | select(.title | test("\\[Audit\\]|audit:")) | .number')

# For each audit, create implementation issues (run audit-issues flow)
for AUDIT in $AUDITS; do
  echo "Processing audit #$AUDIT..."
  # Subagent or manual: read audit, classify, create issues
  # See RFC.flow-audit-issues.md for full process
done

# Verify implementation issues created
gh issue list --repo $REPO --state open --json number,title,labels \
  --jq '.[] | "\(.number)\t\(.title)"'

# ─── STAGE 2: Create Epics ───

# List all open issues for grouping
gh issue list --repo $REPO --state open --json number,title,labels \
  --jq 'sort_by(.number) | .[] | "\(.number)\t\(.title)\t\(.labels | map(.name) | join(","))"'

# Group by theme, create epic(s) per create-epic flow
# For small repos: 1 epic covering everything
# For large repos: split by security/quality/testing/docs

# Get default branch SHA
DEFAULT_BRANCH="dev"  # or "main" for infra repos
SHA=$(gh api repos/$REPO/git/refs/heads/$DEFAULT_BRANCH --jq '.object.sha')

# Create epic issue (fill in children from grouping)
EPIC_URL=$(gh issue create --repo $REPO \
  --title "epic(audit): Audit findings implementation" \
  --label "agentic,complexity:large" \
  --body "BODY_WITH_CHILDREN")
EPIC_NUMBER=$(echo $EPIC_URL | grep -o '[0-9]*$')

# Link children
for CHILD in CHILD_NUMBERS; do
  gh issue comment $CHILD --repo $REPO --body "Parent: #$EPIC_NUMBER"
done

# Create epic branch
gh api repos/$REPO/git/refs -X POST \
  -f ref="refs/heads/epic/$EPIC_NUMBER-audit" \
  -f sha="$SHA"

# ─── STAGE 3: Dispatch ───

# Label Phase 1 blockers for agent dispatch
for BLOCKER in PHASE1_NUMBERS; do
  gh issue edit $BLOCKER --repo $REPO --add-label jules
  gh issue comment $BLOCKER --repo $REPO \
    --body "Target branch: \`epic/$EPIC_NUMBER-audit\` (epic #$EPIC_NUMBER)"
done

# Monitor via issue-epic flow
echo "Epic #$EPIC_NUMBER dispatched. Monitor via issue-epic flow."
```

---

## Parallel Repo Processing

Multiple repos can be processed simultaneously since they're independent. The constraint is agent quota, not repo count.

### Budget Planning

```
Daily Jules quota: 300 tasks
Tasks used today:  N

Available for dispatch:
  Tier 1 repos: ~15 + 5 + 3 = 23 audit issues → ~50 implementation issues
  Tier 2 repos: ~2 + 2 + 2 = 6 audit issues → ~15 implementation issues
  Tier 4 repos: ~8 audit issues → ~20 implementation issues

  Total potential children: ~85
  Dispatch all Phase 1 blockers: ~15-20 tasks (1 per epic)
  Full dispatch all children: ~85 tasks
```

### Parallel Stage 1 (safe — no agent cost)

Stage 1 (audit processing) is free — it creates issues, doesn't dispatch agents. Run all repos in parallel:

```bash
# Subagent per repo — all can run simultaneously
for REPO in core-php core-mcp core-api core-admin core-tenant \
            core-developer core-service-commerce core-devops \
            core-template build ansible-coolify \
            docker-server-php docker-server-blockchain; do
  echo "Subagent: run audit-issues on dappcore/$REPO"
done
```

### Parallel Stage 2 (safe — no agent cost)

Stage 2 (epic creation) is also free. Run after Stage 1 completes per repo.

### Controlled Stage 3 (costs agent quota)

Stage 3 dispatch is where budget matters. Options:

| Strategy | Tasks/day | Throughput | Risk |
|----------|-----------|------------|------|
| Conservative | 10-20 | 2-3 repos | Low — room for retries |
| Moderate | 50-80 | 5-8 repos | Medium — watch for cascade conflicts |
| Aggressive | 150-200 | 10+ repos | High — little room for iteration |

**Recommended:** Start conservative. Dispatch 1 epic per Tier 1 repo (3 epics, ~10 Phase 1 blockers). Monitor for a day. If agents handle well, increase.

---

## Testing the Pipeline

### Test Plan: Onboard Tier 1 Repos

Run the full pipeline on `core-php`, `core-mcp`, and `core-api` to validate the process before scaling to all repos.

#### Step 1: Audit Processing (Stage 1)

```bash
# Process each repo's audit issues — can run in parallel
# These are subagent tasks, each gets the audit-issues flow as instructions

# core-php: 15 audit issues (largest, best test case)
# Prompt: "Run RFC.flow-audit-issues.md on dappcore/core-php"

# core-mcp: 5 audit issues
# Prompt: "Run RFC.flow-audit-issues.md on dappcore/core-mcp"

# core-api: 3 audit issues
# Prompt: "Run RFC.flow-audit-issues.md on dappcore/core-api"
```

#### Step 2: Epic Creation (Stage 2)

After Stage 1, group issues into epics:

```bash
# core-php: 15 audit issues → likely 2-3 themed epics
#   Security epic, Quality epic, possibly Testing epic

# core-mcp: 5 audit issues → 1 audit epic
#   All findings in single epic

# core-api: 3 audit issues → 1 audit epic
#   All findings in single epic
```

#### Step 3: Dispatch (Stage 3)

```bash
# Start with 1 blocker per epic to test the flow
# core-php epic(s): 2-3 blockers dispatched
# core-mcp epic: 1 blocker dispatched
# core-api epic: 1 blocker dispatched
# Total: ~5 tasks from Jules quota
```

#### Step 4: Validate

After first round of PRs arrive:

- [ ] PRs target correct epic branches
- [ ] CI runs and agent fixes failures
- [ ] Reviews arrive (Copilot, CodeRabbit)
- [ ] "Fix code reviews" produces fix commit
- [ ] Thread resolution works
- [ ] Auto-merge completes
- [ ] Parent checklist updated

### Test Plan: PHP Repos (Laravel)

PHP repos use Composer + Pest instead of Go + Task. Verify:

- [ ] CI triggers correctly (different workflow)
- [ ] Agent understands PHP codebase (Pest tests, Pint formatting)
- [ ] `lang:php` label applied to issues
- [ ] Epic branch naming works the same way

---

## Monitoring

### Daily Check

```bash
# Quick status across all repos with epics
for REPO in core core-php core-mcp core-api; do
  OPEN=$(gh issue list --repo dappcore/$REPO --state open --json number --jq 'length')
  PRS=$(gh pr list --repo dappcore/$REPO --state open --json number --jq 'length')
  echo "$REPO: $OPEN open issues, $PRS open PRs"
done
```

### Epic Progress

```bash
# Check epic completion per repo
EPIC=299
REPO="dappcore/core"
gh issue view $EPIC --repo $REPO --json body --jq '
  .body | split("\n") | map(select(test("^- \\[[ x]\\] #"))) |
  { total: length,
    done: map(select(test("^- \\[x\\] #"))) | length,
    remaining: map(select(test("^- \\[ \\] #"))) | length }'
```

### Agent Quota

```bash
# No API for Jules quota — track manually
# Record dispatches in a local file
echo "$(date -u +%Y-%m-%dT%H:%MZ) dispatched #ISSUE to jules in REPO" >> .core/dispatch.log
wc -l .core/dispatch.log  # count today's dispatches
```

---

## Budget Tracking & Continuous Flow

The goal is to keep agents working at all times — never idle, never over-budget. Every team member who connects their repo to Jules gets 300 tasks/day. The orchestrator should use the full team allowance.

### Team Budget Pool

Each team member with a Jules-enabled repo contributes to the daily pool:

| Member | Repos Connected | Daily Quota | Notes |
|--------|----------------|-------------|-------|
| @Snider | core, core-php, core-mcp, core-api, ... | 300 | Primary orchestrator |
| @bodane | (to be connected) | 300 | Code owner |
| (additional members) | (additional repos) | 300 | Per-member quota |

**Total pool = members x 300 tasks/day.** With 2 members: 600 tasks/day.

### Budget Tracking

**Preferred:** Use the Jules CLI for accurate, real-time budget info:

```bash
# Get current usage (when Jules CLI is available)
jules usage          # Shows today's task count and remaining quota
jules usage --team   # Shows per-member breakdown
```

**Fallback:** Track dispatches in a structured log:

```bash
# Dispatch log format (append-only)
# TIMESTAMP REPO ISSUE AGENT EPIC
echo "$(date -u +%Y-%m-%dT%H:%MZ) core-mcp #29 jules #EPIC" >> .core/dispatch.log

# Today's usage
TODAY=$(date -u +%Y-%m-%d)
grep "$TODAY" .core/dispatch.log | wc -l

# Remaining budget
USED=$(grep "$TODAY" .core/dispatch.log | wc -l)
POOL=300  # multiply by team size
echo "Used: $USED / $POOL  Remaining: $((POOL - USED))"
```

**Don't guess the budget.** Either query the CLI or count dispatches. Manual estimates drift.

### Continuous Flow Strategy

The orchestrator should maintain a **pipeline of ready work** so agents are never idle. The flow looks like this:

```
BACKLOG          READY            DISPATCHED        IN PROGRESS       DONE
─────────        ─────            ──────────        ───────────       ────
Audit issues  →  Implementation  →  Labelled for  →  Agent working  →  PR merged
(unprocessed)    issues in epics    agent pickup      on PR             child closed
```

**Key metric: READY queue depth.** If the READY queue is empty, agents will idle when current work finishes. The orchestrator should always maintain 2-3x the daily dispatch rate in READY state.

### Dispatch Cadence

```
Morning (start of day):
  1. Check yesterday's results — tick parent checklists for merged PRs
  2. Check remaining budget from yesterday (unused tasks don't roll over)
  3. Unstick any blocked PRs (merge conflicts → resolve-stuck-prs flow after 2+ attempts, unresolved threads)
  4. Dispatch Phase 1 blockers for new epics (if budget allows)
  5. Dispatch next-phase children for epics where phase completed

Midday (check-in):
  6. Check for new merge conflicts from cascade merges
  7. Send "fix the merge conflict" / "fix the code reviews" as needed
  8. Dispatch more children if budget remains and agents are idle

Evening (wind-down):
  9. Review day's throughput: dispatched vs merged vs stuck
  10. Plan tomorrow's dispatch based on remaining backlog
  11. Run Stage 1/2 on new repos to refill READY queue
```

### Auto-Dispatch Rules

When the orchestrator detects a child issue was completed (merged + closed):

1. Tick the parent checklist
2. Check if the completed phase is now done (all children in phase closed)
3. If phase done → dispatch next phase's children
4. If epic done → merge epic branch to dev, close epic, dispatch next epic
5. Log the dispatch in the budget tracker

```bash
# Detect completed children (structural only)
EPIC=299
REPO="dappcore/core"

# Get unchecked children
UNCHECKED=$(gh issue view $EPIC --repo $REPO --json body --jq '
  [.body | split("\n")[] | select(test("^- \\[ \\] #")) |
   capture("^- \\[ \\] #(?<num>[0-9]+)") | .num] | .[]')

# Check which are actually closed
for CHILD in $UNCHECKED; do
  STATE=$(gh issue view $CHILD --repo $REPO --json state --jq '.state')
  if [ "$STATE" = "CLOSED" ]; then
    echo "Child #$CHILD is closed but unchecked — tick parent and dispatch next"
  fi
done
```

### Filling the Pipeline

To ensure agents always have work:

| When | Action |
|------|--------|
| READY queue < 20 issues | Run Stage 1 on next Tier repo |
| All Tier 1 repos have epics | Move to Tier 2 |
| All audits processed | Run new audits (`[Audit]` issue sweep) |
| Epic completes | Merge branch, dispatch next epic in same repo |
| Daily budget < 50% used by midday | Increase dispatch rate |
| Daily budget > 80% used by morning | Throttle, focus on unsticking |

### Multi-Repo Dispatch Balancing

With multiple repos in flight, balance dispatches across repos to avoid bottlenecks:

```
Priority order for dispatch:
1. Critical/High severity children (security fixes first)
2. Repos with most work remaining (maximise throughput)
3. Children with no dependencies (parallelisable)
4. Repos with CI most likely to pass (lower retry cost)
```

**Never dispatch all budget to one repo.** If `core-php` has 50 children, don't dispatch all 50 today. Spread across repos:

```
Example daily plan (300 budget):
  core:       10 tasks (unstick 2 PRs + dispatch 8 new)
  core-php:   40 tasks (Phase 1 security epic)
  core-mcp:   30 tasks (workspace isolation epic)
  core-api:   20 tasks (webhook security epic)
  Remaining: 200 tasks (Tier 2-4 repos or iteration on above)
```

### Team Onboarding

When a new team member connects their repos:

1. Add their repos to the inventory table
2. Update the pool total (+300/day)
3. Run Stage 1-2 on their repos
4. Include their repos in the dispatch balancing

```bash
# Track team members and their quotas
cat <<'EOF' >> .core/team.yaml
members:
  - login: Snider
    quota: 300
    repos: [core, core-php, core-mcp, core-api, core-admin, core-tenant,
            core-developer, core-service-commerce, core-devops, core-template,
            build, ansible-coolify, docker-server-php, docker-server-blockchain]
  - login: bodane
    quota: 300
    repos: []  # to be connected
EOF
```

### `core dev budget` Command

```bash
core dev budget                      # Show today's usage vs pool
core dev budget --plan               # Suggest optimal dispatch plan for today
core dev budget --history            # Daily usage over past week
core dev budget --team               # Show per-member quota and usage
core dev budget --forecast DAYS      # Project when all epics will complete
```

---

## Failure Modes

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Audit has no actionable findings | Stage 1 produces 0 issues | Close audit as "not applicable" |
| Too few issues for epic (< 3) | Stage 2 grouping | Dispatch directly, skip epic |
| Agent can't handle PHP/Go | PR fails CI repeatedly | Re-assign to different model or human |
| Cascade conflicts | Multiple PRs stuck CONFLICTING | Serialise merges, use epic branch |
| Agent quota exhausted | 300 tasks hit | Wait for daily reset, prioritise |
| Repo has no CI | PRs can't pass checks | Skip CI gate, rely on reviews only |
| Epic branch diverges too far from dev | Merge conflicts on epic → dev | Rebase epic branch periodically |

---

## Quick Reference

```
1. AUDIT    → Run audit-issues flow per repo (free, parallelisable)
2. ORGANISE → Run create-epic flow per repo (free, parallelisable)
3. DISPATCH → Add jules label to Phase 1 blockers (costs quota)
4. MONITOR  → Run issue-epic flow per epic (ongoing)
5. COMPLETE → Merge epic branch to dev, close epic
```

---

---

*Companion to: RFC.flow-audit-issues.md, RFC.flow-create-epic.md, RFC.flow-issue-epic.md*
