---
name: flow-resolve-stuck-prs
description: Use when a PR is stuck CONFLICTING after 2+ failed agent attempts. Manual merge conflict resolution using git worktrees.
---

# Flow: Resolve Stuck PRs

Manually resolve merge conflicts when an implementer has failed to fix them after two attempts, and the PR(s) are the last items blocking an epic.

---

## When to Use

All three conditions must be true:

1. **PR is CONFLICTING/DIRTY** after the implementer was asked to fix it (at least twice)
2. **The PR is blocking epic completion** — it's one of the last unchecked children
3. **No other approach worked** — "Can you fix the merge conflict?" was sent and either got no response or the push still left conflicts

## Inputs

- **Repo**: `owner/repo`
- **PR numbers**: The stuck PRs (e.g. `#287, #291`)
- **Target branch**: The branch the PRs target (e.g. `dev`, `epic/101-medium-migration`)

## Process

### Step 1: Confirm Stuck Status

Verify each PR is genuinely stuck — not just slow.

```bash
for PR in 287 291; do
  echo "=== PR #$PR ==="
  gh pr view $PR --repo OWNER/REPO --json mergeable,mergeStateStatus,updatedAt \
    --jq '{mergeable, mergeStateStatus, updatedAt}'
done
```

**Skip if:** `mergeStateStatus` is not `DIRTY` — the PR isn't actually conflicting.

### Step 2: Check Attempt History

Count how many times the implementer was asked and whether it responded.

```bash
# Count "fix the merge conflict" comments
gh pr view $PR --repo OWNER/REPO --json comments \
  --jq '[.comments[] | select(.body | test("merge conflict"; "i"))] | length'

# Check last commit date vs last conflict request
gh pr view $PR --repo OWNER/REPO --json commits \
  --jq '.commits[-1] | {sha: .oid[:8], date: .committedDate}'
```

**Proceed only if:** 2+ conflict fix requests were sent AND either:
- No commit after the last request (implementer didn't respond), OR
- A commit was pushed but `mergeStateStatus` is still `DIRTY` (fix attempt failed)

### Step 3: Clone and Resolve Locally

Task a single agent (or do it manually) to resolve conflicts for ALL stuck PRs in one session.

```bash
# Ensure we have the latest
git fetch origin

# For each stuck PR
for PR in 287 291; do
  BRANCH=$(gh pr view $PR --repo OWNER/REPO --json headRefName --jq '.headRefName')
  TARGET=$(gh pr view $PR --repo OWNER/REPO --json baseRefName --jq '.baseRefName')

  git checkout "$BRANCH"
  git pull origin "$BRANCH"

  # Merge target branch into PR branch
  git merge "origin/$TARGET" --no-edit

  # If conflicts exist, resolve them
  # Agent should: read each conflicted file, choose the correct resolution,
  # stage the resolved files, and commit
  git add -A
  git commit -m "chore: resolve merge conflicts with $TARGET"
  git push origin "$BRANCH"
done
```

**Agent instructions when dispatching:**
> Resolve the merge conflicts on PR #X, #Y, #Z in `owner/repo`.
> For each PR: checkout the PR branch, merge the target branch, resolve all conflicts
> preserving the intent of both sides, commit, and push.
> If a conflict is ambiguous (both sides changed the same logic in incompatible ways),
> prefer the target branch version and note what you dropped in the commit message.

### Step 4: Verify Resolution

After pushing, confirm the PR is no longer conflicting.

```bash
# Wait a few seconds for GitHub to recalculate
sleep 10

for PR in 287 291; do
  STATUS=$(gh pr view $PR --repo OWNER/REPO --json mergeStateStatus --jq '.mergeStateStatus')
  echo "PR #$PR: $STATUS"
done
```

**Expected:** `CLEAN` or `BLOCKED` (waiting for checks, not conflicts).

### Step 5: Handle Failure

If the PR is **still conflicting** after manual resolution:

```bash
# Label for human intervention
gh issue edit $PR --repo OWNER/REPO --add-label "needs-intervention"

# Comment for the gatekeeper
gh pr comment $PR --repo OWNER/REPO \
  --body "Automated conflict resolution failed after 2+ implementer attempts and 1 manual attempt. Needs human review."
```

Create the label if it doesn't exist:
```bash
gh label create "needs-intervention" --repo OWNER/REPO \
  --description "Automated resolution failed — needs human review" \
  --color "B60205" 2>/dev/null
```

The orchestrator should then **skip this PR** and continue with other epic children. Don't block the entire epic on one stuck PR.

---

## Decision Flowchart

```
PR is CONFLICTING
  └─ Was implementer asked to fix? (check comment history)
       ├─ No → Send "Can you fix the merge conflict?" (issue-epic flow)
       └─ Yes, 1 time → Send again, wait for response
            └─ Yes, 2+ times → THIS FLOW
                 └─ Agent resolves locally
                      ├─ Success → PR clean, pipeline continues
                      └─ Failure → Label `needs-intervention`, skip PR
```

## Dispatching as a Subagent

When the orchestrator detects a PR matching the trigger conditions, it can dispatch this flow as a single task:

```
Resolve merge conflicts on PRs #287 and #291 in dappcore/core.

Both PRs target `dev`. The implementer was asked to fix conflicts 2+ times
but they remain DIRTY. Check out each PR branch, merge origin/dev, resolve
all conflicts, commit, and push. If any PR can't be resolved, add the
`needs-intervention` label.
```

**Cost:** 0 Jules tasks (this runs locally or via Claude Code, not via Jules label).

---

## Integration

**Called by:** `issue-epic.md` — when a PR has been CONFLICTING for 2+ fix attempts
**Calls:** Nothing — this is a terminal resolution flow
**Fallback:** `needs-intervention` label → human gatekeeper reviews manually

---

*Created: 2026-02-04*
*Companion to: RFC.flow-issue-epic.md*
