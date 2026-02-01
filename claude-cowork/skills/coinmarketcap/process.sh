#!/usr/bin/env bash
# Process downloaded CoinMarketCap data
# Usage: ./process.sh <downloads-dir> [--output=DIR]

set -e

DOWNLOADS="$1"
OUTPUT="./cmc-archive"

for arg in "$@"; do
    case "$arg" in
        --output=*) OUTPUT="${arg#*=}" ;;
    esac
done

mkdir -p "$OUTPUT"

echo "=== Processing CoinMarketCap downloads ==="

# Process API JSON files first (most reliable)
for file in "$DOWNLOADS"/cmc-*-api-detail.json; do
    [ -f "$file" ] || continue

    COIN=$(basename "$file" | sed 's/cmc-\(.*\)-api-detail.json/\1/')
    COIN_DIR="$OUTPUT/$COIN"
    mkdir -p "$COIN_DIR"

    echo "Processing: $COIN"

    python3 << PYEOF
import json
import os

try:
    data = json.load(open('$file', 'r'))

    if 'data' in data:
        coin = data['data']

        # Extract metadata
        metadata = {
            'id': coin.get('id'),
            'name': coin.get('name'),
            'symbol': coin.get('symbol'),
            'slug': coin.get('slug'),
            'description': coin.get('description', ''),
            'logo': coin.get('logo'),
            'category': coin.get('category'),
            'dateAdded': coin.get('dateAdded'),
            'urls': coin.get('urls', {}),
            'tags': coin.get('tags', []),
        }

        with open('$COIN_DIR/metadata.json', 'w') as f:
            json.dump(metadata, f, indent=2)
        print(f"  Created metadata.json")

        # Create markdown summary
        with open('$COIN_DIR/INDEX.md', 'w') as f:
            f.write(f"# {metadata['name']} ({metadata['symbol']})\n\n")
            f.write(f"## Metadata\n\n")
            f.write(f"| Field | Value |\n")
            f.write(f"|-------|-------|\n")
            f.write(f"| Name | {metadata['name']} |\n")
            f.write(f"| Symbol | {metadata['symbol']} |\n")
            f.write(f"| CMC ID | {metadata['id']} |\n")
            f.write(f"| Added | {metadata['dateAdded']} |\n")
            f.write(f"| Category | {metadata.get('category', 'N/A')} |\n\n")

            f.write(f"## Links\n\n")
            urls = metadata.get('urls', {})
            for url_type, url_list in urls.items():
                if url_list:
                    f.write(f"- **{url_type}**: {', '.join(url_list[:3])}\n")

            f.write(f"\n## Description\n\n")
            f.write(metadata.get('description', 'No description')[:2000])
            f.write("\n")

        print(f"  Created INDEX.md")

except Exception as e:
    print(f"  Error processing: {e}")
PYEOF
done

# Process historical data
for file in "$DOWNLOADS"/cmc-*-api-historical.json; do
    [ -f "$file" ] || continue

    COIN=$(basename "$file" | sed 's/cmc-\(.*\)-api-historical.json/\1/')
    COIN_DIR="$OUTPUT/$COIN"
    mkdir -p "$COIN_DIR/historical"

    echo "Processing historical: $COIN"

    python3 << PYEOF
import json
import csv
from datetime import datetime

try:
    data = json.load(open('$file', 'r'))

    if 'data' in data and 'quotes' in data['data']:
        quotes = data['data']['quotes']

        # Group by year
        by_year = {}
        for quote in quotes:
            ts = quote.get('timestamp', quote.get('time', ''))
            if ts:
                year = ts[:4]
                if year not in by_year:
                    by_year[year] = []
                by_year[year].append({
                    'date': ts[:10],
                    'open': quote.get('quote', {}).get('USD', {}).get('open', quote.get('open')),
                    'high': quote.get('quote', {}).get('USD', {}).get('high', quote.get('high')),
                    'low': quote.get('quote', {}).get('USD', {}).get('low', quote.get('low')),
                    'close': quote.get('quote', {}).get('USD', {}).get('close', quote.get('close')),
                    'volume': quote.get('quote', {}).get('USD', {}).get('volume', quote.get('volume')),
                    'market_cap': quote.get('quote', {}).get('USD', {}).get('market_cap', quote.get('market_cap')),
                })

        for year, rows in by_year.items():
            filename = f'$COIN_DIR/historical/{year}.csv'
            with open(filename, 'w', newline='') as f:
                writer = csv.DictWriter(f, fieldnames=['date', 'open', 'high', 'low', 'close', 'volume', 'market_cap'])
                writer.writeheader()
                writer.writerows(sorted(rows, key=lambda x: x['date']))
            print(f"  Created historical/{year}.csv ({len(rows)} rows)")

except Exception as e:
    print(f"  Error: {e}")
PYEOF
done

# Process CoinGecko data as backup
for file in "$DOWNLOADS"/coingecko-*-api.json; do
    [ -f "$file" ] || continue

    COIN=$(basename "$file" | sed 's/coingecko-\(.*\)-api.json/\1/')
    COIN_DIR="$OUTPUT/$COIN"
    mkdir -p "$COIN_DIR"

    echo "Processing CoinGecko: $COIN"

    python3 << PYEOF
import json

try:
    data = json.load(open('$file', 'r'))

    # Extract useful fields
    gecko_data = {
        'coingecko_id': data.get('id'),
        'coingecko_rank': data.get('coingecko_rank'),
        'genesis_date': data.get('genesis_date'),
        'sentiment_up': data.get('sentiment_votes_up_percentage'),
        'sentiment_down': data.get('sentiment_votes_down_percentage'),
        'developer_data': data.get('developer_data', {}),
        'community_data': data.get('community_data', {}),
    }

    with open('$COIN_DIR/coingecko.json', 'w') as f:
        json.dump(gecko_data, f, indent=2)
    print(f"  Created coingecko.json")

except Exception as e:
    print(f"  Error: {e}")
PYEOF
done

# Process market/exchange data
for file in "$DOWNLOADS"/cmc-*-api-markets.json; do
    [ -f "$file" ] || continue

    COIN=$(basename "$file" | sed 's/cmc-\(.*\)-api-markets.json/\1/')
    COIN_DIR="$OUTPUT/$COIN"
    mkdir -p "$COIN_DIR"

    echo "Processing markets: $COIN"

    python3 << PYEOF
import json

try:
    data = json.load(open('$file', 'r'))

    if 'data' in data and 'marketPairs' in data['data']:
        pairs = data['data']['marketPairs']

        markets = []
        for pair in pairs[:50]:  # Top 50 markets
            markets.append({
                'exchange': pair.get('exchangeName'),
                'pair': pair.get('marketPair'),
                'price': pair.get('price'),
                'volume_24h': pair.get('volumeUsd'),
                'type': pair.get('marketType'),
            })

        with open('$COIN_DIR/markets.json', 'w') as f:
            json.dump(markets, f, indent=2)

        # Add to INDEX.md
        with open('$COIN_DIR/INDEX.md', 'a') as f:
            f.write(f"\n## Markets (Top 10)\n\n")
            f.write(f"| Exchange | Pair | Volume 24h |\n")
            f.write(f"|----------|------|------------|\n")
            for m in markets[:10]:
                vol = m.get('volume_24h', 0)
                vol_str = f"${vol:,.0f}" if vol else "N/A"
                f.write(f"| {m['exchange']} | {m['pair']} | {vol_str} |\n")

        print(f"  Created markets.json ({len(markets)} pairs)")

except Exception as e:
    print(f"  Error: {e}")
PYEOF
done

echo ""
echo "=== Processing Complete ==="
echo "Output: $OUTPUT/"
