---
name: qa
description: Run iterative QA loop until all checks pass
args: [--fix] [--quick]
run: ${CLAUDE_PLUGIN_ROOT}/scripts/qa.sh $@
---

# QA Loop

Run QA checks and fix issues iteratively.

## Action
1. Detect project type from go.mod or composer.json
2. Run `core go qa` or `core php qa`
3. Parse output for fixable issues
4. Apply fixes and re-run
5. Report final status
