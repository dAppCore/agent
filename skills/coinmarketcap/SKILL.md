# CoinMarketCap Collector

Archive coin data, historical prices, and metadata from CoinMarketCap.

## Data Available

| Data Type | Source | Notes |
|-----------|--------|-------|
| Current price/market cap | Main page | Live data |
| Historical prices | /historical-data/ | OHLCV by date range |
| Project description | Main page | About section |
| Social links | Main page | Twitter, Discord, etc |
| Exchanges | /markets/ | Trading pairs |
| On-chain data | /onchain-analysis/ | If available |
| News mentions | /news/ | Related articles |

## Usage

### Generate Jobs

```bash
# All data for a coin
./generate-jobs.sh lethean > jobs.txt

# Just historical prices (date range)
./generate-jobs.sh lethean --historical --from=2018-01-01 --to=2024-12-31 > jobs.txt

# Multiple coins
./generate-jobs.sh lethean monero bitcoin > jobs.txt
```

### Process Downloads

```bash
./process.sh ./downloads/ --output=./cmc-archive/
```

## Output

```
cmc-lethean/
├── metadata.json        # Name, symbol, links, description
├── current.json         # Latest price/mcap/volume
├── historical/
│   ├── 2018.csv         # OHLCV data
│   ├── 2019.csv
│   └── ...
├── markets.json         # Exchange listings
└── INDEX.md             # Summary
```

## Job Format

```
URL|FILENAME|TYPE|METADATA
https://coinmarketcap.com/currencies/lethean/|cmc-lethean-main.html|cmc-main|coin=lethean
https://coinmarketcap.com/currencies/lethean/historical-data/|cmc-lethean-historical.html|cmc-historical|coin=lethean
```

## Notes

- CMC has rate limiting - use delays
- Historical data may require pagination
- Some data behind API paywall - scrape public pages
