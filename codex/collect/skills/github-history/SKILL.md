# GitHub History Collection Skill

Collect GitHub repositories, issues, and PRs for archival and triage analysis.

## Prerequisites

```bash
# Install Borg
go install github.com/Snider/Borg@latest
```

## Usage

```bash
# Clone a single repository
borg collect github repo https://github.com/LetheanNetwork/lthn-app-vpn

# Clone all repos from an org
borg collect github repos LetheanNetwork

# Output to encrypted container
borg collect github repos LetheanNetwork --format stim -o lethean.stim
```

## Target Registry

### Lethean Ecosystem
- `LetheanNetwork`
- `letheanVPN`
- `LetheanMovement`

### CryptoNote Active
- `monero-project`
- `hyle-team`
- `zanoio`
- `kevacoin-project`
- `scala-network`
- `Karbovanets`
- `wownero`
- `ConcealNetwork`
- `ryo-currency`

### Salvage Priority (dead/abandoned)
- `haven-protocol-org`
- `graft-project`
- `graft-community`
- `oxen-io`
- `loki-project`
- `turtlecoin`
- `masari-project`
- `aeonix`
- `nerva-project`
- `sumoprojects`
- `deroproject`
- `bcndev`
- `electroneum`

### Non-CN Reference
- `theQRL`
- `hyperswarm`
- `holepunchto`
- `openhive-network`
- `octa-space`

## Batch Collection

```bash
# Collect everything into encrypted archive
borg collect github repos LetheanNetwork,monero-project,graft-project \
  --format stim -o cryptonote-archive.stim
```

## Triage Workflow

1. Collect repos with Borg
2. Review issues marked DISMISSED or IGNORED
3. Identify salvageable features
4. Document in project-archaeology skill
