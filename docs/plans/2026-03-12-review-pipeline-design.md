# Review Pipeline Plugin Design

**Date**: 2026-03-12
**Status**: Approved
**Location**: `core/agent/claude/review/`

## Goal

Build a 5-stage automated code review pipeline as a Claude Code plugin command (`/review:pipeline`) that dispatches specialised agent personas sequentially, each building on the previous stage's findings.

## Architecture

Extend the existing `claude/review` plugin (not a new plugin). Add skills that reference agent persona files from `agents/` by path — single source of truth, no duplication.

### File Structure

```
claude/review/
├── .claude-plugin/plugin.json    # Updated — add skills
├── commands/
│   ├── pipeline.md               # NEW — /review:pipeline orchestrator
│   ├── review.md                 # Existing (unchanged)
│   ├── security.md               # Existing (unchanged)
│   └── pr.md                     # Existing (unchanged)
├── skills/
│   ├── security-review.md        # Stage 1: Security Engineer
│   ├── senior-dev-fix.md         # Stage 2: Senior Developer (fix)
│   ├── test-analysis.md          # Stage 3: API Tester
│   ├── architecture-review.md    # Stage 4: Backend Architect
│   └── reality-check.md          # Stage 5: Reality Checker
├── hooks.json                    # Existing (unchanged)
└── scripts/
    └── post-pr-create.sh         # Existing (unchanged)
```

### Agent Personas (source of truth)

| Stage | Agent | Persona File |
|-------|-------|--------------|
| 1 | Security Engineer | `agents/engineering/engineering-security-engineer.md` |
| 2 | Senior Developer | `agents/engineering/engineering-senior-developer.md` |
| 3 | API Tester | `agents/testing/testing-api-tester.md` |
| 4 | Backend Architect | `agents/engineering/engineering-backend-architect.md` |
| 5 | Reality Checker | `agents/testing/testing-reality-checker.md` |

## Pipeline Flow

```
/review:pipeline [range]
  │
  ├─ Stage 1: Security Engineer (read-only review)
  │   → Findings: Critical/High/Medium/Low issues
  │   → If Critical found: flag for Stage 2
  │
  ├─ Stage 2: Senior Developer
  │   → If security Criticals: FIX them, then re-run Stage 1
  │   → If no Criticals: skip to Stage 3
  │
  ├─ Stage 3: API Tester (run tests, analyse coverage)
  │   → Test results + coverage gaps
  │
  ├─ Stage 4: Backend Architect (architecture fit)
  │   → Lifecycle event usage, Actions pattern, tenant isolation
  │
  └─ Stage 5: Reality Checker (final gate)
      → Verdict: READY / NEEDS WORK / FAILED
      → Aggregated report
```

## Command Interface

```
/review:pipeline                     # Staged changes
/review:pipeline HEAD~3..HEAD        # Commit range
/review:pipeline --pr=123            # PR (via gh)
/review:pipeline --stage=security    # Run single stage only
/review:pipeline --skip=fix          # Skip the fix stage (review only)
```

## Skill Design

Each skill file is a markdown document that:

1. Reads the agent persona from the `agents/` directory at dispatch time
2. Constructs a subagent prompt combining: persona + diff context + prior stage findings
3. Dispatches via the Agent tool (general-purpose subagent)
4. Returns structured findings in a consistent format

Skills are lightweight orchestration — the agent personas contain the domain knowledge.

## Output Format

```markdown
# Review Pipeline Report

## Stage 1: Security Review
**Agent**: Security Engineer
[Structured findings with severity, file:line, attack vector, fix]

## Stage 2: Fixes Applied
**Agent**: Senior Developer
[What was fixed, or "Skipped — no Critical issues"]

## Stage 3: Test Analysis
**Agent**: API Tester
[Test pass/fail count, coverage gaps for changed code]

## Stage 4: Architecture Review
**Agent**: Backend Architect
[Lifecycle events, Actions pattern, tenant isolation, namespace mapping]

## Stage 5: Final Verdict
**Agent**: Reality Checker
**Status**: READY / NEEDS WORK / FAILED
**Quality Rating**: C+ / B- / B / B+
[Evidence-based summary with specific file references]
```

## Scope Boundaries (YAGNI)

- No persistent storage of review results
- No automatic PR commenting (add later via hook if needed)
- No parallel agent dispatch (sequential by design — each builds on previous)
- No custom agent selection — the 5-agent team is fixed
- No CodeRabbit integration (separate learning exercise)

## Success Criteria

- `/review:pipeline` runs all 5 stages on a diff and produces an aggregated report
- Each stage uses the tailored agent persona (not generic prompts)
- Security Criticals trigger the fix→re-review loop
- Reality Checker produces an evidence-based verdict with test output
- Individual stages can be run standalone via `--stage=`
- Plugin installs cleanly and doesn't break existing `/review` commands
