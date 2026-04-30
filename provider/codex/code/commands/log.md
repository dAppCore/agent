---
name: log
description: Smart log viewing with filtering and analysis.
args: [--errors|--since <duration>|--grep <pattern>|--request <id>|analyse]
---

# Smart Log Viewing

Tails, filters, and analyzes `laravel.log`.

## Usage

/core:log                     # Tail laravel.log
/core:log --errors            # Only errors
/core:log --since 1h          # Last hour
/core:log --grep "User"       # Filter by pattern
/core:log --request abc123    # Show logs for a specific request
/core:log analyse             # Summarize errors

## Action

This command is implemented by the script at `claude/code/scripts/log.sh`.
