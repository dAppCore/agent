---
name: website
description: Crawl and collect a website using Borg
args: <url> [--depth N] [--format stim|tim|tar] [-o output]
---

# Website Collection

Crawl and collect websites using Borg.

## Usage

```
/collect:website https://getmasari.org
/collect:website https://docs.lethean.io --depth 3
/collect:website https://graft.network --format stim -o graft-site.stim
```

## Action

Run Borg to crawl the website:

```bash
borg collect website <url> [--depth <N>] [--format <format>] [-o <output>]
```

Default depth is 2 levels.

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `--depth` | 2 | How many levels deep to crawl |
| `--format` | tar | Output format (tar, tim, stim) |
| `-o` | auto | Output filename |

## Examples

```bash
# Basic crawl
borg collect website https://getmasari.org

# Deep crawl with encryption
borg collect website https://docs.lethean.io --depth 5 --format stim -o lethean-docs.stim

# Wayback Machine archive
borg collect website "https://web.archive.org/web/*/graft.network" --depth 3
```

## Use Cases

1. **Project Documentation** - Archive docs before they go offline
2. **Wayback Snapshots** - Collect historical versions
3. **Forum Threads** - Archive discussion pages
4. **PWA Collection** - Use `borg collect pwa` for progressive web apps
