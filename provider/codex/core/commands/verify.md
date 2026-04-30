---
name: verify
description: Verify work is complete before stopping
args: [--quick|--full]
---

# Work Verification

Verify that work is complete and ready to commit or push.

## Verification Steps

1. Check for uncommitted changes
2. Check for debug statements
3. Run tests
4. Run lint and static analysis
5. Check formatting

## Output

Return a READY or NOT READY verdict with the specific failing checks called out first.
