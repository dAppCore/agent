# GitHub History Collection Skill

Collect and score GitHub issues and PRs for triage analysis.

## Usage

```bash
# Single repo
./collect.sh https://github.com/LetheanNetwork/lthn-app-vpn

# Entire org (all repos)
./collect.sh https://github.com/LetheanNetwork --org

# Just issues (skip PRs)
./collect.sh https://github.com/LetheanNetwork/lthn-app-vpn --issues-only

# Just PRs (skip issues)
./collect.sh https://github.com/LetheanNetwork/lthn-app-vpn --prs-only

# Custom rate limit delay
./collect.sh https://github.com/LetheanNetwork --org --delay=0.5
```

## Output Structure

```
repo/
├── {org}/
│   └── {repo}/
│       ├── Issue/
│       │   ├── 001.md        # Sequential, no gaps
│       │   ├── 002.md
│       │   ├── 003.md
│       │   └── INDEX.md      # Scored index
│       ├── PR/
│       │   ├── 001.md
│       │   ├── 002.md
│       │   └── INDEX.md
│       └── .json/            # Raw API responses
│           ├── issues-list.json
│           ├── issue-{n}.json
│           ├── prs-list.json
│           └── pr-{n}.json
```

### Sequential vs GitHub Numbers

- **Filename**: `001.md`, `002.md`, etc. - sequential, no gaps
- **Inside file**: `# Issue #47: ...` - preserves original GitHub number
- **INDEX.md**: Maps both: `| 001 | #47 | Title | SCORE |`

This ensures clean sequential browsing while maintaining traceability to GitHub.

## Reception Scores

| Score | Meaning | Triage Action |
|-------|---------|---------------|
| ADDRESSED | Closed after discussion | Review if actually fixed |
| DISMISSED | Labeled wontfix/invalid | **RECLAIM candidate** |
| IGNORED | Closed, no response | **RECLAIM candidate** |
| STALE | Open, no replies | Needs attention |
| ACTIVE | Open with discussion | In progress |
| MERGED | PR accepted | Done |
| REJECTED | PR closed unmerged | Review why |
| PENDING | PR still open | Needs review |

## Requirements

- `gh` CLI authenticated (`gh auth login`)
- `jq` installed

## Batch Collection

Supports comma-separated targets for batch runs:

```bash
# Batch orgs
./collect.sh "LetheanNetwork,graft-project,oxen-io" --org

# Batch repos
./collect.sh "LetheanNetwork/lthn-app-vpn,monero-project/monero"
```

## Full Registry List

Copy-paste ready commands for the complete CryptoNote ecosystem:

```bash
# === LETHEAN ECOSYSTEM ===
./collect.sh "LetheanNetwork,letheanVPN,LetheanMovement" --org

# === CRYPTONOTE ACTIVE ===
./collect.sh "monero-project,hyle-team,zanoio,kevacoin-project,scala-network" --org
./collect.sh "Karbovanets,wownero,ConcealNetwork,ryo-currency" --org

# === SALVAGE PRIORITY (dead/abandoned) ===
./collect.sh "haven-protocol-org,graft-project,graft-community" --org
./collect.sh "oxen-io,loki-project" --org
./collect.sh "turtlecoin,masari-project,aeonix,nerva-project,sumoprojects" --org
./collect.sh "deroproject,bcndev,electroneum" --org

# === NON-CN REFERENCE ===
./collect.sh "theQRL,hyperswarm,holepunchto,openhive-network,octa-space" --org
```

### One-liner for everything

```bash
./collect.sh "LetheanNetwork,letheanVPN,LetheanMovement,monero-project,haven-protocol-org,hyle-team,zanoio,kevacoin-project,scala-network,deroproject,Karbovanets,wownero,turtlecoin,masari-project,aeonix,oxen-io,loki-project,graft-project,graft-community,nerva-project,ConcealNetwork,ryo-currency,sumoprojects,bcndev,electroneum,theQRL,hyperswarm,holepunchto,openhive-network,octa-space" --org
```

## Example Run

```bash
$ ./collect.sh "LetheanNetwork,graft-project" --org

=== Collecting all repos from org: LetheanNetwork ===
=== Collecting: LetheanNetwork/lthn-app-vpn ===
    Output: ./repo/LetheanNetwork/lthn-app-vpn/
Fetching issues...
  Found 145 issues
  Fetching issue #1 -> 001.md
  ...
  Created Issue/INDEX.md
Fetching PRs...
  Found 98 PRs
  ...
  Created PR/INDEX.md

=== Collecting all repos from org: graft-project ===
=== Collecting: graft-project/graft-network ===
    Output: ./repo/graft-project/graft-network/
...

=== Collection Complete ===
Output: ./repo/
```
