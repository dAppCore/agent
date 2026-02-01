#!/usr/bin/env bash
# Hook: collect-whitepaper.sh
# Called when a whitepaper URL is detected during collection
# Usage: ./collect-whitepaper.sh <URL> [destination-folder]

set -e

URL="$1"
DEST="${2:-./whitepapers}"

if [ -z "$URL" ]; then
    echo "Usage: $0 <url> [destination]" >&2
    exit 1
fi

# Detect paper type from URL
detect_category() {
    local url="$1"
    case "$url" in
        *cryptonote*) echo "cryptonote" ;;
        *iacr.org*|*eprint*) echo "research" ;;
        *arxiv.org*) echo "research" ;;
        *monero*|*getmonero*) echo "research" ;;
        *lethean*|*lthn*) echo "lethean" ;;
        *) echo "uncategorized" ;;
    esac
}

# Generate safe filename from URL
safe_filename() {
    local url="$1"
    basename "$url" | sed 's/[^a-zA-Z0-9._-]/-/g'
}

CATEGORY=$(detect_category "$URL")
FILENAME=$(safe_filename "$URL")
TARGET_DIR="$DEST/$CATEGORY"
TARGET_FILE="$TARGET_DIR/$FILENAME"

mkdir -p "$TARGET_DIR"

# Check if already collected
if [ -f "$TARGET_FILE" ]; then
    echo "Already collected: $TARGET_FILE"
    exit 0
fi

echo "Collecting whitepaper:"
echo "  URL: $URL"
echo "  Category: $CATEGORY"
echo "  Destination: $TARGET_FILE"

# Create job entry for proxy collection
echo "$URL|$FILENAME|whitepaper|category=$CATEGORY" >> "$DEST/.pending-jobs.txt"

echo "Job queued: $DEST/.pending-jobs.txt"
echo ""
echo "To collect immediately (if you have direct access):"
echo "  curl -L -o '$TARGET_FILE' '$URL'"
