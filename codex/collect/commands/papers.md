---
name: papers
description: Collect whitepapers from the ledger-papers registry
args: [--category <name>] [--all] [--search <term>]
---

# Whitepaper Collection

Collect academic papers and whitepapers from the registry.

## Usage

```
/collect:papers --category cryptography
/collect:papers --all
/collect:papers --search bulletproofs
```

## Action

### List categories
```bash
jq -r '.papers[].category' skills/ledger-papers/registry.json | sort -u
```

### Collect by category
```bash
# Get URLs for a category
jq -r '.papers[] | select(.category == "<category>") | .url' skills/ledger-papers/registry.json > urls.txt

# Download each
while read url; do
  borg collect website "$url" --depth 0
done < urls.txt
```

### Collect all
```bash
jq -r '.papers[].url' skills/ledger-papers/registry.json | while read url; do
  borg collect website "$url" --depth 0
done
```

## Categories

| Category | Count | Examples |
|----------|-------|----------|
| genesis | 4 | Bitcoin, b-money, hashcash |
| cryptonote | 2 | CryptoNote v2.0, CNS standards |
| mrl | 11 | Monero Research Lab papers |
| privacy | 9 | Zcash, Mimblewimble, Lelantus |
| cryptography | 10 | Bulletproofs, CLSAG, PLONK |
| consensus | 7 | PBFT, Tendermint, Casper |
| defi | 7 | Uniswap, Aave, Compound |
| layer2 | 7 | Lightning, Plasma, Rollups |

## Academic Sources

For papers not in registry, search:

```bash
# IACR ePrint
borg collect website "https://eprint.iacr.org/search?q=<term>" --depth 1

# arXiv
borg collect website "https://arxiv.org/search/?query=<term>&searchtype=all" --depth 1
```

## Output

Papers are collected to:
```
skills/ledger-papers/archive/<category>/<paper>.pdf
```
