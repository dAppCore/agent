---
name: bitcointalk
description: Scrape and archive BitcoinTalk mega threads with author attribution and timestamps
---

# BitcoinTalk Thread Collector

Scrape and archive BitcoinTalk mega threads with author attribution and timestamps.

## Usage

```bash
# Single thread
./collect.sh https://bitcointalk.org/index.php?topic=2769739.0

# Just the topic ID
./collect.sh 2769739

# Limit pages (default: all)
./collect.sh 2769739 --pages=10

# Output to specific folder
./collect.sh 2769739 --output=./lethean-ann
```

## Output

```
bitcointalk-2769739/
├── thread.json          # Full structured data
├── thread.md            # Combined markdown
├── posts/
│   ├── POST-001.md      # Individual posts
│   ├── POST-002.md
│   └── ...
└── INDEX.md             # Thread overview + key posts
```

## Post Scoring

| Score | Meaning |
|-------|---------|
| ANN | Original announcement post |
| UPDATE | Official team update |
| QUESTION | Community question |
| ANSWER | Team response to question |
| SUPPORT | Positive community feedback |
| CONCERN | Raised issue/criticism |
| FUD | Identified as FUD/trolling |
| OFFTOPIC | Not relevant to project |

## Requirements

- `curl` or `wget`
- `pup` (HTML parser) or `python3` with beautifulsoup4

## Notes

- Respects rate limits (1 request per 2 seconds)
- Handles pagination automatically (.0, .20, .40, etc)
- Extracts: author, date, post rank, trust score, content
- Identifies team members vs community
