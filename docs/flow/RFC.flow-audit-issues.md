---
name: flow-audit-issues
description: Use when processing [Audit] issues to create implementation issues. Converts security/quality audit findings into actionable child issues for agent dispatch.
---

# Flow: Audit Issues

Turn audit findings into actionable implementation issues. Every finding matters — even nitpicks hint at framework-level patterns.

---

## Philosophy

> Every audit finding is valid. No dismissing, no "won't fix".

An agent found it for a reason. Even if the individual fix seems trivial, it may:
- Reveal a **pattern** across the codebase (10 similar issues = framework change)
- Become **training data** (good responses teach future models; bad responses go in the "bad responses" set — both have value)
- Prevent a **real vulnerability** that looks minor in isolation

Label accurately. Let the data accumulate. Patterns emerge from volume.

## When to Use

- An audit issue exists (e.g. `[Audit] OWASP Top 10`, `audit: Error handling`)
- The audit contains findings that need implementation work
- You need to convert audit prose into discrete, assignable issues

## Inputs

- **Audit issue**: The `[Audit]` or `audit:` issue with findings
- **Repo**: Where the audit was performed

## Process

### Step 1: Read the Audit

Read the audit issue body. It contains findings grouped by category/severity.

```bash
gh issue view AUDIT_NUMBER --repo OWNER/REPO
```

### Step 2: Classify Each Finding

For each finding, determine:

| Field | Values | Purpose |
|-------|--------|---------|
| **Severity** | `critical`, `high`, `medium`, `low` | Priority ordering |
| **Type** | `security`, `quality`, `performance`, `testing`, `docs` | Categorisation |
| **Scope** | `single-file`, `package`, `framework` | Size of fix |
| **Complexity** | `small`, `medium`, `large` | Agent difficulty |

### Scope Matters Most

| Scope | What it means | Example |
|-------|---------------|---------|
| `single-file` | Fix in one file, no API changes | Add input validation to one handler |
| `package` | Fix across a package, internal API may change | Add error wrapping throughout pkg/mcp |
| `framework` | Requires core abstraction change, affects many packages | Add centralised input sanitisation middleware |

**Nitpicky single-file issues that repeat across packages → framework scope.** The individual finding is small but the pattern is big. Create both:
1. Individual issues for each occurrence (labelled `single-file`)
2. A framework issue that solves all of them at once (labelled `framework`)

The framework issue becomes a blocker in an epic. The individual issues become children that validate the framework fix works.

### Step 3: Create Implementation Issues

One issue per finding. Use consistent title format.

```bash
gh issue create --repo OWNER/REPO \
  --title "TYPE(PACKAGE): DESCRIPTION" \
  --label "SEVERITY,TYPE,complexity:SIZE,SCOPE" \
  --body "$(cat <<'EOF'
Parent audit: #AUDIT_NUMBER

## Finding

WHAT_THE_AUDIT_FOUND

## Location

- `path/to/file.go:LINE`

## Fix

WHAT_NEEDS_TO_CHANGE

## Acceptance Criteria

- [ ] CRITERION
EOF
)"
```

### Title Format

```
type(scope): short description

fix(mcp): validate tool handler input parameters
security(api): add rate limiting to webhook endpoint
quality(cli): replace Fatal with structured Error
test(container): add edge case tests for Stop()
docs(release): document archive format options
```

### Label Mapping

| Audit category | Labels |
|----------------|--------|
| OWASP/security | `security`, severity label, `lang:go` or `lang:php` |
| Error handling | `quality`, `complexity:medium` |
| Test coverage | `testing`, `complexity:medium` |
| Performance | `performance`, severity label |
| Code complexity | `quality`, `complexity:large` |
| Documentation | `docs`, `complexity:small` |
| Input validation | `security`, `quality` |
| Race conditions | `security`, `performance`, `complexity:large` |

### Step 4: Detect Patterns

After creating individual issues, look for patterns:

```
3+ issues with same fix type across different packages
  → Create a framework-level issue
  → Link individual issues as children
  → The framework fix obsoletes the individual fixes
```

**Example pattern:** 5 audit findings say "add error wrapping" in different packages. The real fix is a framework-level `errors.Wrap()` helper or middleware. Create:
- 1 framework issue: "feat(errors): add contextual error wrapping middleware"
- 5 child issues: each package migration (become validation that the framework fix works)

### Step 5: Create Epic (if enough issues)

If 3+ implementation issues were created from one audit, group them into an epic using the `create-epic` flow.

If fewer than 3, just label them for direct dispatch — no epic needed.

### Step 6: Mark Audit as Processed

Once all findings have implementation issues:

```bash
# Comment linking to created issues
gh issue comment AUDIT_NUMBER --repo OWNER/REPO \
  --body "Implementation issues created: #A, #B, #C, #D"

# Close the audit issue
gh issue close AUDIT_NUMBER --repo OWNER/REPO --reason completed
```

The audit is done. The implementation issues carry the work forward.

---

## Staleness Check

Before processing an audit, verify findings are still relevant:

```bash
# Check if the file/line still exists
gh api repos/OWNER/REPO/contents/PATH --jq '.sha' 2>&1
```

If the file was deleted or heavily refactored, the finding may be stale. But:
- **Don't discard stale findings.** The underlying pattern may still exist elsewhere.
- **Re-scan if stale.** The audit agent may have found something that moved, not something that was fixed.
- **Only skip if the entire category was resolved** (e.g. "add tests" but test coverage is now 90%).

---

## Training Data Value

Every issue created from an audit becomes training data:

| Issue outcome | Training value |
|---------------|----------------|
| Fixed correctly | Positive example: finding → fix |
| Fixed but review caught problems | Mixed: finding valid, fix needed iteration |
| Dismissed as not applicable | Negative example: audit produced false positive |
| Led to framework change | High value: pattern detection signal |
| Nitpick that revealed bigger issue | High value: small finding → large impact |

**None of these are worthless.** Even false positives teach the model what NOT to flag. Label the outcome in the training journal so the pipeline can sort them.

### Journal Extension for Audit-Origin Issues

```jsonc
{
  // ... standard journal fields ...

  "origin": {
    "type": "audit",
    "audit_issue": 183,
    "audit_category": "owasp",
    "finding_severity": "medium",
    "finding_scope": "package",
    "pattern_detected": true,
    "framework_issue": 250
  }
}
```

---

## Quick Reference

```
1. Read audit issue
2. Classify each finding (severity, type, scope, complexity)
3. Create one issue per finding (consistent title/labels)
4. Detect patterns (3+ similar → framework issue)
5. Group into epic if 3+ issues (use create-epic flow)
6. Close audit issue, link to implementation issues
```

---

*Created: 2026-02-04*
*Companion to: RFC.flow-issue-epic.md, RFC.flow-create-epic.md*
