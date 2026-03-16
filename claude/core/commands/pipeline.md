---
name: pipeline
description: Run the 5-agent review pipeline on code changes
args: [commit-range|--pr=N|--stage=NAME|--skip=fix]
---

# Review Pipeline

Run a 5-stage automated code review pipeline using specialised agent personas.

## Usage

```
/core:pipeline                       # Staged changes
/core:pipeline HEAD~3..HEAD          # Commit range
/core:pipeline --pr=123              # PR diff (via gh)
/core:pipeline --stage=security      # Single stage only
/core:pipeline --skip=fix            # Review only, no fixes
```

## Pipeline Stages

| Stage | Agent | Role | Modifies Code? |
|-------|-------|------|----------------|
| 1 | Security Engineer | Threat analysis, injection, tenant isolation | No |
| 2 | Senior Developer | Fix Critical security findings | Yes |
| 3 | API Tester | Run tests, analyse coverage gaps | No |
| 4 | Backend Architect | Architecture fit, lifecycle events, Actions pattern | No |
| 5 | Reality Checker | Final gate — evidence-based verdict | No |

## Process

### Step 1: Gather the diff

Determine what code to review based on arguments:

```bash
# Staged changes (default)
git diff --cached

# Commit range
git diff HEAD~3..HEAD

# PR
gh pr diff 123

# Also get the list of changed files
git diff --name-only HEAD~3..HEAD
```

Store the diff and file list — every stage needs them.

### Step 2: Identify the package

Determine which package the changes belong to by checking file paths. This tells agents where to run tests:

```bash
# If files are in src/Core/ or app/Core/ → core/php package
# If files are in a core-{name}/ directory → that package
# Check for composer.json or go.mod to confirm
```

### Step 3: Run the pipeline

Dispatch each stage as a subagent using the Agent tool. Each stage receives:
- The diff context
- The list of changed files
- Findings from all previous stages
- Its agent persona (read from agents/ directory)

**Stage 1 — Security Review:**
- Read persona: `agents/engineering/engineering-security-engineer.md`
- Dispatch subagent with persona + diff
- Task: Read-only security review. Find threats, injection, tenant isolation gaps
- Output: Structured findings with severity ratings
- If any CRITICAL findings → flag for Stage 2

**Stage 2 — Fix (conditional):**
- Read persona: `agents/engineering/engineering-senior-developer.md`
- SKIP if `--skip=fix` was passed
- SKIP if Stage 1 found no CRITICAL issues
- Dispatch subagent with persona + Stage 1 Critical findings
- Task: Fix the Critical security issues
- After fixing: re-dispatch Stage 1 to verify fixes
- Output: List of files modified and what was fixed

**Stage 3 — Test Analysis:**
- Read persona: `agents/testing/testing-api-tester.md`
- Dispatch subagent with persona + diff + changed files
- Task: Run tests (`composer test` or `core go test`), analyse which changes have test coverage
- Output: Test results (pass/fail/count) + coverage gaps

**Stage 4 — Architecture Review:**
- Read persona: `agents/engineering/engineering-backend-architect.md`
- Dispatch subagent with persona + diff + changed files
- Task: Check lifecycle event usage, Actions pattern adherence, tenant isolation, namespace mapping
- Output: Architecture assessment with specific findings

**Stage 5 — Reality Check (final gate):**
- Read persona: `agents/testing/testing-reality-checker.md`
- Dispatch subagent with persona + ALL prior stage findings + test output
- Task: Evidence-based final verdict. Default to NEEDS WORK.
- Output: Verdict (READY / NEEDS WORK / FAILED) + quality rating + required fixes

### Step 4: Aggregate report

Combine all stage outputs into the final report:

```markdown
# Review Pipeline Report

## Stage 1: Security Review
**Agent**: Security Engineer
[Stage 1 findings]

## Stage 2: Fixes Applied
**Agent**: Senior Developer
[Stage 2 output, or "Skipped — no Critical issues"]

## Stage 3: Test Analysis
**Agent**: API Tester
[Stage 3 test results + coverage gaps]

## Stage 4: Architecture Review
**Agent**: Backend Architect
[Stage 4 architecture assessment]

## Stage 5: Final Verdict
**Agent**: Reality Checker
**Status**: [READY / NEEDS WORK / FAILED]
**Quality Rating**: [C+ / B- / B / B+]
[Evidence-based summary]

---
Pipeline completed: [timestamp]
Stages run: [1-5]
```

## Single Stage Mode

When `--stage=NAME` is passed, run only that stage:

| Name | Stage |
|------|-------|
| `security` | Stage 1: Security Engineer |
| `fix` | Stage 2: Senior Developer |
| `test` | Stage 3: API Tester |
| `architecture` | Stage 4: Backend Architect |
| `reality` | Stage 5: Reality Checker |

For single-stage mode, still gather the diff but skip prior/subsequent stages.

## Agent Persona Paths

All personas live in the `agents/` directory relative to the plugin root's parent:

```
${CLAUDE_PLUGIN_ROOT}/../../agents/engineering/engineering-security-engineer.md
${CLAUDE_PLUGIN_ROOT}/../../agents/engineering/engineering-senior-developer.md
${CLAUDE_PLUGIN_ROOT}/../../agents/testing/testing-api-tester.md
${CLAUDE_PLUGIN_ROOT}/../../agents/engineering/engineering-backend-architect.md
${CLAUDE_PLUGIN_ROOT}/../../agents/testing/testing-reality-checker.md
```

Read each persona file before dispatching that stage's subagent.
