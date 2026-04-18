# Agentic Pipeline v2 — Autonomous Dispatch→Verify→Merge

> The full autonomous pipeline: issue → dispatch → implement → verify → PR → merge.
> CodeRabbit findings = 0 is the KPI.

---

## Pipeline Flow

```
Issue created (Forge/GitHub)
  → core-agent picks up event
  → Selects flow YAML based on event type + repo
  → Prepares sandboxed workspace (CODEX.md, .core/reference/)
  → Dispatches agent (codex/gemini/claude)
  → Agent implements in workspace
  → QA flow runs (build, test, vet, lint)
  → If QA passes → create PR to dev
  → CodeRabbit reviews PR
  → If findings = 0 → auto-merge
  → If findings > 0 → dispatch fix agent → repeat
  → PR merged → training data captured
  → Issue closed
```

## Key Design Decisions

### Sandboxing
Agents MUST be sandboxed to their assigned repo. Unsandboxed writes caused the CLI mess
(agent wrote files to wrong repo). Workspace isolation is non-negotiable.

### CodeRabbit KPI
CodeRabbit findings = 0 is the target. Every finding means:
- Template didn't prevent it → fix the template
- Model didn't catch it → add to training data
- Convention wasn't documented → add to RFC

Zero findings = complete convention coverage.

### Checkin API
Agents check in with status via api.lthn.sh. Current blocker: Forge webhooks
need to fire to lthn.sh so the orchestrator knows when to start the pipeline.

### Security Model (from Charon flows)
Orchestrator uses STRUCTURAL signals only (labels, PR state, review counts).
Never parses comment CONTENT — immune to prompt injection via issue comments.

## Agent Pool Configuration

See `code/core/go/agent/RFC.md` §Dispatch & Pool Routing for the full `agent.yaml` schema (concurrency, rates, model variants, agent identities).

Concurrency enforced by runner service (core/agent). Slot reservation prevents
TOCTOU race between parallel dispatches.

## go-process Improvements Needed

- `Timeout` — kill after N minutes (currently agents can run forever)
- `GracePeriod` — SIGTERM before SIGKILL
- `KillGroup` — kill process group, not just PID (prevents orphaned subprocesses)

## Metrics

- 25 repos auto-merged in recent sweep
- 74 findings on core/agent alone (70+ fixed)
- Zero-finding rate improving as templates capture conventions

## `core pipeline` Command Tree (Go Implementation)

```
core pipeline
├── audit <repo>              # Stage 1: audit issues → implementation issues
├── epic
│   ├── create <repo>         # Stage 2: group issues into epics
│   ├── run <epic-number>     # Stage 3: dispatch + monitor an epic
│   ├── status [epic-number]  # Show epic progress
│   └── sync <epic-number>    # Tick parent checklist from closed children
├── monitor [repo]            # Watch all open PRs, auto-intervene
├── fix
│   ├── reviews <pr-number>   # "Can you fix the code reviews?"
│   ├── conflicts <pr-number> # "Can you fix the merge conflict?"
│   ├── format <pr-number>    # gofmt, commit, push (no AI)
│   └── threads <pr-number>   # Resolve all threads after fix
├── onboard <repo>            # Full: audit → epic → dispatch
├── budget                    # Daily usage vs pool
│   ├── plan                  # Optimal dispatch for today
│   └── log                   # Append dispatch event
└── training
    ├── capture <pr-number>   # Journal entry for merged PR
    ├── stats                 # Summary across journals
    └── export                # Clean export for LEM training
```

## MetaReader — Structural Signals Only

The core abstraction. Every pipeline decision comes through this interface. **NEVER reads comment bodies, commit messages, PR descriptions, or review content.**

```go
type MetaReader interface {
    GetPRMeta(repo string, pr int) (*PRMeta, error)
    GetEpicMeta(repo string, issue int) (*EpicMeta, error)
    GetIssueState(repo string, issue int) (string, error)
    GetCommentReactions(repo string, commentID int64) ([]ReactionMeta, error)
}
```

### PRMeta
```go
type PRMeta struct {
    Number          int
    State           string    // OPEN, MERGED, CLOSED
    Mergeable       string    // MERGEABLE, CONFLICTING, UNKNOWN
    HeadSHA         string
    HeadDate        time.Time
    AutoMerge       bool
    BaseBranch      string
    HeadBranch      string
    Checks          []CheckMeta
    ThreadsTotal    int
    ThreadsResolved int
    HasEyesReaction bool      // 👀 = agent acknowledged
}

type CheckMeta struct {
    Name       string // "qa", "build", "org-gate"
    Conclusion string // "SUCCESS", "FAILURE", ""
    Status     string // "COMPLETED", "QUEUED", "IN_PROGRESS"
}
```

### EpicMeta
```go
type EpicMeta struct {
    Number   int
    State    string
    Children []ChildMeta
}

type ChildMeta struct {
    Number  int
    Checked bool   // [x] vs [ ]
    State   string // OPEN, CLOSED
    PRs     []int
}
```

### Security: What's Explicitly Excluded

The MetaReader has NO methods for:
- `GetCommentBodies` — injection vector
- `GetCommitMessages` — can contain crafted instructions
- `GetPRDescription` — attacker-controlled in fork PRs
- `GetReviewThreadContent` — untrusted input

Implementation uses `gh api` with `--jq` filters that strip content at the query level. Content never enters the Go process.

## Three-Stage Pipeline

```
STAGE 1: AUDIT (flow: audit-issues)
  Input:  Repo with [Audit] issues
  Output: Implementation issues (1 per finding)
  → Classify findings (severity, type, scope, complexity)
  → Detect patterns (3+ similar → framework issue)
  → Close audit issues, link to children

STAGE 2: ORGANISE (flow: create-epic)
  Input:  Implementation issues
  Output: Epic parent with children, branch, phase ordering
  → Group by theme (security, quality, testing)
  → Order into phases (blockers → parallel → cleanup)
  → Create epic branch off dev

STAGE 3: EXECUTE (flow: issue-epic)
  Input:  Epic with children, branch
  Output: Merged PRs, closed issues, training data
  → Dispatch Phase 1 to agents
  → Monitor: CI, reviews, conflicts, merges
  → Intervene: fix reviews / fix conflicts
  → Phase complete → dispatch next phase
  → Epic complete → merge epic branch to dev
```

## Gotchas (Battle-Tested)

| Gotcha | Fix |
|--------|-----|
| Jules creates PRs as user, not bot | Match by branch/issue linkage, not author |
| `git push origin dev` ambiguous (tag+branch) | Use `HEAD:refs/heads/dev` |
| Base branch gofmt breaks ALL PRs | Fix base first, not the PRs |
| Auto-merge needs explicit permissions in caller | Add `permissions: contents: write, pull-requests: write` |
| `--squash` conflicts with merge queue | Use `--auto` alone — queue controls strategy |

## Knowledge Accumulation (Discussions Strategy)

Non-actionable findings (nitpicks, patterns, style preferences) get posted to a queryable knowledge base (Forge/OpenBrain). When patterns emerge, humans create issues.

```
Build → Agents review → Actionable → Fix immediately
                      → Non-actionable → Post to knowledge base
                                         → Patterns emerge
                                         → Human creates Issue
                                         → Agent picks up via pipeline
```

### Discussion Categories

| Channel | Category | Purpose |
|---------|----------|---------|
| 🚧 dev | PR build findings | Per-PR QA findings |
| 🛩️ alpha | Canary findings | Early testing |
| 🛸 beta | Integration findings | Integration testing |
| 🚀 stable | Release audit | Production audit |

### Naming: `{tool}:v{VERSION}`

`qa:v0.0.4.pr.264`, `lint:v0.0.4-alpha.42`, `audit:v0.0.4`

Tool prefixes: `qa:`, `lint:`, `static:`, `docker:`, `e2e:`, `perf:`, `security:`, `audit:`

### Pattern Detection

Query discussions to surface patterns across builds:
```bash
# 47 aria-label mentions across dev discussions → time for a11y audit issue
gh api graphql ... | grep -c "aria-label"
```

### CLI Integration

```bash
core go qa --post-findings    # Post lint findings to discussion
core php qa --post-findings   # Same for PHP
core qa                       # Aggregated summary
```

### Connection to Training

Discussion patterns → Issue → Agent implements → PR merged → findings captured as LEM training data. The feedback loop that makes agents better at conventions over time.

---

## Related RFCs

- `code/core/agent/flow/` — Flow YAML system
- `code/core/agent/RFC.md` — Agent dispatch system
- `project/lthn/lem/RFC-TRAINING-PIPELINE.md` — Findings → training data
