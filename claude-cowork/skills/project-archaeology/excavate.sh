#!/bin/bash
# Project Archaeology - Deep excavation of abandoned CryptoNote projects
# Usage: ./excavate.sh <project-name> [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILLS_DIR="$(dirname "$SCRIPT_DIR")"
REGISTRY="$SKILLS_DIR/cryptonote-discovery/registry.json"
OUTPUT_DIR="$SCRIPT_DIR/digs"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
SCAN_ONLY=false
RESUME=false
ONLY_COLLECTORS=""

usage() {
    echo "Usage: $0 <project-name> [options]"
    echo ""
    echo "Options:"
    echo "  --scan-only       Check what's accessible without downloading"
    echo "  --resume          Resume interrupted excavation"
    echo "  --only=a,b,c      Run specific collectors only"
    echo "  --help            Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 masari                    # Full excavation"
    echo "  $0 masari --scan-only        # Quick accessibility check"
    echo "  $0 masari --only=github,btt  # GitHub and BitcoinTalk only"
    exit 1
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[!]${NC} $1"
}

error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Get project data from registry
get_project() {
    local name="$1"
    jq -r --arg n "$name" '.projects[] | select(.name | ascii_downcase == ($n | ascii_downcase))' "$REGISTRY"
}

# Check if a collector should run
should_run() {
    local collector="$1"
    if [ -z "$ONLY_COLLECTORS" ]; then
        return 0
    fi
    echo "$ONLY_COLLECTORS" | grep -q "$collector"
}

# Scan a URL to check if accessible
check_url() {
    local url="$1"
    local status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" 2>/dev/null || echo "000")
    if [ "$status" = "200" ] || [ "$status" = "301" ] || [ "$status" = "302" ]; then
        return 0
    fi
    return 1
}

# Main excavation function
excavate() {
    local project_name="$1"
    local project=$(get_project "$project_name")

    if [ -z "$project" ] || [ "$project" = "null" ]; then
        error "Project '$project_name' not found in registry"
        echo "Add it to: $REGISTRY"
        exit 1
    fi

    # Extract project data
    local name=$(echo "$project" | jq -r '.name')
    local symbol=$(echo "$project" | jq -r '.symbol')
    local status=$(echo "$project" | jq -r '.status')
    local github_orgs=$(echo "$project" | jq -r '.github[]?' 2>/dev/null)
    local btt_topic=$(echo "$project" | jq -r '.bitcointalk // empty')
    local website=$(echo "$project" | jq -r '.website // empty')
    local explorer=$(echo "$project" | jq -r '.explorer // empty')
    local cmc=$(echo "$project" | jq -r '.cmc // empty')

    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  PROJECT ARCHAEOLOGY: ${name} (${symbol})${NC}"
    echo -e "${BLUE}  Status: ${status}${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo ""

    # Create output directory
    local dig_dir="$OUTPUT_DIR/$project_name"
    mkdir -p "$dig_dir"/{github,releases,bitcointalk,website,explorer,market,papers,community}

    # Start excavation log
    local log_file="$dig_dir/EXCAVATION.md"
    echo "# Excavation Log: $name ($symbol)" > "$log_file"
    echo "" >> "$log_file"
    echo "**Started:** $(date)" >> "$log_file"
    echo "**Status at dig time:** $status" >> "$log_file"
    echo "" >> "$log_file"
    echo "---" >> "$log_file"
    echo "" >> "$log_file"

    # Phase 1: GitHub (highest priority - often deleted first)
    if should_run "github"; then
        echo "## GitHub Repositories" >> "$log_file"
        echo "" >> "$log_file"

        for org in $github_orgs; do
            log "Checking GitHub org: $org"

            if $SCAN_ONLY; then
                if check_url "https://github.com/$org"; then
                    success "GitHub org accessible: $org"
                    echo "- [x] \`$org\` - accessible" >> "$log_file"
                else
                    warn "GitHub org NOT accessible: $org"
                    echo "- [ ] \`$org\` - NOT accessible" >> "$log_file"
                fi
            else
                log "Running github-history collector on $org..."
                # Would call: $SKILLS_DIR/github-history/collect.sh "https://github.com/$org" --org
                echo "- Collected: \`$org\`" >> "$log_file"
            fi
        done
        echo "" >> "$log_file"
    fi

    # Phase 2: BitcoinTalk
    if should_run "btt" || should_run "bitcointalk"; then
        echo "## BitcoinTalk Thread" >> "$log_file"
        echo "" >> "$log_file"

        if [ -n "$btt_topic" ]; then
            local btt_url="https://bitcointalk.org/index.php?topic=$btt_topic"
            log "Checking BitcoinTalk topic: $btt_topic"

            if $SCAN_ONLY; then
                if check_url "$btt_url"; then
                    success "BitcoinTalk thread accessible"
                    echo "- [x] Topic $btt_topic - accessible" >> "$log_file"
                else
                    warn "BitcoinTalk thread NOT accessible"
                    echo "- [ ] Topic $btt_topic - NOT accessible" >> "$log_file"
                fi
            else
                log "Running bitcointalk collector..."
                # Would call: $SKILLS_DIR/bitcointalk/collect.sh "$btt_topic"
                echo "- Collected: Topic $btt_topic" >> "$log_file"
            fi
        else
            warn "No BitcoinTalk topic ID in registry"
            echo "- [ ] No topic ID recorded" >> "$log_file"
        fi
        echo "" >> "$log_file"
    fi

    # Phase 3: Website via Wayback
    if should_run "wayback" || should_run "website"; then
        echo "## Website (Wayback Machine)" >> "$log_file"
        echo "" >> "$log_file"

        if [ -n "$website" ]; then
            log "Checking Wayback Machine for: $website"
            local wayback_api="https://archive.org/wayback/available?url=$website"

            if $SCAN_ONLY; then
                local wayback_check=$(curl -s "$wayback_api" | jq -r '.archived_snapshots.closest.available // "false"')
                if [ "$wayback_check" = "true" ]; then
                    success "Wayback snapshots available for $website"
                    echo "- [x] \`$website\` - snapshots available" >> "$log_file"
                else
                    warn "No Wayback snapshots for $website"
                    echo "- [ ] \`$website\` - no snapshots" >> "$log_file"
                fi
            else
                log "Running wayback collector..."
                # Would call: $SKILLS_DIR/job-collector/generate-jobs.sh wayback "$website"
                echo "- Collected: \`$website\`" >> "$log_file"
            fi
        else
            warn "No website in registry"
            echo "- [ ] No website recorded" >> "$log_file"
        fi
        echo "" >> "$log_file"
    fi

    # Phase 4: Block Explorer
    if should_run "explorer"; then
        echo "## Block Explorer" >> "$log_file"
        echo "" >> "$log_file"

        if [ -n "$explorer" ]; then
            log "Checking block explorer: $explorer"

            if $SCAN_ONLY; then
                if check_url "https://$explorer"; then
                    success "Block explorer online: $explorer"
                    echo "- [x] \`$explorer\` - online" >> "$log_file"
                else
                    warn "Block explorer OFFLINE: $explorer"
                    echo "- [ ] \`$explorer\` - OFFLINE" >> "$log_file"
                fi
            else
                log "Running block-explorer collector..."
                echo "- Collected: \`$explorer\`" >> "$log_file"
            fi
        else
            warn "No explorer in registry"
            echo "- [ ] No explorer recorded" >> "$log_file"
        fi
        echo "" >> "$log_file"
    fi

    # Phase 5: Market Data (CMC)
    if should_run "cmc" || should_run "market"; then
        echo "## Market Data" >> "$log_file"
        echo "" >> "$log_file"

        if [ -n "$cmc" ]; then
            log "Checking CoinMarketCap: $cmc"

            if $SCAN_ONLY; then
                if check_url "https://coinmarketcap.com/currencies/$cmc/"; then
                    success "CMC page exists: $cmc"
                    echo "- [x] CMC: \`$cmc\` - exists" >> "$log_file"
                else
                    warn "CMC page NOT found: $cmc"
                    echo "- [ ] CMC: \`$cmc\` - not found" >> "$log_file"
                fi
            else
                log "Running coinmarketcap collector..."
                echo "- Collected: \`$cmc\`" >> "$log_file"
            fi
        else
            warn "No CMC slug in registry"
            echo "- [ ] No CMC slug recorded" >> "$log_file"
        fi
        echo "" >> "$log_file"
    fi

    # Finalize log
    echo "---" >> "$log_file"
    echo "" >> "$log_file"
    echo "**Completed:** $(date)" >> "$log_file"

    if $SCAN_ONLY; then
        echo ""
        success "Scan complete. See: $log_file"
    else
        echo ""
        success "Excavation complete. Output in: $dig_dir"
        echo ""
        log "Next steps:"
        echo "  1. Review: $log_file"
        echo "  2. Generate: $dig_dir/SALVAGE-REPORT.md"
        echo "  3. Write: $dig_dir/LESSONS.md"
    fi
}

# Parse arguments
if [ $# -lt 1 ]; then
    usage
fi

PROJECT="$1"
shift

while [ $# -gt 0 ]; do
    case "$1" in
        --scan-only)
            SCAN_ONLY=true
            ;;
        --resume)
            RESUME=true
            ;;
        --only=*)
            ONLY_COLLECTORS="${1#*=}"
            ;;
        --help)
            usage
            ;;
        *)
            error "Unknown option: $1"
            usage
            ;;
    esac
    shift
done

# Run excavation
excavate "$PROJECT"
