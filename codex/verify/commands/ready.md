---
name: ready
description: Quick check if work is ready to commit
---

# Ready Check

Quick verification that work is ready to commit.

## Checks

1. **No uncommitted changes left behind**
2. **No debug statements**
3. **Code is formatted**

## Process

```bash
# Check for changes
git status --porcelain

# Quick format check
core go fmt --check 2>/dev/null || core php fmt --test 2>/dev/null
```

## Output

```
## Ready Check

✓ All changes staged
✓ No debug statements
✓ Code formatted

**Ready to commit!**
```

Or:

```
## Ready Check

✗ Unstaged changes: 2 files
✓ No debug statements
✗ Formatting needed: 1 file

**Not ready** - run `/verify:verify` for details
```

## When to Use

Use `/verify:ready` for a quick check before committing.
Use `/verify:verify` for full verification including tests.
