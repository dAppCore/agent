# feat(collect): Add project archaeology/excavation

## Summary

Add `core collect excavate` command to perform deep excavation of abandoned cryptocurrency projects before data is lost.

## Required Commands

```bash
core collect excavate <project>              # Full excavation
core collect excavate <project> --scan-only  # Check what's accessible
core collect excavate <project> --resume     # Resume interrupted dig
core collect excavate <project> --only=github,btt  # Specific collectors
```

## Current Shell Script Being Replaced

- `claude/skills/project-archaeology/excavate.sh` - 312 lines of bash

## Excavation Pipeline

Runs collectors in priority order:

| Phase | Source | Collector | Priority | Notes |
|-------|--------|-----------|----------|-------|
| 1 | GitHub repos | `collect github` | P1 | Often deleted first |
| 1 | GitHub releases | `collect github` | P1 | Binaries disappear |
| 2 | BitcoinTalk ANN | `collect bitcointalk` | P2 | Usually persists |
| 2 | Website (Wayback) | `collect wayback` | P2 | Snapshots exist |
| 3 | Block explorer | `collect explorer` | P3 | Chain data |
| 3 | CoinMarketCap | `collect market` | P3 | Historical prices |
| 4 | Reddit | `collect reddit` | P4 | Community context |
| 4 | Medium posts | `collect medium` | P4 | Announcements |

## Project Registry

Projects defined in `cryptonote-discovery/registry.json`:
```json
{
  "name": "Masari",
  "symbol": "MSR",
  "status": "abandoned",
  "github": ["masari-project"],
  "bitcointalk": "2769739",
  "website": "getmasari.org",
  "explorer": "explorer.getmasari.org",
  "cmc": "masari"
}
```

## Output Structure

```
digs/{project}/
├── EXCAVATION.md       # Dig log with timestamps
├── SALVAGE-REPORT.md   # What's worth keeping
├── LESSONS.md          # Post-mortem analysis
├── github/             # All repo history
├── releases/           # Wallet binaries, checksums
├── bitcointalk/        # Thread archive
├── website/            # Wayback snapshots
├── explorer/           # Chain data samples
├── market/             # Price history
├── papers/             # Whitepapers, docs
└── community/          # Reddit, Medium, etc
```

## Scan Mode

`--scan-only` checks accessibility without downloading:
- GitHub org accessible?
- BitcoinTalk thread exists?
- Wayback snapshots available?
- Block explorer online?
- CMC page exists?

## Output Format

```json
{
  "project": "masari",
  "status": "abandoned",
  "accessible": {
    "github": true,
    "bitcointalk": true,
    "wayback": true,
    "explorer": false,
    "cmc": true
  },
  "output": "digs/masari/"
}
```
