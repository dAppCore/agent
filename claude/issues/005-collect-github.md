# feat(collect): Add GitHub history collection

## Summary

Add `core collect github` command to archive GitHub issues and PRs from repositories and organisations.

## Required Commands

```bash
core collect github <org/repo>              # Collect issues + PRs from repo
core collect github <org> --org             # Collect all repos in org
core collect github <org1,org2> --org       # Batch collect multiple orgs
core collect github <repo> --issues-only    # Issues only
core collect github <repo> --prs-only       # PRs only
core collect github --check-rate            # Show rate limit status
```

## Current Shell Script Being Replaced

- `claude/skills/github-history/collect.sh` - 517 lines of bash

## Features

1. **Rate limit protection**
   - Check every N calls
   - Auto-pause at 25% remaining (75% used)
   - Resume after reset + 10s buffer

2. **Incremental collection**
   - Skip already-fetched issues/PRs
   - Resume interrupted collections

3. **Reception scoring**
   - ADDRESSED: Closed after discussion
   - DISMISSED: Labeled wontfix/invalid
   - IGNORED: Closed with no response
   - STALE: Open with no replies
   - ACTIVE: Open with discussion
   - MERGED/REJECTED/PENDING for PRs

4. **Output structure**
   ```
   repo/{org}/{repo}/
   ├── Issue/
   │   ├── INDEX.md
   │   ├── 001.md
   │   └── 002.md
   ├── PR/
   │   ├── INDEX.md
   │   ├── 001.md
   │   └── 002.md
   └── .json/
       ├── issues-list.json
       └── prs-list.json
   ```

5. **Index generation**
   - Markdown tables with seq, GitHub #, title, score
   - Score legend

## Output Format

Progress to stderr, final summary to stdout:
```json
{
  "repo": "host-uk/core",
  "issues": 47,
  "prs": 23,
  "output": "repo/host-uk/core/"
}
```
