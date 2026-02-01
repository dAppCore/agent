# feat(hooks): SessionStart context injection

## Summary

Add SessionStart hooks that inject relevant project context at the start of each session.

## Problem

Claude starts each session without knowing:
- Recent git changes
- Open issues assigned to the user
- CI status
- Current branch and uncommitted work

This leads to repeated context-gathering at the start of every session.

## Solution

SessionStart hook that runs `core` commands to gather context and inject it.

## Hook Configuration

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "core ai session-context",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
```

## Context to Gather

`core ai session-context` should collect:

1. **Git status:**
   ```bash
   BRANCH=$(git branch --show-current)
   CHANGES=$(git status --short | head -10)
   ```

2. **Recent activity:**
   ```bash
   COMMITS=$(git log --oneline -5)
   ```

3. **Open issues (if gh available):**
   ```bash
   ISSUES=$(core dev issues --assignee @me --limit 5)
   ```

4. **CI status:**
   ```bash
   CI=$(core qa health --brief)
   ```

5. **Restored session state:**
   ```bash
   if [ -f ~/.claude/sessions/scratchpad.md ]; then
     # Include previous session context
   fi
   ```

## Output Format

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "## Session Context\n\n**Branch:** feature/foo\n**Uncommitted:** 3 files\n\n**Recent commits:**\n- abc123 fix: resolve test flake\n- def456 feat: add new endpoint\n\n**Your issues:**\n- #42 Bug in login flow\n- #38 Add caching\n\n**CI:** All green ✓"
  }
}
```

## Environment Variables

Also set useful env vars via `CLAUDE_ENV_FILE`:

```bash
if [ -n "$CLAUDE_ENV_FILE" ]; then
  echo "export PROJECT_ROOT=\"$(pwd)\"" >> "$CLAUDE_ENV_FILE"
  echo "export GIT_BRANCH=\"$BRANCH\"" >> "$CLAUDE_ENV_FILE"
fi
```

## Conditional Context

Only inject what's relevant:
- Skip CI status if not in a repo with workflows
- Skip issues if gh not authenticated
- Skip session restore if >3 hours old
