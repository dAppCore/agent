---
name: flow-create-epic
description: Use when grouping 3+ ungrouped issues into epics with branches. Creates parent epic issues with checklists and corresponding epic branches.
---

# Flow: Create Epic

Turn a group of related issues into an epic with child issues, an epic branch, and a parent checklist — ready for the issue-epic flow to execute.

---

## When to Use

- A repo has multiple open issues that share a theme (audit, migration, feature area)
- You want to parallelise work across agents on related tasks
- You need to track progress of a multi-issue effort

## Inputs

- **Repo**: `owner/repo`
- **Theme**: What groups these issues (e.g. "security audit", "io migration", "help system")
- **Candidate issues**: Found by label, keyword, or manual selection

## Process

### Step 1: Find Candidate Issues

Search for issues that belong together. Use structural signals only — labels, title patterns, repo.

```bash
# By label
gh search issues --repo OWNER/REPO --state open --label LABEL --json number,title

# By title pattern
gh search issues --repo OWNER/REPO --state open --json number,title \
  --jq '.[] | select(.title | test("PATTERN"))'

# All open issues in a repo (for small repos)
gh issue list --repo OWNER/REPO --state open --json number,title,labels
```

Group candidates by dependency order if possible:
- **Blockers first**: Interface changes, shared types, core abstractions
- **Parallel middle**: Independent migrations, per-package work
- **Cleanup last**: Deprecation removal, docs, final validation

### Step 2: Check for Existing Epics

Before creating a new epic, check if one already exists.

```bash
# Search for issues with child checklists in the repo
gh search issues --repo OWNER/REPO --state open --json number,title,body \
  --jq '.[] | select(.body | test("- \\[[ x]\\] #\\d+")) | {number, title}'
```

If an epic exists for this theme, update it instead of creating a new one.

### Step 3: Order the Children

Arrange child issues into phases based on dependencies:

```
Phase 1: Blockers (must complete before Phase 2)
  - Interface definitions, shared types, core changes

Phase 2: Parallel work (independent, can run simultaneously)
  - Per-package migrations, per-file changes

Phase 3: Cleanup (depends on Phase 2 completion)
  - Remove deprecated code, update docs, final validation
```

Within each phase, issues are independent and can be dispatched to agents in parallel.

### Step 4: Create the Epic Issue

Create a parent issue with the child checklist.

```bash
gh issue create --repo OWNER/REPO \
  --title "EPIC_TITLE" \
  --label "agentic,complexity:large" \
  --body "$(cat <<'EOF'
## Overview

DESCRIPTION OF THE EPIC GOAL.

## Child Issues

### Phase 1: PHASE_NAME (blocking)
- [ ] #NUM - TITLE
- [ ] #NUM - TITLE

### Phase 2: PHASE_NAME (parallelisable)
- [ ] #NUM - TITLE
- [ ] #NUM - TITLE

### Phase 3: PHASE_NAME (cleanup)
- [ ] #NUM - TITLE

## Acceptance Criteria

- [ ] CRITERION_1
- [ ] CRITERION_2
EOF
)"
```

**Checklist format matters.** The issue-epic flow detects children via `- [ ] #NUM` and `- [x] #NUM` patterns. Use exactly this format.

### Step 5: Link Children to Parent

Add a `Parent: #EPIC_NUMBER` line to each child issue body, or comment it.

```bash
for CHILD in NUM1 NUM2 NUM3; do
  gh issue comment $CHILD --repo OWNER/REPO --body "Parent: #EPIC_NUMBER"
done
```

### Step 6: Create the Epic Branch

Create a branch off dev (or the repo's default branch) for the epic.

```bash
# Get default branch SHA
SHA=$(gh api repos/OWNER/REPO/git/refs/heads/dev --jq '.object.sha')

# Create epic branch
gh api repos/OWNER/REPO/git/refs -X POST \
  -f ref="refs/heads/epic/EPIC_NUMBER-SLUG" \
  -f sha="$SHA"
```

**Naming:** `epic/<issue-number>-<short-slug>` (e.g. `epic/118-mcp-daemon`)

### Step 7: Dispatch Blockers

Add the agent label to the first unchecked child in each phase (the blocker). Add a target branch comment.

```bash
# Label the blocker
gh issue edit CHILD_NUM --repo OWNER/REPO --add-label jules

# Comment the target branch
gh issue comment CHILD_NUM --repo OWNER/REPO \
  --body "Target branch: \`epic/EPIC_NUMBER-SLUG\` (epic #EPIC_NUMBER)"
```

**IMPORTANT:** Adding the agent label (e.g. `jules`) immediately dispatches work. Only label when ready. Each label costs a daily task from the agent's quota.

---

## Creating Epics from Audit Issues

Many repos have standalone audit issues (e.g. `[Audit] Security`, `[Audit] Performance`). These can be grouped into a single audit epic per repo.

### Pattern: Audit Epic

```bash
# Find all audit issues in a repo
gh issue list --repo OWNER/REPO --state open --label jules \
  --json number,title --jq '.[] | select(.title | test("\\[Audit\\]|audit:"))'
```

Group by category and create an epic:

```markdown
## Child Issues

### Security
- [ ] #36 - OWASP Top 10 security review
- [ ] #37 - Input validation and sanitization
- [ ] #38 - Authentication and authorization flows

### Quality
- [ ] #41 - Code complexity and maintainability
- [ ] #42 - Test coverage and quality
- [ ] #43 - Performance bottlenecks

### Ops
- [ ] #44 - API design and consistency
- [ ] #45 - Documentation completeness
```

Audit issues are typically independent (no phase ordering needed) — all can be dispatched in parallel.

---

## Creating Epics from Feature Issues

Feature repos (e.g. `core-claude`) may have many related feature issues that form a product epic.

### Pattern: Feature Epic

Group by dependency:
1. **Foundation**: Core abstractions the features depend on
2. **Features**: Independent feature implementations
3. **Integration**: Cross-feature integration, docs, onboarding

---

## Checklist

Before dispatching an epic:

- [ ] Candidate issues identified and ordered
- [ ] No existing epic covers this theme
- [ ] Epic issue created with `- [ ] #NUM` checklist
- [ ] Children linked back to parent (`Parent: #NUM`)
- [ ] Epic branch created (`epic/<number>-<slug>`)
- [ ] Blocker issues (Phase 1 first children) labelled for dispatch
- [ ] Target branch commented on labelled issues
- [ ] Agent quota checked (don't over-dispatch)

---

*Companion to: RFC.flow-issue-epic.md*
