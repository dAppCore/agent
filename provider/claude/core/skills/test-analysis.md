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
