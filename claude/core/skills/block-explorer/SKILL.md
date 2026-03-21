---
name: block-explorer
description: Archive blockchain data from CryptoNote block explorers
---

# Block Explorer Collector

Archive blockchain data from CryptoNote block explorers.

## Data Available

| Data Type | Notes |
|-----------|-------|
| Genesis block | First block, network params |
| Block history | Height, timestamps, difficulty |
| Network stats | Hashrate, emission, supply |
| Transaction patterns | Volume, sizes, fees |
| Top addresses | Rich list (if available) |

## Common CryptoNote Explorer APIs

Most CryptoNote explorers expose similar JSON APIs:

```
/api/info                    # Network stats
/api/block/[height|hash]     # Block data
/api/transaction/[hash]      # Transaction data
/api/mempool                 # Pending transactions
/api/emission                # Supply data
```

## Usage

```bash
# Generate jobs for known explorers
./generate-jobs.sh lethean > jobs.txt

# Custom explorer URL
./generate-jobs.sh --url=https://explorer.example.com > jobs.txt

# Get historical blocks (sampling)
./generate-jobs.sh lethean --blocks=1000 --sample=daily > jobs.txt
```

## Job Output

```
# API endpoints
https://explorer.lethean.io/api/info|explorer-lthn-info.json|explorer-api|coin=lethean
https://explorer.lethean.io/api/emission|explorer-lthn-emission.json|explorer-api|coin=lethean
https://explorer.lethean.io/api/block/1|explorer-lthn-block-1.json|explorer-api|coin=lethean,block=1
```

## Known Explorers

| Project | Explorer | API |
|---------|----------|-----|
| Lethean | explorer.lethean.io | ✅ |
| Monero | xmrchain.net | ✅ |
| Haven | explorer.havenprotocol.org | ✅ |
| Karbo | explorer.karbo.io | ✅ |
| Wownero | explore.wownero.com | ✅ |

## Archived Data

```
explorer-lethean/
├── info.json               # Network summary
├── emission.json           # Supply data
├── genesis.json            # Block 0
├── blocks/
│   ├── monthly-samples.json  # One block per month
│   └── milestones.json       # Key heights
└── INDEX.md
```
