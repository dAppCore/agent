## SANDBOX: You are restricted to this directory only. No absolute paths, no cd .., no editing outside src/.

Read CLAUDE.md for project conventions.
Review all Go files in src/ for:
- Error handling: should use coreerr.E() from go-log, not fmt.Errorf or errors.New
- Compile-time interface checks: var _ Interface = (*Impl)(nil)
- Import aliasing: stdlib io aliased as goio
- UK English in comments (colour not color, initialise not initialize)
- No fmt.Print* debug statements (use go-log)
- Test coverage gaps

Report findings with file:line references. Do not fix — only report.
