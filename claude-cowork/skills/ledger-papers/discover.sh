#!/usr/bin/env bash
# Discover CryptoNote extension papers
# Usage: ./discover.sh [--all] [--category=NAME] [--project=NAME] [--topic=NAME]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY="$SCRIPT_DIR/registry.json"

# Check for jq
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required" >&2
    exit 1
fi

CATEGORY=""
PROJECT=""
TOPIC=""
ALL=0

# Parse args
for arg in "$@"; do
    case "$arg" in
        --all) ALL=1 ;;
        --category=*) CATEGORY="${arg#*=}" ;;
        --project=*) PROJECT="${arg#*=}" ;;
        --topic=*) TOPIC="${arg#*=}" ;;
        --search-iacr) SEARCH_IACR=1 ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --all              All known papers"
            echo "  --category=NAME    Filter by category (mrl, iacr, projects, attacks)"
            echo "  --project=NAME     Filter by project (monero, haven, masari, etc)"
            echo "  --topic=NAME       Filter by topic (bulletproofs, ringct, etc)"
            echo "  --search-iacr      Generate IACR search jobs"
            echo ""
            echo "Categories:"
            jq -r '.categories | keys[]' "$REGISTRY"
            exit 0
            ;;
    esac
done

echo "# Ledger Papers Archive - $(date +%Y-%m-%d)"
echo "# Format: URL|FILENAME|TYPE|METADATA"
echo "#"

emit_paper() {
    local url="$1"
    local id="$2"
    local category="$3"
    local title="$4"

    local filename="${id}.pdf"
    local metadata="category=$category,title=$title"

    echo "${url}|${filename}|paper|${metadata}"
}

# Process categories
process_category() {
    local cat_name="$1"

    echo "# === $cat_name ==="

    # Get papers in category
    local papers
    papers=$(jq -c ".categories[\"$cat_name\"].papers[]?" "$REGISTRY" 2>/dev/null)

    echo "$papers" | while read -r paper; do
        [ -z "$paper" ] && continue

        local id title url urls
        id=$(echo "$paper" | jq -r '.id')
        title=$(echo "$paper" | jq -r '.title // "Unknown"')

        # Check topic filter
        if [ -n "$TOPIC" ]; then
            if ! echo "$paper" | jq -e ".topics[]? | select(. == \"$TOPIC\")" > /dev/null 2>&1; then
                continue
            fi
        fi

        # Check project filter
        if [ -n "$PROJECT" ]; then
            local paper_project
            paper_project=$(echo "$paper" | jq -r '.project // ""')
            if [ "$paper_project" != "$PROJECT" ]; then
                continue
            fi
        fi

        # Get URL (single or first from array)
        url=$(echo "$paper" | jq -r '.url // .urls[0] // ""')

        if [ -n "$url" ]; then
            emit_paper "$url" "$id" "$cat_name" "$title"
        fi

        # Also emit alternate URLs for wayback
        urls=$(echo "$paper" | jq -r '.urls[]? // empty' 2>/dev/null)
        echo "$urls" | while read -r alt_url; do
            [ -z "$alt_url" ] && continue
            [ "$alt_url" = "$url" ] && continue
            echo "# alt: $alt_url"
        done
    done

    echo "#"
}

# Main logic
if [ "$ALL" = "1" ] || [ -z "$CATEGORY" ]; then
    # All categories - dynamically from registry
    jq -r '.categories | keys[]' "$REGISTRY" | while read -r cat; do
        process_category "$cat"
    done
else
    # Single category
    process_category "$CATEGORY"
fi

# IACR search jobs
if [ "$SEARCH_IACR" = "1" ]; then
    echo "# === IACR Search Jobs ==="
    jq -r '.search_patterns.iacr[]' "$REGISTRY" | while read -r term; do
        encoded=$(echo "$term" | sed 's/ /+/g')
        echo "https://eprint.iacr.org/search?q=${encoded}|iacr-search-${encoded}.html|search|source=iacr,term=$term"
    done
fi
