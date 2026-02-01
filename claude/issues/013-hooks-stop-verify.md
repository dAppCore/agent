# feat(hooks): Stop hook to verify work is complete

## Summary

Add a Stop hook that verifies Claude has actually completed the task before stopping, preventing premature "I'm done" responses.

## Problem

Claude sometimes stops too early:
- "I've updated the file" (but didn't run tests)
- "The fix is complete" (but there are lint errors)
- "Done" (but left debug statements)

## Solution

A Stop hook (type: "agent") that spawns a verification subagent to check:
1. Were tests run? Did they pass?
2. Are there uncommitted changes that should be committed?
3. Are there debug statements left in code?
4. Does the output match what was requested?

## Hook Configuration

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "agent",
            "prompt": "Verify the work is complete. Check: 1) Tests passed if code was changed, 2) No debug statements (dd, dump, fmt.Println), 3) Code is formatted. Context: $ARGUMENTS",
            "timeout": 60
          }
        ]
      }
    ]
  }
}
```

## Alternative: Command-based

For faster verification without subagent overhead:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "core ai verify-complete"
          }
        ]
      }
    ]
  }
}
```

## Verification Checks

`core ai verify-complete` should check:

1. **If code was modified:**
   - Run `core go test` or `core php test`
   - Check for debug statements
   - Verify formatting

2. **If commit was requested:**
   - Check git status for uncommitted changes

3. **Return decision:**
   ```json
   {
     "decision": "block",
     "reason": "Tests have not been run. Please run: core go test"
   }
   ```

## Guard Against Loops

Check `stop_hook_active` to prevent infinite loops:

```bash
#!/bin/bash
INPUT=$(cat)
STOP_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active')

if [ "$STOP_ACTIVE" = "true" ]; then
  # Already continued once, allow stop
  exit 0
fi

# Run verification...
```
