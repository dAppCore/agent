---
name: github
description: Collect GitHub repositories or entire organisations using Borg
args: <url or org name> [--format stim|tim|tar] [-o output]
---

# GitHub Collection

Collect GitHub repositories using Borg.

## Usage

```
/collect:github LetheanNetwork
/collect:github https://github.com/monero-project/monero
/collect:github graft-project --format stim -o graft.stim
```

## Action

Determine if the argument is a repo URL or org name, then run the appropriate Borg command:

**For organisation (all repos):**
```bash
borg collect github repos <org> [--format <format>] [-o <output>]
```

**For single repo:**
```bash
borg collect github repo <url> [--format <format>] [-o <output>]
```

## Formats

| Format | Extension | Description |
|--------|-----------|-------------|
| tar | .tar | Plain tarball |
| tim | .tim | OCI-compatible container bundle |
| stim | .stim | Encrypted container (will prompt for password) |

## Examples

```bash
# Clone all Lethean repos
borg collect github repos LetheanNetwork

# Clone and encrypt
borg collect github repos graft-project --format stim -o graft-archive.stim

# Single repo
borg collect github repo https://github.com/monero-project/monero
```

## Target Registry

See `skills/github-history/SKILL.md` for the full list of CryptoNote orgs to collect.

### Quick Targets

**Active:**
- `monero-project`, `hyle-team`, `zanoio`, `wownero`

**Salvage Priority:**
- `graft-project`, `turtlecoin`, `masari-project`, `oxen-io`
