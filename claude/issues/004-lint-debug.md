# feat(qa): Add debug statement detection

## Summary

Add `core qa debug` command to detect debug statements in code, for use in Claude Code PostToolUse hooks.

## Required Commands

```bash
core qa debug <file>        # Check single file for debug statements
core qa debug --staged      # Check all staged files
core qa debug --changed     # Check all changed files
```

## Current Shell Script Being Replaced

- `claude/scripts/check-debug.sh` - Warns about debug statements after edits

## Detection Patterns

**Go files (*.go):**
- `fmt.Println`
- `fmt.Printf` (without format verbs suggesting actual logging)
- `log.Println`
- `log.Printf`
- `spew.Dump`
- `pp.Println`

**PHP files (*.php):**
- `dd(`
- `dump(`
- `var_dump(`
- `print_r(`
- `ray(`

**JavaScript/TypeScript (*.js, *.ts, *.tsx):**
- `console.log`
- `console.debug`
- `debugger`

## Output Format

```json
{
  "file": "src/main.go",
  "warnings": [
    {"line": 42, "type": "fmt.Println", "content": "fmt.Println(\"debug\")"},
    {"line": 87, "type": "log.Println", "content": "log.Println(err)"}
  ]
}
```

For hooks, output to stderr:
```
[qa] WARNING: Debug statements found in src/main.go
  42: fmt.Println("debug")
  87: log.Println(err)
```
