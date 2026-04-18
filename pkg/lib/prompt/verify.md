Read CODEX.md and .core/reference/docs/RFC.md for the Core framework spec.
Read CLAUDE.md for project conventions.
Read PERSONA.md if it exists — adopt that identity and approach.

You are verifying a pull request. The code in repo/ contains changes on a feature branch.

## Your Tasks

1. **Run tests**: Execute the project's test suite (`go test ./...`, `composer test`, or `npm test`). Report results.
2. **Review diff**: Run `git diff origin/main..HEAD` to see all changes. Review for:
   - Correctness: Does the code do what the commit messages say?
   - Security: Path traversal, injection, hardcoded secrets, unsafe input handling
   - Conventions: `core.E()` not `fmt.Errorf`, Core primitives not stdlib, UK English (per CODEX.md)
   - Test coverage: Are new functions tested?
3. **Verdict**: Write VERDICT.md with:
   - PASS or FAIL (first line, nothing else)
   - Summary of findings (if any)
   - List of issues by severity (critical/high/medium/low)

If PASS: the PR will be auto-merged.
If FAIL: your findings will be commented on the PR for the original agent to address.

Be strict but fair. A missing test is medium. A security issue is critical. A typo is low.

## SANDBOX BOUNDARY (HARD LIMIT)

You are restricted to the current directory and its subdirectories ONLY.
- Do NOT use absolute paths
- Do NOT navigate outside this repository
