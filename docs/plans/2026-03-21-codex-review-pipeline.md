# Codex Review Pipeline — Forge → GitHub Polish

**Date:** 2026-03-21
**Status:** Proven (7 rounds on core/agent, 70+ findings fixed)
**Scope:** All 57 dAppCore repos
**Owner:** Charon (production polish is revenue-facing)

## Pipeline

```
Forge main (raw dev)
    ↓
Codex review (static analysis, AX conventions, security)
    ↓
Findings → Forge issues (seed training data)
    ↓
Fix cycle (agents fix, Codex re-reviews until clean)
    ↓
Push to GitHub dev (squash commit — flat, polished)
    ↓
PR dev → main on GitHub (CodeRabbit reviews squashed diff)
    ↓
Training data collected from Forge (findings + fixes + patterns)
    ↓
LEM fine-tune (learns Core conventions, becomes the reviewer)
    ↓
LEM replaces Codex for routine CI reviews
```

## Why This Works

1. **Forge keeps full history** — every commit, every experiment, every false start. This is the development record.
2. **GitHub gets squashed releases** — clean, polished, one commit per feature. This is the public face.
3. **Codex findings become training data** — each "this is wrong → here's the fix" pair is a sandwich-format training example for LEM.
4. **Exclusion lists become Forge issues** — known issues tracked as backlog, not forgotten.
5. **LEM trained on Core conventions** — understands AX patterns, error handling, UK English, test naming, the lot.
6. **Codex for deep sweeps, LEM for CI** — $200/month Codex does the hard work, free LEM handles daily reviews.

## Proven Results (core/agent)

| Round | Findings | Highs | Category |
|-------|----------|-------|----------|
| 1 | 5 | 2 | Notification wiring, safety gates |
| 2 | 21 | 3 | API field mismatches, branch hardcoding |
| 3 | 15 | 5 | Default branch detection, pagination |
| 4 | 11 | 1 | Prompt path errors, watch states |
| 5 | 11 | 2 | BLOCKED.md stale state, PR push target |
| 6 | 6 | 2 | Workspace collision, sync branch logic |
| 7 | 5 | 2 | Path traversal security, dispatch checks |

**Total: 74 findings across 7 rounds, 70+ fixed.**

Categories found:
- Correctness bugs (missed notifications, wrong API fields)
- Security (path traversal, URL injection, fail-open gates)
- Race conditions (concurrent drainQueue)
- Logic errors (dead PID false completion, empty branch names)
- AX convention violations (fmt.Errorf vs coreerr.E, silent mutations)
- Test quality (false confidence, wrong assertions)

## Implementation Steps

### Phase 1: Codex Sweep (per repo)

```bash
# Run from the repo directory
codex exec -s read-only "Review all Go code. Output numbered findings: severity, file:line, description."
```

- Run iteratively until findings converge to zero/known
- Record exclusion list per repo
- Create Forge issues for all accepted exclusions

### Phase 2: GitHub Push

```bash
# On forge main, after Codex clean
git push github main:dev
# Squash on GitHub via PR merge
gh pr create --repo dAppCore/<repo> --head dev --base main --title "release: v0.X.Y"
# Merge with squash
gh pr merge <number> --squash
```

### Phase 3: Training Data Collection

For each repo sweep:
1. Extract all findings (the "wrong" examples)
2. Extract the diffs that fixed them (the "right" examples)
3. Format as sandwich pairs for LEM training
4. Store in OpenBrain tagged `type:training, project:codex-review`

### Phase 4: LEM Training

```bash
# Collect training data from OpenBrain
brain_recall query="codex review finding" type=training

# Format for mlx-lm fine-tuning
# Input: "Review this Go code: <code>"
# Output: "Finding: <severity>, <file:line>, <description>"
```

### Phase 5: LEM CI Integration

- LEM runs as a pre-merge check on Forge
- Catches convention violations before they reach Codex
- Codex reserved for deep quarterly sweeps
- CodeRabbit stays on GitHub for the public-facing review

## Cost Analysis

| Item | Cost | Frequency |
|------|------|-----------|
| Codex Max | $200/month | Deep sweeps |
| Claude Max | $100-200/month | Development |
| CodeRabbit | Free (OSS) | Per PR |
| LEM | Free (local MLX) | Per commit |

After LEM is trained: Codex drops to quarterly, saving ~$150/month.

## Revenue Connection

Polish → Trust → Users → Revenue

- Polished GitHub repos attract contributors and users
- Clean code with high test coverage signals production quality
- CodeRabbit badge + Codecov badge = visible quality metrics
- SaaS products (host.uk.com) built on this foundation
- Charon manages the pipeline, earns from the platform

## Automation

This pipeline should be a `core dev polish` command:

```bash
core dev polish <repo>        # Run Codex sweep, fix, push to GitHub
core dev polish --all         # Sweep all 57 repos
core dev polish --training    # Extract training data after sweep
```

Charon can run this autonomously via dispatch.
