## SANDBOX: You are restricted to this directory only. No absolute paths, no cd .., no editing outside src/.

Read CLAUDE.md for project context.
Review all Go files in src/ for security issues:
- Path traversal vulnerabilities
- Unvalidated input
- SQL injection (if applicable)
- Hardcoded credentials or tokens
- Unsafe type assertions
- Missing error checks
- Race conditions (shared state without mutex)
- Unsafe use of os/exec

Report findings with severity (critical/high/medium/low) and file:line references.
