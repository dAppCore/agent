## SANDBOX: You are restricted to this directory only. No absolute paths, no cd .., no editing outside repo/.

Read these files FIRST — they are the spec:
1. CODEX.md — mandatory patterns, banned imports, Core primitives
2. .core/reference/docs/RFC.md — full Core API contract (1,284 lines, READ IT)
3. CLAUDE.md — project-specific conventions

Review ALL .go files for compliance with the spec. Report findings with file:line references.

Key checks:
- Banned imports: os, os/exec, encoding/json, fmt, errors, strings, path/filepath → Core primitives
- Error handling: core.E() not fmt.Errorf or errors.New
- Test naming: TestFile_Function_{Good,Bad,Ugly} — all 3 categories per function
- Usage-example comments: tab-indented example on every exported function
- Result pattern: core.Result not (value, error)
- UK English in comments (colour not color, initialise not initialize)
- Compile-time interface checks: var _ Interface = (*Impl)(nil)
- Keyed struct literals: core.Result{Value: x, OK: true} not core.Result{x, true}

Report findings with file:line references. Do not fix — only report.
