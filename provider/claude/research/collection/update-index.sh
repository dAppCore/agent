#!/usr/bin/env bash
# Hook: update-index.sh
# Called after collection completes to update indexes

WHITEPAPERS_DIR="${1:-./whitepapers}"

echo "[update-index] Updating whitepaper index..."

# Count papers in each category
for category in cryptonote lethean research uncategorized; do
    dir="$WHITEPAPERS_DIR/$category"
    if [ -d "$dir" ]; then
        count=$(find "$dir" -name "*.pdf" 2>/dev/null | wc -l | tr -d ' ')
        echo "  $category: $count papers"
    fi
done

# Update INDEX.md with collected papers
INDEX="$WHITEPAPERS_DIR/INDEX.md"
if [ -f "$INDEX" ]; then
    # Add collected papers section if not exists
    if ! grep -q "## Recently Collected" "$INDEX"; then
        echo "" >> "$INDEX"
        echo "## Recently Collected" >> "$INDEX"
        echo "" >> "$INDEX"
        echo "_Last updated: $(date +%Y-%m-%d)_" >> "$INDEX"
        echo "" >> "$INDEX"
    fi
fi

# Process pending jobs
PENDING="$WHITEPAPERS_DIR/.pending-jobs.txt"
if [ -f "$PENDING" ]; then
    count=$(wc -l < "$PENDING" | tr -d ' ')
    echo "[update-index] $count papers queued for collection"
fi

echo "[update-index] Done"
