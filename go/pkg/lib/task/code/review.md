# Code Review Task

Review all changed files for bugs, security issues, and convention violations.

## Process

1. Run `git diff --name-only origin/main..HEAD` to find changed files
2. Read each changed file
3. Check against the conventions in `review/conventions.md`
4. Rate each finding by confidence (0-100, report >= 50)
5. Output findings by severity

## Output

```
[SEVERITY] file.go:LINE (confidence: N)
Description of the issue.
Suggested fix.
```

End with: `X critical, Y high, Z medium, W low findings.`
