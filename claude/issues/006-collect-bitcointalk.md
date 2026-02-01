# feat(collect): Add BitcoinTalk thread collection

## Summary

Add `core collect bitcointalk` command to archive BitcoinTalk forum threads.

## Required Commands

```bash
core collect bitcointalk <topic-id>           # Collect full thread
core collect bitcointalk <url>                # Collect from URL
core collect bitcointalk <id> --pages=5       # Limit pages
core collect bitcointalk <id> --output=DIR    # Custom output dir
```

## Current Shell Script Being Replaced

- `claude/skills/bitcointalk/collect.sh` - 270 lines of bash + embedded Python

## Features

1. **Rate limiting**
   - Respectful delay between requests (default 2s)
   - Configurable via `--delay=N`

2. **Post type detection**
   - ANN: Original announcement (post #1)
   - UPDATE: Contains [UPDATE]/[RELEASE]/[ANNOUNCEMENT]
   - QUESTION: Contains question mark in first 200 chars
   - COMMUNITY: General discussion

3. **Output structure**
   ```
   bitcointalk-{topic}/
   ├── INDEX.md
   ├── pages/
   │   ├── page-0.html
   │   └── page-20.html
   └── posts/
       ├── POST-0001.md
       └── POST-0002.md
   ```

4. **Post metadata**
   - Author
   - Date
   - Post type/score
   - Original content

5. **Incremental collection**
   - Resume interrupted collections
   - Skip already-fetched pages

## Output Format

```json
{
  "topic_id": "2769739",
  "title": "Lethean - Privacy Blockchain VPN",
  "posts": 1247,
  "pages": 63,
  "output": "bitcointalk-2769739/"
}
```
