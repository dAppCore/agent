#!/usr/bin/env bash
# Generate job list for CoinMarketCap collection
# Usage: ./generate-jobs.sh <coin-slug> [options] > jobs.txt

set -e

COINS=()
HISTORICAL=0
FROM_DATE="2017-01-01"
TO_DATE=$(date +%Y-%m-%d)

# Parse args
for arg in "$@"; do
    case "$arg" in
        --historical) HISTORICAL=1 ;;
        --from=*) FROM_DATE="${arg#*=}" ;;
        --to=*) TO_DATE="${arg#*=}" ;;
        --*) ;;
        *) COINS+=("$arg") ;;
    esac
done

if [ ${#COINS[@]} -eq 0 ]; then
    echo "Usage: $0 <coin-slug> [coin-slug...] [--historical] [--from=DATE] [--to=DATE]" >&2
    echo "" >&2
    echo "Examples:" >&2
    echo "  $0 lethean" >&2
    echo "  $0 lethean --historical --from=2018-01-01" >&2
    echo "  $0 lethean monero bitcoin" >&2
    exit 1
fi

# Header
echo "# CoinMarketCap job list - $(date +%Y-%m-%d)"
echo "# Coins: ${COINS[*]}"
echo "# Format: URL|FILENAME|TYPE|METADATA"
echo "#"

for COIN in "${COINS[@]}"; do
    SLUG=$(echo "$COIN" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')

    echo "# === $SLUG ==="

    # Main page (current data, description, links)
    echo "https://coinmarketcap.com/currencies/${SLUG}/|cmc-${SLUG}-main.html|cmc-main|coin=$SLUG"

    # Markets/exchanges
    echo "https://coinmarketcap.com/currencies/${SLUG}/markets/|cmc-${SLUG}-markets.html|cmc-markets|coin=$SLUG"

    # Historical data page
    echo "https://coinmarketcap.com/currencies/${SLUG}/historical-data/|cmc-${SLUG}-historical.html|cmc-historical|coin=$SLUG"

    # News
    echo "https://coinmarketcap.com/currencies/${SLUG}/news/|cmc-${SLUG}-news.html|cmc-news|coin=$SLUG"

    # API endpoints (if accessible without auth)
    # These return JSON and are more reliable than scraping
    echo "https://api.coinmarketcap.com/data-api/v3/cryptocurrency/detail?slug=${SLUG}|cmc-${SLUG}-api-detail.json|cmc-api|coin=$SLUG,type=detail"
    echo "https://api.coinmarketcap.com/data-api/v3/cryptocurrency/market-pairs/latest?slug=${SLUG}&limit=100|cmc-${SLUG}-api-markets.json|cmc-api|coin=$SLUG,type=markets"

    # Historical data via API (may need date chunks)
    if [ "$HISTORICAL" = "1" ]; then
        echo "#"
        echo "# Historical data: $FROM_DATE to $TO_DATE"

        # Convert dates to timestamps
        FROM_TS=$(date -j -f "%Y-%m-%d" "$FROM_DATE" "+%s" 2>/dev/null || date -d "$FROM_DATE" "+%s")
        TO_TS=$(date -j -f "%Y-%m-%d" "$TO_DATE" "+%s" 2>/dev/null || date -d "$TO_DATE" "+%s")

        # CMC historical API (public, limited)
        echo "https://api.coinmarketcap.com/data-api/v3/cryptocurrency/historical?slug=${SLUG}&timeStart=${FROM_TS}&timeEnd=${TO_TS}|cmc-${SLUG}-api-historical.json|cmc-api|coin=$SLUG,type=historical"

        # Also try the web scrape version with date range
        echo "https://coinmarketcap.com/currencies/${SLUG}/historical-data/?start=${FROM_DATE//\-/}&end=${TO_DATE//\-/}|cmc-${SLUG}-historical-range.html|cmc-historical|coin=$SLUG,from=$FROM_DATE,to=$TO_DATE"
    fi

    echo "#"
done

echo "# === Additional data sources ==="
echo "#"

# CoinGecko as backup (often has more historical data)
for COIN in "${COINS[@]}"; do
    SLUG=$(echo "$COIN" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
    echo "https://www.coingecko.com/en/coins/${SLUG}|coingecko-${SLUG}-main.html|coingecko|coin=$SLUG"
    echo "https://api.coingecko.com/api/v3/coins/${SLUG}|coingecko-${SLUG}-api.json|coingecko-api|coin=$SLUG"
    echo "https://api.coingecko.com/api/v3/coins/${SLUG}/market_chart?vs_currency=usd&days=max|coingecko-${SLUG}-history.json|coingecko-api|coin=$SLUG,type=history"
done
