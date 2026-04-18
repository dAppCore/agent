#!/usr/bin/env bash
# domain-scan.sh — Find all host.uk.com / host.test references in a CorePHP app.
# Usage: ./domain-scan.sh /path/to/app [domain_pattern]
# Default domain pattern: host\.uk\.com|host\.test

APP_DIR="${1:-.}"
PATTERN="${2:-host\.uk\.com|host\.test}"

echo "=== Domain Reference Scan ==="
echo "Directory: $APP_DIR"
echo "Pattern:   $PATTERN"
echo ""

echo "--- By Directory ---"
for dir in app config database public resources routes; do
    [ -d "$APP_DIR/$dir" ] || continue
    count=$(grep -rlE "$PATTERN" "$APP_DIR/$dir" 2>/dev/null | wc -l | tr -d ' ')
    [ "$count" -gt 0 ] && printf "%-20s %s files\n" "$dir/" "$count"
done

# Root files
echo ""
echo "--- Root Files ---"
for f in .env.example vite.config.js CLAUDE.md robots.txt Makefile playwright.config.ts; do
    [ -f "$APP_DIR/$f" ] && grep -qE "$PATTERN" "$APP_DIR/$f" 2>/dev/null && printf "  %s\n" "$f"
done

echo ""
echo "--- Critical Files (app code, not docs) ---"
grep -rnE "$PATTERN" \
    "$APP_DIR/app/" \
    "$APP_DIR/config/" \
    "$APP_DIR/database/seeders/" \
    "$APP_DIR/public/js/" \
    "$APP_DIR/public/errors/" \
    "$APP_DIR/public/robots.txt" \
    "$APP_DIR/vite.config.js" \
    "$APP_DIR/.env.example" \
    2>/dev/null \
    | grep -v '/docs/' \
    | grep -v '/plans/' \
    | grep -v 'node_modules' \
    | grep -v 'vendor/' \
    || echo "(none found)"

echo ""
echo "--- Shared Infra References (review — may be intentional) ---"
grep -rnE 'analytics\.host\.uk\.com|cdn\.host\.uk\.com' \
    "$APP_DIR/app/" \
    "$APP_DIR/config/" \
    2>/dev/null \
    || echo "(none found)"

exit 0
