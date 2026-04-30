---
name: excavate
description: Full archaeological dig on a dead/dying CryptoNote project
args: <project-name or org>
---

# Project Excavation

Run a full archaeological dig on an abandoned project using Borg.

## Usage

```
/collect:excavate masari
/collect:excavate graft-project
/collect:excavate turtlecoin
```

## Action

Run a multi-step excavation:

### 1. Collect GitHub repos
```bash
borg collect github repos <org> --format stim -o digs/<project>/github.stim
```

### 2. Collect website (if still up)
```bash
borg collect website https://<project-domain> --depth 3 --format stim -o digs/<project>/website.stim
```

### 3. Collect Wayback snapshots
```bash
borg collect website "https://web.archive.org/web/*/<project-domain>" --depth 2 -o digs/<project>/wayback.stim
```

### 4. Create reports
After collection, create:
- `digs/<project>/SALVAGE-REPORT.md` - What's worth keeping
- `digs/<project>/LESSONS.md` - Post-mortem analysis

## Known Dig Sites

| Project | Org | Domain | Urgency |
|---------|-----|--------|---------|
| Masari | masari-project | getmasari.org | HIGH |
| TurtleCoin | turtlecoin | turtlecoin.lol | HIGH |
| Graft | graft-project | graft.network | HIGH |
| Aeon | aeonix | aeon.cash | MEDIUM |
| Nerva | nerva-project | nerva.one | MEDIUM |
| Loki/Oxen | oxen-io | oxen.io | MEDIUM |

## Output Structure

```
digs/<project>/
├── github.stim       # All repos (encrypted)
├── website.stim      # Current site (encrypted)
├── wayback.stim      # Historical snapshots
├── SALVAGE-REPORT.md # What to keep
└── LESSONS.md        # Post-mortem
```

## Report Templates

Use templates from `skills/project-archaeology/templates/`:
- `SALVAGE-REPORT.md` - Valuable code/features
- `LESSONS.md` - Why it died, warning signs
