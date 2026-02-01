# feat(collect): Add collected data processing

## Summary

Add `core collect process` command to convert collected HTML/JSON files into clean markdown.

## Required Commands

```bash
core collect process <source> <downloads-dir>  # Process downloaded files
core collect process bitcointalk ./downloads   # BitcoinTalk HTML → MD
core collect process reddit ./downloads        # Reddit JSON → MD
core collect process wayback ./downloads       # Wayback HTML → MD
core collect process medium ./downloads        # Medium RSS → MD
```

## Current Shell Script Being Replaced

- `claude/skills/job-collector/process.sh` - 243 lines of bash + embedded Python

## Supported Sources

1. **bitcointalk** / **btt**
   - Input: HTML pages
   - Extract: posts, authors, dates
   - Output: POST-NNNN.md files

2. **reddit**
   - Input: JSON from Reddit API
   - Extract: posts, comments, scores
   - Output: REDDIT-NNNN.md files

3. **wayback**
   - Input: HTML from Wayback Machine
   - Extract: title, body text
   - Output: {basename}.md files

4. **medium**
   - Input: RSS/XML feed
   - Extract: title, author, date, content
   - Output: MEDIUM-NNNN.md files

## Output Structure

```
processed/
├── INDEX.md
└── posts/
    ├── POST-0001.md
    ├── POST-0002.md
    └── ...
```

## Index Generation

Auto-generates INDEX.md with:
- Source metadata
- Post count
- Links to all posts

## Output Format

```json
{
  "source": "bitcointalk",
  "input_files": 15,
  "posts_extracted": 347,
  "output": "processed/"
}
```
