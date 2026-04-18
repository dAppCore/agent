---
name: collect
description: Quick collection using Borg - auto-detects resource type
args: <url or target>
---

# Quick Collect

Auto-detect and collect any resource using Borg.

## Usage

```
/collect:collect https://github.com/LetheanNetwork
/collect:collect https://docs.lethean.io
/collect:collect masari-project
```

## Action

Borg's `all` command handles auto-detection:

```bash
borg all <url-or-target>
```

This will:
1. Detect if it's a GitHub URL → collect repos
2. Detect if it's a website → crawl it
3. Detect if it's a PWA → download the app

## Examples

```bash
# GitHub org - collects all repos
borg all https://github.com/LetheanNetwork

# Website - crawls and packages
borg all https://docs.lethean.io

# With encryption
borg all https://github.com/graft-project --format stim -o graft.stim
```

## Specialised Commands

For more control, use specific commands:

| Command | Use Case |
|---------|----------|
| `/collect:github` | GitHub repos with org support |
| `/collect:website` | Website crawling with depth control |
| `/collect:excavate` | Full project archaeology |
| `/collect:papers` | Whitepaper collection from registry |
