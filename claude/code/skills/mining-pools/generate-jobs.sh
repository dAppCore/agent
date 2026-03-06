#!/usr/bin/env bash
# Generate mining pool collection jobs
# Usage: ./generate-jobs.sh <coin> [--url=URL] [--all]

set -e

COIN=""
POOL_URL=""
ALL_POOLS=0

# Known pools registry
declare -A POOLS_LETHEAN=(
    ["herominers"]="https://lethean.herominers.com"
    ["gntl"]="https://lthn.pool.gntl.uk"
)

declare -A POOLS_MONERO=(
    ["supportxmr"]="https://supportxmr.com"
    ["nanopool"]="https://xmr.nanopool.org"
    ["hashvault"]="https://monero.hashvault.pro"
)

declare -A POOLS_WOWNERO=(
    ["herominers"]="https://wownero.herominers.com"
)

# Parse args
for arg in "$@"; do
    case "$arg" in
        --url=*) POOL_URL="${arg#*=}" ;;
        --all) ALL_POOLS=1 ;;
        --*) ;;
        *) COIN="$arg" ;;
    esac
done

emit_pool_jobs() {
    local pool_name="$1"
    local pool_url="$2"
    local coin="$3"

    local slug="${coin}-${pool_name}"

    echo "# === ${pool_name} (${coin}) ==="

    # Common nodejs-pool API endpoints
    echo "${pool_url}/api/stats|pool-${slug}-stats.json|pool-api|coin=$coin,pool=$pool_name"
    echo "${pool_url}/api/pool/blocks|pool-${slug}-blocks.json|pool-api|coin=$coin,pool=$pool_name"
    echo "${pool_url}/api/pool/payments|pool-${slug}-payments.json|pool-api|coin=$coin,pool=$pool_name"
    echo "${pool_url}/api/network/stats|pool-${slug}-network.json|pool-api|coin=$coin,pool=$pool_name"
    echo "${pool_url}/api/config|pool-${slug}-config.json|pool-api|coin=$coin,pool=$pool_name"

    # Web pages
    echo "${pool_url}/|pool-${slug}-home.html|pool-web|coin=$coin,pool=$pool_name"
    echo "${pool_url}/#/blocks|pool-${slug}-blocks-page.html|pool-web|coin=$coin,pool=$pool_name"

    echo "#"
}

echo "# Mining Pool Jobs - $(date +%Y-%m-%d)"
echo "# Format: URL|FILENAME|TYPE|METADATA"
echo "#"

if [ "$ALL_POOLS" = "1" ]; then
    for pool in "${!POOLS_LETHEAN[@]}"; do
        emit_pool_jobs "$pool" "${POOLS_LETHEAN[$pool]}" "lethean"
    done
    for pool in "${!POOLS_MONERO[@]}"; do
        emit_pool_jobs "$pool" "${POOLS_MONERO[$pool]}" "monero"
    done
    for pool in "${!POOLS_WOWNERO[@]}"; do
        emit_pool_jobs "$pool" "${POOLS_WOWNERO[$pool]}" "wownero"
    done
elif [ -n "$POOL_URL" ]; then
    pool_name=$(echo "$POOL_URL" | sed 's|.*://||; s|/.*||; s|\..*||')
    emit_pool_jobs "$pool_name" "$POOL_URL" "${COIN:-unknown}"
elif [ -n "$COIN" ]; then
    case "$COIN" in
        lethean|lthn)
            for pool in "${!POOLS_LETHEAN[@]}"; do
                emit_pool_jobs "$pool" "${POOLS_LETHEAN[$pool]}" "lethean"
            done
            ;;
        monero|xmr)
            for pool in "${!POOLS_MONERO[@]}"; do
                emit_pool_jobs "$pool" "${POOLS_MONERO[$pool]}" "monero"
            done
            ;;
        wownero|wow)
            for pool in "${!POOLS_WOWNERO[@]}"; do
                emit_pool_jobs "$pool" "${POOLS_WOWNERO[$pool]}" "wownero"
            done
            ;;
        *)
            echo "# Unknown coin: $COIN" >&2
            echo "# Use --url= to specify pool URL" >&2
            exit 1
            ;;
    esac
else
    echo "Usage: $0 <coin> [--url=URL] [--all]" >&2
    echo "" >&2
    echo "Known coins: lethean, monero, wownero" >&2
    exit 1
fi
