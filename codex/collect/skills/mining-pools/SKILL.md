# Mining Pool Collector

Archive mining pool statistics, historical hashrate, and block discovery data.

## Data Available

| Data Type | Source | Notes |
|-----------|--------|-------|
| Current hashrate | Pool API | Network stats |
| Block history | Pool API | Found blocks, rewards |
| Miner stats | Pool API | If public |
| Payment history | Pool API | Payout records |
| Pool config | Pool page | Ports, difficulty, fees |

## Known Pools by Coin

### Lethean
- https://lethean.herominers.com/
- https://lthn.pool.gntl.uk/

### Common Pool Software
- nodejs-pool (Snipa)
- cryptonote-universal-pool
- cryptonote-nodejs-pool
- xmr-node-proxy

## Usage

```bash
# Generate jobs for known pools
./generate-jobs.sh lethean > jobs.txt

# Custom pool
./generate-jobs.sh --url=https://pool.example.com --coin=example > jobs.txt

# All known pools
./generate-jobs.sh --all > jobs.txt
```

## Output

```
pool-lethean-herominers/
├── config.json          # Pool configuration
├── network.json         # Network stats snapshot
├── blocks.json          # Found blocks
├── payments.json        # Payout history
└── INDEX.md             # Summary
```

## Job Format

```
URL|FILENAME|TYPE|METADATA
https://lethean.herominers.com/api/stats|pool-lthn-hero-stats.json|pool-api|coin=lethean,pool=herominers
https://lethean.herominers.com/api/pool/blocks|pool-lthn-hero-blocks.json|pool-api|coin=lethean,pool=herominers
```

## Notes

- Many pools use similar API formats (nodejs-pool standard)
- Historical data often not retained - snapshot what's available
- Pool shutdowns common - archive before they disappear
