---
name: job-collector
description: Two-phase collection pattern — generate jobs, then process downloaded files
---

# Job-Based Collector

Two-phase collection pattern: generate jobs, then process downloaded files.

## Workflow

```
1. Generate jobs    →  jobs.txt (list of URLs)
2. Feed to proxy    →  (your infrastructure)
3. Process results  →  markdown output
```

## Usage

### Phase 1: Generate Job List

```bash
# BitcoinTalk thread
./generate-jobs.sh bitcointalk 2769739 > jobs.txt

# Reddit thread/subreddit
./generate-jobs.sh reddit "r/lethean" --limit=100 > jobs.txt

# Wayback Machine snapshots
./generate-jobs.sh wayback "lethean.io" > jobs.txt

# Medium author/publication
./generate-jobs.sh medium "@lethean" > jobs.txt
```

### Phase 2: Process Downloaded Files

```bash
# After proxy fetches all URLs to ./downloads/
./process.sh bitcointalk ./downloads/ --output=./archive/
```

## Job File Format

```
# jobs.txt
URL|OUTPUT_FILENAME|TYPE|METADATA
https://bitcointalk.org/index.php?topic=2769739.0|btt-2769739-p0.html|bitcointalk|page=0
https://bitcointalk.org/index.php?topic=2769739.20|btt-2769739-p20.html|bitcointalk|page=20
```

## Supported Sources

| Source | Job Generator | Processor |
|--------|---------------|-----------|
| bitcointalk | ✅ | ✅ |
| reddit | ✅ | ✅ |
| wayback | ✅ | ✅ |
| medium | ✅ | ✅ |
| twitter | 🔜 | 🔜 |
| discord-export | N/A | ✅ (local files) |
| telegram-export | N/A | ✅ (local files) |
