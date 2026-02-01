# feat(collect): Add cryptocurrency market data collection

## Summary

Add `core collect market` command to collect historical price/market data from CoinMarketCap, CoinGecko, etc.

## Required Commands

```bash
core collect market <coin>                    # Collect current data
core collect market <coin> --historical       # Include historical data
core collect market <coin> --from=2018-01-01  # Date range
core collect market <coin1,coin2>             # Multiple coins
core collect market jobs <coin>               # Generate job list
```

## Current Shell Scripts Being Replaced

- `claude/skills/coinmarketcap/generate-jobs.sh` - Generates CMC collection jobs
- `claude/skills/coinmarketcap/process.sh` - Processes collected CMC data

## Data Sources

1. **CoinMarketCap**
   - Main page (description, links)
   - Markets/exchanges
   - Historical data
   - News
   - API endpoints (where available)

2. **CoinGecko** (backup)
   - Main page
   - API for historical data

## Output Structure

```
market/{coin}/
├── INDEX.md
├── cmc/
│   ├── main.json
│   ├── markets.json
│   └── historical.json
├── coingecko/
│   ├── detail.json
│   └── history.json
└── summary.md
```

## Job Format

For `jobs` subcommand (pipe to collector):
```
URL|FILENAME|TYPE|METADATA
https://coinmarketcap.com/currencies/lethean/|cmc-lethean-main.html|cmc-main|coin=lethean
```

## Output Format

```json
{
  "coin": "lethean",
  "sources": ["cmc", "coingecko"],
  "historical_days": 2190,
  "output": "market/lethean/"
}
```
