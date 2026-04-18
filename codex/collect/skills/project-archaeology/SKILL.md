# Project Archaeology

Deep excavation of abandoned CryptoNote projects before they vanish.

## Purpose

When a CryptoNote project dies, its artifacts scatter:
- GitHub repos get deleted or archived
- BitcoinTalk threads go stale
- Websites go offline
- Block explorers shut down
- Discord servers empty out

This skill orchestrates a **full dig** using Borg to preserve everything salvageable.

## Collection with Borg

```bash
# Clone all repos from a dying project
borg collect github repos masari-project --format stim -o masari-github.stim

# Archive the website via Wayback
borg collect website https://web.archive.org/web/*/getmasari.org --depth 3

# Package everything into encrypted archive
borg compile -f Borgfile -e "archive-password" -o masari-full-dig.stim
```

## What Gets Collected

| Source | Borg Command | Priority |
|--------|--------------|----------|
| GitHub repos | `borg collect github repos <org>` | P1 |
| GitHub releases | `borg collect github repo <url>` | P1 |
| Websites | `borg collect website <url>` | P2 |
| Wayback snapshots | `borg collect website web.archive.org/...` | P2 |

## Output Structure

```
digs/
└── <project-name>/
    ├── SALVAGE-REPORT.md   # What's worth keeping
    ├── LESSONS.md          # What killed it, what we learned
    ├── github.stim         # All repo history (encrypted)
    ├── website.stim        # Website snapshots (encrypted)
    └── papers/             # Whitepapers, docs
```

## Report Templates

### SALVAGE-REPORT.md
What code/ideas are worth extracting:
- Unique protocol innovations
- Wallet features
- Mining algorithms
- Community tools

### LESSONS.md
Post-mortem analysis:
- Timeline of decline
- Root causes (dev burnout, drama, funding, tech debt)
- Warning signs to watch for

## Known Dig Sites

| Project | Symbol | Death Year | Urgency | Notes |
|---------|--------|------------|---------|-------|
| TurtleCoin | TRTL | 2023 | HIGH | Team burned out, great docs |
| Masari | MSR | 2022 | HIGH | Uncle mining code valuable |
| Aeon | AEON | 2021 | MEDIUM | Pruning/lightweight work |
| Nerva | XNV | 2022 | MEDIUM | Anti-pool algo interesting |
| Sumokoin | SUMO | 2021 | LOW | Drama-killed, large ring research |
| Ryo | RYO | 2023 | LOW | GPU algo work |

## Batch Excavation

```bash
# Collect multiple dead projects
for org in turtlecoin masari-project aeonix nerva-project; do
  borg collect github repos "$org" --format stim -o "digs/${org}.stim"
done
```

---

*"The past is not dead. It's not even past." — but GitHub repos definitely are.*
