#!/usr/bin/env bash
# Generate block explorer collection jobs
# Usage: ./generate-jobs.sh <coin> [--blocks=N] [--sample=daily|weekly|monthly]

set -e

COIN=""
EXPLORER_URL=""
SAMPLE="monthly"
BLOCK_COUNT=100

# Known explorers
declare -A EXPLORERS=(
    ["lethean"]="https://explorer.lethean.io"
    ["monero"]="https://xmrchain.net"
    ["haven"]="https://explorer.havenprotocol.org"
    ["karbo"]="https://explorer.karbo.io"
    ["wownero"]="https://explore.wownero.com"
    ["dero"]="https://explorer.dero.io"
    ["masari"]="https://explorer.getmasari.org"
    ["turtlecoin"]="https://explorer.turtlecoin.lol"
    ["conceal"]="https://explorer.conceal.network"
)

# Parse args
for arg in "$@"; do
    case "$arg" in
        --url=*) EXPLORER_URL="${arg#*=}" ;;
        --blocks=*) BLOCK_COUNT="${arg#*=}" ;;
        --sample=*) SAMPLE="${arg#*=}" ;;
        --*) ;;
        *) COIN="$arg" ;;
    esac
done

if [ -z "$COIN" ] && [ -z "$EXPLORER_URL" ]; then
    echo "Usage: $0 <coin> [--url=URL] [--blocks=N] [--sample=daily|weekly|monthly]" >&2
    echo "" >&2
    echo "Known coins: ${!EXPLORERS[*]}" >&2
    exit 1
fi

# Get explorer URL
if [ -z "$EXPLORER_URL" ]; then
    EXPLORER_URL="${EXPLORERS[$COIN]}"
    if [ -z "$EXPLORER_URL" ]; then
        echo "# ERROR: Unknown coin '$COIN'. Use --url= to specify explorer." >&2
        exit 1
    fi
fi

SLUG=$(echo "$COIN" | tr '[:upper:]' '[:lower:]')

echo "# Block Explorer Jobs for $COIN"
echo "# Explorer: $EXPLORER_URL"
echo "# Sample: $SAMPLE"
echo "# Format: URL|FILENAME|TYPE|METADATA"
echo "#"

# Core API endpoints
echo "# === Core Data ==="
echo "${EXPLORER_URL}/api/info|explorer-${SLUG}-info.json|explorer-api|coin=$SLUG,type=info"
echo "${EXPLORER_URL}/api/emission|explorer-${SLUG}-emission.json|explorer-api|coin=$SLUG,type=emission"
echo "${EXPLORER_URL}/api/supply|explorer-${SLUG}-supply.json|explorer-api|coin=$SLUG,type=supply"
echo "${EXPLORER_URL}/api/mempool|explorer-${SLUG}-mempool.json|explorer-api|coin=$SLUG,type=mempool"

# Genesis block
echo "#"
echo "# === Genesis Block ==="
echo "${EXPLORER_URL}/api/block/0|explorer-${SLUG}-block-0.json|explorer-api|coin=$SLUG,block=0"
echo "${EXPLORER_URL}/api/block/1|explorer-${SLUG}-block-1.json|explorer-api|coin=$SLUG,block=1"

# Milestone blocks (if we know the heights)
echo "#"
echo "# === Milestone Blocks ==="
for height in 10000 50000 100000 500000 1000000 2000000; do
    echo "${EXPLORER_URL}/api/block/${height}|explorer-${SLUG}-block-${height}.json|explorer-api|coin=$SLUG,block=$height"
done

# Sample blocks by time
echo "#"
echo "# === Sampled Blocks (estimate heights) ==="
case "$SAMPLE" in
    daily)
        # ~720 blocks/day for 2-min blocks
        STEP=720
        ;;
    weekly)
        STEP=5040
        ;;
    monthly)
        STEP=21600
        ;;
esac

for ((i=0; i<BLOCK_COUNT; i++)); do
    height=$((i * STEP))
    echo "${EXPLORER_URL}/api/block/${height}|explorer-${SLUG}-sample-${height}.json|explorer-api|coin=$SLUG,block=$height,sample=$SAMPLE"
done

# Web pages (for scraping if API fails)
echo "#"
echo "# === Web Pages (backup) ==="
echo "${EXPLORER_URL}/|explorer-${SLUG}-home.html|explorer-web|coin=$SLUG"
echo "${EXPLORER_URL}/blocks|explorer-${SLUG}-blocks.html|explorer-web|coin=$SLUG"
echo "${EXPLORER_URL}/stats|explorer-${SLUG}-stats.html|explorer-web|coin=$SLUG"
