#!/usr/bin/env bash
# Discover all collection sources for a CryptoNote project
# Usage: ./discover.sh <project-name> | ./discover.sh --abandoned | ./discover.sh --all

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY="$SCRIPT_DIR/registry.json"

# Get project data from registry
get_project() {
    local name="$1"
    jq -r ".projects[] | select(.name | ascii_downcase == \"$(echo $name | tr '[:upper:]' '[:lower:]')\")" "$REGISTRY"
}

# List abandoned projects
list_abandoned() {
    jq -r '.projects[] | select(.status == "abandoned" or .status == "low-activity" or .status == "dead") | .name' "$REGISTRY"
}

# List all projects
list_all() {
    jq -r '.projects[].name' "$REGISTRY"
}

# Generate sources for a project
generate_sources() {
    local name="$1"
    local project=$(get_project "$name")

    if [ -z "$project" ] || [ "$project" = "null" ]; then
        echo "# ERROR: Project '$name' not found in registry" >&2
        return 1
    fi

    local symbol=$(echo "$project" | jq -r '.symbol')
    local status=$(echo "$project" | jq -r '.status')

    echo "# === $name ($symbol) ==="
    echo "# Status: $status"
    echo "#"

    # GitHub repos
    echo "# GitHub Organizations:"
    echo "$project" | jq -r '.github[]?' | while read org; do
        [ -n "$org" ] && echo "github|https://github.com/$org|$name"
    done

    # BitcoinTalk
    local btt=$(echo "$project" | jq -r '.bitcointalk // empty')
    if [ -n "$btt" ]; then
        echo "#"
        echo "# BitcoinTalk:"
        echo "bitcointalk|https://bitcointalk.org/index.php?topic=$btt.0|$name"
    fi

    # CMC/CoinGecko
    local cmc=$(echo "$project" | jq -r '.cmc // empty')
    local gecko=$(echo "$project" | jq -r '.coingecko // empty')
    echo "#"
    echo "# Market Data:"
    [ -n "$cmc" ] && echo "cmc|https://coinmarketcap.com/currencies/$cmc/|$name"
    [ -n "$gecko" ] && echo "coingecko|https://coingecko.com/en/coins/$gecko|$name"

    # Website/Explorer
    local website=$(echo "$project" | jq -r '.website // empty')
    local explorer=$(echo "$project" | jq -r '.explorer // empty')
    echo "#"
    echo "# Web Properties:"
    [ -n "$website" ] && echo "wayback|https://$website|$name"
    [ -n "$explorer" ] && echo "explorer|https://$explorer|$name"

    # Salvageable features
    local salvage=$(echo "$project" | jq -r '.salvageable[]?' 2>/dev/null)
    if [ -n "$salvage" ]; then
        echo "#"
        echo "# Salvageable:"
        echo "$project" | jq -r '.salvageable[]?' | while read item; do
            echo "#   - $item"
        done
    fi

    echo "#"
}

# Main
case "$1" in
    --abandoned)
        echo "# Abandoned CryptoNote Projects (Salvage Candidates)"
        echo "# Format: source|url|project"
        echo "#"
        for proj in $(list_abandoned); do
            generate_sources "$proj"
        done
        ;;
    --all)
        echo "# All CryptoNote Projects"
        echo "# Format: source|url|project"
        echo "#"
        for proj in $(list_all); do
            generate_sources "$proj"
        done
        ;;
    --list)
        list_all
        ;;
    --list-abandoned)
        list_abandoned
        ;;
    "")
        echo "Usage: $0 <project-name> | --abandoned | --all | --list" >&2
        echo "" >&2
        echo "Examples:" >&2
        echo "  $0 lethean          # Sources for Lethean" >&2
        echo "  $0 monero           # Sources for Monero" >&2
        echo "  $0 --abandoned      # All abandoned projects" >&2
        echo "  $0 --all            # Everything" >&2
        echo "  $0 --list           # Just list project names" >&2
        exit 1
        ;;
    *)
        generate_sources "$1"
        ;;
esac
