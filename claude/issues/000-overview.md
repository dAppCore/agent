# feat: Replace shell scripts with core CLI commands

## Summary

This tracking issue covers the migration from shell scripts to native `core` CLI commands in `core-agent`. Every shell script should be replaced by a `core` command, making the plugin purely configuration (JSON, MD) + calls to `core`.

## Philosophy

- Every agent using `core` tests the framework
- Missing features get raised as issues
- Code as if functionality exists (TDD approach)
- The CLI becomes battle-tested and bulletproof

## Issues

### Claude Code Hooks (core ai)

| Issue | Command | Replaces |
|-------|---------|----------|
| #001 | `core ai session` | pre-compact.sh, session-start.sh, suggest-compact.sh |
| #002 | `core ai context` | capture-context.sh, extract-actionables.sh |
| #003 | `core ai hook` | prefer-core.sh, block-docs.sh, post-commit-check.sh |

### Quality Assurance (core qa)

| Issue | Command | Replaces |
|-------|---------|----------|
| #004 | `core qa debug` | check-debug.sh |

### Data Collection (core collect)

| Issue | Command | Replaces |
|-------|---------|----------|
| #005 | `core collect github` | github-history/collect.sh |
| #006 | `core collect bitcointalk` | bitcointalk/collect.sh |
| #007 | `core collect market` | coinmarketcap/*.sh |
| #008 | `core collect papers` | ledger-papers/discover.sh, collect-whitepaper.sh |
| #009 | `core collect excavate` | project-archaeology/excavate.sh |
| #010 | `core collect process` | job-collector/process.sh |
| #011 | `core collect dispatch` | collection/dispatch.sh |

## Target State

After all issues are implemented:

```
claude/
├── hooks/
│   └── hooks.json          # Calls core ai hook validate-*
├── commands/
│   └── remember.md         # Calls core ai context add
├── collection/
│   └── hooks.json          # Calls core collect dispatch
└── skills/
    └── */
        └── SKILL.md        # Documentation only, calls core collect *
```

No shell scripts. Just JSON config + markdown docs + `core` CLI calls.

### Hook Improvements (feedback cycle)

| Issue | Feature | Purpose |
|-------|---------|---------|
| #012 | Test output filtering | Reduce noise, show only failures |
| #013 | Stop verification | Verify work complete before stopping |
| #014 | Auto-test on edit | Run tests async after code changes |
| #015 | Session context | Inject git/issues/CI context on start |
| #016 | Silent formatting | Auto-format without output noise |
| #017 | Expose/hide policy | Define what to show vs suppress |

## Implementation Order

1. **Phase 1**: `core ai session` + `core ai context` (enables hooks to work)
2. **Phase 2**: `core ai hook` + `core qa debug` (safety + quality)
3. **Phase 3**: Hook improvements (012-017) - better feedback cycle
4. **Phase 4**: `core collect github` + `core collect bitcointalk` (most used)
5. **Phase 5**: Remaining collection commands
