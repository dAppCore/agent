---
name: project-archaeology
description: Deep excavation of abandoned CryptoNote projects before they vanish
---

# Project Archaeology

Deep excavation of abandoned CryptoNote projects before they vanish.

## Purpose

When a CryptoNote project dies, its artifacts scatter:
- GitHub repos get deleted or archived
- BitcoinTalk threads go stale
- Websites go offline
- Block explorers shut down
- Discord servers empty out

This skill orchestrates a **full dig** on a dead project — running all collectors in sequence to preserve everything salvageable before it's gone forever.

## Usage

```bash
# Full excavation of a project
./excavate.sh masari

# Quick scan (just check what's still accessible)
./excavate.sh masari --scan-only

# Specific collectors only
./excavate.sh masari --only=github,bitcointalk

# Resume interrupted dig
./excavate.sh masari --resume
```

## What Gets Collected

| Source | Collector Used | Priority |
|--------|----------------|----------|
| GitHub repos | `github-history` | P1 - often deleted first |
| GitHub releases | `wallet-releases` | P1 - binaries disappear |
| BitcoinTalk ANN | `bitcointalk` | P2 - usually persists |
| Website (Wayback) | `job-collector wayback` | P2 - snapshots exist |
| Block explorer | `block-explorer` | P3 - chain data |
| CoinMarketCap | `coinmarketcap` | P3 - historical prices |
| Whitepapers | `whitepaper-archive` | P1 - research value |
| Reddit | `job-collector reddit` | P4 - community context |
| Medium posts | `job-collector medium` | P4 - announcements |

## Output Structure

```
digs/
└── <project-name>/
    ├── EXCAVATION.md       # Dig log with timestamps
    ├── SALVAGE-REPORT.md   # What's worth keeping
    ├── LESSONS.md          # What killed it, what we learned
    ├── github/             # All repo history
    ├── releases/           # Wallet binaries, checksums
    ├── bitcointalk/        # Thread archive
    ├── website/            # Wayback snapshots
    ├── explorer/           # Chain data samples
    ├── market/             # Price history, volume
    ├── papers/             # Whitepapers, docs
    └── community/          # Reddit, Medium, etc
```

## Report Templates

### SALVAGE-REPORT.md
What code/ideas are worth extracting:
- Unique protocol innovations
- Wallet features
- Mining algorithms
- Community tools
- Documentation patterns

### LESSONS.md
Post-mortem analysis:
- Timeline of decline
- Root causes (dev burnout, drama, funding, tech debt)
- Warning signs to watch for
- What could have saved it

## Integration with cryptonote-discovery

```bash
# Get list of abandoned projects
cd ../cryptonote-discovery
./discover.sh --list-abandoned

# Excavate all abandoned projects (batch mode)
for proj in $(./discover.sh --list-abandoned); do
    ../project-archaeology/excavate.sh "$proj"
done
```

## Known Dig Sites

Projects confirmed dead/dying that need excavation:

| Project | Symbol | Death Year | Urgency | Notes |
|---------|--------|------------|---------|-------|
| TurtleCoin | TRTL | 2023 | HIGH | Team burned out, great docs |
| Masari | MSR | 2022 | HIGH | Uncle mining code valuable |
| Aeon | AEON | 2021 | MEDIUM | Pruning/lightweight work |
| Nerva | XNV | 2022 | MEDIUM | Anti-pool algo interesting |
| Sumokoin | SUMO | 2021 | LOW | Drama-killed, large ring research |
| Ryo | RYO | 2023 | LOW | GPU algo work |

## Requirements

- All collector skills installed
- `gh` CLI authenticated
- `jq` installed
- Sufficient disk space for archives
- Patience (full dig can take hours)

## Adding New Dig Sites

When you discover a dead CryptoNote project:

1. Add to `../cryptonote-discovery/registry.json`
2. Include `"salvageable": [...]` field
3. Run `./excavate.sh <project> --scan-only` first
4. If sources still accessible, run full dig

---

*"The past is not dead. It's not even past." — but GitHub repos definitely are.*
