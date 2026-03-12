# Review Pipeline Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `/review:pipeline` command to the existing `claude/review` plugin that dispatches 5 specialised agent personas sequentially for automated code review.

**Architecture:** Extend `claude/review/` with a `pipeline.md` command and 5 skill files (one per review stage). Each skill reads its agent persona from `agents/` at dispatch time and constructs a subagent prompt with diff context + prior findings. The command orchestrates the pipeline sequentially.

**Tech Stack:** Claude Code plugin system (commands, skills, hooks.json), Agent tool for subagent dispatch, git/gh CLI for diff collection.

---

### Task 1: Update plugin.json

**Files:**
- Modify: `claude/review/.claude-plugin/plugin.json`

**Step 1: Read the current plugin.json**

```bash
cat /Users/snider/Code/core/agent/claude/review/.claude-plugin/plugin.json
```

Current content:
```json
{
  "name": "review",
  "description": "Code review automation - PR review, security checks, best practices",
  "version": "0.1.0",
  "author": {
    "name": "Host UK"
  }
}
```

**Step 2: Update plugin.json with new version and description**

```json
{
  "name": "review",
  "description": "Code review automation — 5-agent review pipeline, PR review, security checks, architecture validation",
  "version": "0.2.0",
  "author": {
    "name": "Host UK"
  }
}
```

**Step 3: Commit**

```bash
git add claude/review/.claude-plugin/plugin.json
git commit -m "chore(review): bump plugin to v0.2.0 for pipeline feature"
```

---

### Task 2: Create the pipeline command

**Files:**
- Create: `claude/review/commands/pipeline.md`

**Step 1: Create the command file**

This is the main orchestrator. It tells Claude how to run the 5-stage pipeline. The command reads the diff, then dispatches subagents in sequence using the Agent tool.

```markdown
---
name: pipeline
description: Run the 5-agent review pipeline on code changes
args: [commit-range|--pr=N|--stage=NAME|--skip=fix]
---

# Review Pipeline

Run a 5-stage automated code review pipeline using specialised agent personas.

## Usage

```
/review:pipeline                     # Staged changes
/review:pipeline HEAD~3..HEAD        # Commit range
/review:pipeline --pr=123            # PR diff (via gh)
/review:pipeline --stage=security    # Single stage only
/review:pipeline --skip=fix          # Review only, no fixes
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
```

**Step 2: Verify the command is valid markdown with frontmatter**

```bash
head -5 claude/review/commands/pipeline.md
```

Expected: YAML frontmatter with `name: pipeline`, `description`, `args`.

**Step 3: Commit**

```bash
git add claude/review/commands/pipeline.md
git commit -m "feat(review): add /review:pipeline command — 5-agent review orchestrator"
```

---

### Task 3: Create Stage 1 skill — Security Review

**Files:**
- Create: `claude/review/skills/security-review.md`

**Step 1: Create the skill file**

```markdown
---
name: security-review
description: Stage 1 of review pipeline — dispatch Security Engineer agent for threat analysis, injection review, and tenant isolation checks on code changes
---

# Security Review Stage

Dispatch the **Security Engineer** agent to perform a read-only security review of code changes.

## When to Use

This skill is invoked as Stage 1 of the `/review:pipeline` command. It can also be triggered standalone via `/review:pipeline --stage=security`.

## Agent Persona

Read the Security Engineer persona from:
```
agents/engineering/engineering-security-engineer.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Read the diff and list of changed files
3. Dispatch a subagent with the Agent tool using this prompt structure:

```
[Persona content from engineering-security-engineer.md]

## Your Task

Perform a security-focused code review of the following changes. This is a READ-ONLY review — do not modify any files.

### Changed Files
[List of changed files]

### Diff
[Full diff content]

### Focus Areas
- Arbitrary code execution vectors
- Method/class injection from DB or config values
- Tenant isolation (BelongsToWorkspace on all tenant-scoped models)
- Input validation in Action handle() methods
- Namespace safety (allowlists where class names come from external sources)
- Error handling (no silent swallowing, no stack trace leakage)
- Secrets in code (API keys, credentials, .env values)

### Output Format

Produce findings in this exact format:

## Security Review Findings

### CRITICAL
- **file:line** — [Title]: [Description]. **Attack vector**: [How]. **Fix**: [What to change]

### HIGH
- **file:line** — [Title]: [Description]. **Fix**: [What to change]

### MEDIUM
- **file:line** — [Title]: [Description]. **Fix**: [What to change]

### LOW
- **file:line** — [Title]: [Description]

### Positive Controls
[Things done well — allowlists, guards, scoping]

**Summary**: X critical, Y high, Z medium, W low
```

4. Return the subagent's findings as the stage output
```

**Step 2: Commit**

```bash
git add claude/review/skills/security-review.md
git commit -m "feat(review): add security-review skill — Stage 1 of pipeline"
```

---

### Task 4: Create Stage 2 skill — Senior Dev Fix

**Files:**
- Create: `claude/review/skills/senior-dev-fix.md`

**Step 1: Create the skill file**

```markdown
---
name: senior-dev-fix
description: Stage 2 of review pipeline — dispatch Senior Developer agent to fix Critical security findings from Stage 1
---

# Senior Developer Fix Stage

Dispatch the **Senior Developer** agent to fix Critical security findings from Stage 1.

## When to Use

Invoked as Stage 2 of `/review:pipeline` ONLY when Stage 1 found Critical issues. Skipped when `--skip=fix` is passed or when there are no Critical findings.

## Agent Persona

Read the Senior Developer persona from:
```
agents/engineering/engineering-senior-developer.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Construct a prompt with the Critical findings from Stage 1
3. Dispatch a subagent with the Agent tool:

```
[Persona content from engineering-senior-developer.md]

## Your Task

Fix the following CRITICAL security issues found during review. Apply the fixes directly to the source files.

### Critical Findings to Fix
[Stage 1 Critical findings — exact file:line, description, recommended fix]

### Rules
- Fix ONLY the Critical issues listed above — do not refactor surrounding code
- Follow existing code style (spacing, braces, naming)
- declare(strict_types=1) in every PHP file
- UK English in all comments and strings
- Run tests after fixing to verify nothing breaks:
  [appropriate test command for the package]

### Output Format

## Fixes Applied

### Fix 1: [Title]
**File**: `path/to/file.php:line`
**Issue**: [What was wrong]
**Change**: [What was changed]

### Fix 2: ...

**Tests**: [PASS/FAIL — test output summary]
```

4. After the subagent completes, re-dispatch Stage 1 (security-review) to verify the fixes resolved the Critical issues
5. If Criticals persist after one fix cycle, report them in the final output rather than looping indefinitely
```

**Step 2: Commit**

```bash
git add claude/review/skills/senior-dev-fix.md
git commit -m "feat(review): add senior-dev-fix skill — Stage 2 of pipeline"
```

---

### Task 5: Create Stage 3 skill — Test Analysis

**Files:**
- Create: `claude/review/skills/test-analysis.md`

**Step 1: Create the skill file**

```markdown
---
name: test-analysis
description: Stage 3 of review pipeline — dispatch API Tester agent to run tests and analyse coverage of changed code
---

# Test Analysis Stage

Dispatch the **API Tester** agent to run tests and identify coverage gaps for the changed code.

## When to Use

Invoked as Stage 3 of `/review:pipeline`. Can be run standalone via `/review:pipeline --stage=test`.

## Agent Persona

Read the API Tester persona from:
```
agents/testing/testing-api-tester.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Determine the test command based on the package:
   - PHP packages: `composer test` or `vendor/bin/phpunit [specific test files]`
   - Go packages: `core go test` or `go test ./...`
3. Dispatch a subagent:

```
[Persona content from testing-api-tester.md]

## Your Task

Run the test suite and analyse coverage for the following code changes. Do NOT write new tests — this is analysis only.

### Changed Files
[List of changed files from the diff]

### Instructions

1. **Run existing tests**
   [Test command for this package]
   Report: total tests, passed, failed, assertion count

2. **Analyse coverage of changes**
   For each changed file, find the corresponding test file(s). Read both the source change and the test.
   Report whether the specific change is exercised by existing tests.

3. **Identify coverage gaps**
   List changes that have NO test coverage, with specific descriptions of what's untested.

### Output Format

## Test Analysis

### Test Results
**Command**: `[exact command run]`
**Result**: X tests, Y assertions, Z failures

### Coverage of Changes

| Changed File | Test File | Change Covered? | Gap |
|-------------|-----------|-----------------|-----|
| `path:lines` | `test/path` | YES/NO | [What's untested] |

### Coverage Gaps
1. **file:line** — [What's changed but untested]
2. ...

### Recommendations
[Specific tests that should be written — Pest syntax for PHP, _Good/_Bad/_Ugly for Go]

**Summary**: X/Y changes covered, Z gaps identified
```

4. Return the subagent's analysis as the stage output
```

**Step 2: Commit**

```bash
git add claude/review/skills/test-analysis.md
git commit -m "feat(review): add test-analysis skill — Stage 3 of pipeline"
```

---

### Task 6: Create Stage 4 skill — Architecture Review

**Files:**
- Create: `claude/review/skills/architecture-review.md`

**Step 1: Create the skill file**

```markdown
---
name: architecture-review
description: Stage 4 of review pipeline — dispatch Backend Architect agent to check lifecycle events, Actions pattern, tenant isolation, and namespace mapping
---

# Architecture Review Stage

Dispatch the **Backend Architect** agent to review code changes for architectural correctness.

## When to Use

Invoked as Stage 4 of `/review:pipeline`. Can be run standalone via `/review:pipeline --stage=architecture`.

## Agent Persona

Read the Backend Architect persona from:
```
agents/engineering/engineering-backend-architect.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Dispatch a subagent:

```
[Persona content from engineering-backend-architect.md]

## Your Task

Review the following code changes for architectural correctness. This is a READ-ONLY review.

### Changed Files
[List of changed files]

### Diff
[Full diff content]

### Check These Patterns

1. **Lifecycle Events**: Are modules using `$listens` arrays in Boot.php? Are routes registered via event callbacks (`$event->routes()`), not direct `Route::get()` calls?

2. **Actions Pattern**: Is business logic in Action classes with `use Action` trait? Or is it leaking into controllers/Livewire components?

3. **Tenant Isolation**: Do new/modified models that hold tenant data use `BelongsToWorkspace`? Are migrations adding `workspace_id` with foreign key and cascade delete?

4. **Namespace Mapping**: Do files follow `src/Core/` → `Core\`, `src/Mod/` → `Core\Mod\`, `app/Mod/` → `Mod\`?

5. **Go Services** (if applicable): Are services registered via factory functions? Using `ServiceRuntime[T]`? Implementing `Startable`/`Stoppable`?

6. **Dependency Direction**: Do changes respect the dependency graph? Products depend on core-php and core-tenant, never on each other.

### Output Format

## Architecture Review

### Lifecycle Events
[Findings or "Correct — events used properly"]

### Actions Pattern
[Findings or "Correct — logic in Actions"]

### Tenant Isolation
[Findings or "Correct — BelongsToWorkspace on all tenant models"]

### Namespace Mapping
[Findings or "Correct"]

### Dependency Direction
[Findings or "Correct"]

### Issues
- **VIOLATION**: file:line — [Description]
- **WARNING**: file:line — [Description]
- **SUGGESTION**: file:line — [Description]

**Summary**: X violations, Y warnings, Z suggestions
```

3. Return the subagent's review as the stage output
```

**Step 2: Commit**

```bash
git add claude/review/skills/architecture-review.md
git commit -m "feat(review): add architecture-review skill — Stage 4 of pipeline"
```

---

### Task 7: Create Stage 5 skill — Reality Check

**Files:**
- Create: `claude/review/skills/reality-check.md`

**Step 1: Create the skill file**

```markdown
---
name: reality-check
description: Stage 5 of review pipeline — dispatch Reality Checker agent as final gate with evidence-based verdict
---

# Reality Check Stage (Final Gate)

Dispatch the **Reality Checker** agent as the final review gate. Defaults to NEEDS WORK.

## When to Use

Invoked as Stage 5 (final stage) of `/review:pipeline`. Can be run standalone via `/review:pipeline --stage=reality`.

## Agent Persona

Read the Reality Checker persona from:
```
agents/testing/testing-reality-checker.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Gather ALL prior stage findings into a single context block
3. Dispatch a subagent:

```
[Persona content from testing-reality-checker.md]

## Your Task

You are the FINAL GATE. Review all prior stage findings and produce an evidence-based verdict. Default to NEEDS WORK.

### Prior Stage Findings

#### Stage 1: Security Review
[Stage 1 output]

#### Stage 2: Fixes Applied
[Stage 2 output, or "Skipped"]

#### Stage 3: Test Analysis
[Stage 3 output]

#### Stage 4: Architecture Review
[Stage 4 output]

### Changed Files
[List of changed files]

### Your Assessment

1. **Cross-reference all findings** — do security fixes have tests? Do architecture violations have security implications?
2. **Verify evidence** — are test results real (actual command output) or claimed?
3. **Check for gaps** — what did previous stages miss?
4. **Apply your FAIL triggers** — fantasy assessments, missing evidence, architecture violations

### Output Format

## Final Verdict

**Status**: READY / NEEDS WORK / FAILED
**Quality Rating**: C+ / B- / B / B+

### Evidence Summary
| Check | Status | Evidence |
|-------|--------|----------|
| Tests pass | YES/NO | [Command + output] |
| Lint clean | YES/NO | [Command + output] |
| Security issues resolved | YES/NO | [Remaining count] |
| Architecture correct | YES/NO | [Violation count] |
| Tenant isolation verified | YES/NO | [Specific check] |
| UK English | YES/NO | [Violations found] |
| Test coverage of changes | X/Y | [Gap count] |

### Outstanding Issues
1. **[CRITICAL/IMPORTANT/MINOR]**: file:line — [Issue]
2. ...

### Required Before Merge
1. [Specific action with file path]
2. ...

### What's Done Well
[Positive findings from all stages]

---
**Reviewer**: Reality Checker
**Date**: [Date]
**Re-review required**: YES/NO
```

4. Return the subagent's verdict as the final pipeline output
```

**Step 2: Commit**

```bash
git add claude/review/skills/reality-check.md
git commit -m "feat(review): add reality-check skill — Stage 5 final gate"
```

---

### Task 8: Create skills directory and verify plugin structure

**Files:**
- Verify: `claude/review/skills/` contains all 5 skill files
- Verify: `claude/review/commands/pipeline.md` exists
- Verify: `claude/review/.claude-plugin/plugin.json` is updated

**Step 1: Verify the complete file structure**

```bash
find claude/review/ -type f | sort
```

Expected output:
```
claude/review/.claude-plugin/plugin.json
claude/review/commands/pipeline.md
claude/review/commands/pr.md
claude/review/commands/review.md
claude/review/commands/security.md
claude/review/hooks.json
claude/review/scripts/post-pr-create.sh
claude/review/skills/architecture-review.md
claude/review/skills/reality-check.md
claude/review/skills/security-review.md
claude/review/skills/senior-dev-fix.md
claude/review/skills/test-analysis.md
```

**Step 2: Verify agent persona files exist**

```bash
ls -la agents/engineering/engineering-security-engineer.md \
       agents/engineering/engineering-senior-developer.md \
       agents/testing/testing-api-tester.md \
       agents/engineering/engineering-backend-architect.md \
       agents/testing/testing-reality-checker.md
```

Expected: All 5 files exist.

**Step 3: Final commit**

```bash
git add -A claude/review/
git commit -m "feat(review): complete review pipeline plugin — 5-agent automated code review"
```

---

### Task 9: Smoke test the plugin

**Step 1: Test that the pipeline command is recognised**

From the `core/agent` directory, verify the plugin structure is valid by checking the command is loadable:

```bash
# Check frontmatter is valid
head -5 claude/review/commands/pipeline.md
```

Expected: Valid YAML frontmatter with `name: pipeline`.

**Step 2: Test that skill files have valid frontmatter**

```bash
for f in claude/review/skills/*.md; do echo "=== $f ==="; head -4 "$f"; echo; done
```

Expected: Each skill has `name:` and `description:` in frontmatter.

**Step 3: Test the pipeline manually**

Open a Claude Code session in a repo with recent changes and run:

```
/review:pipeline HEAD~1..HEAD
```

Verify:
- Stage 1 dispatches and returns security findings
- Stage 2 is skipped (if no Criticals) or runs fixes
- Stage 3 runs tests and reports coverage
- Stage 4 checks architecture patterns
- Stage 5 produces a verdict

**Step 4: Test single-stage mode**

```
/review:pipeline --stage=security HEAD~1..HEAD
```

Verify: Only Stage 1 runs.
