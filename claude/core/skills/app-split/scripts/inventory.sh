#!/usr/bin/env bash
# inventory.sh — List all modules and their domain bindings in a CorePHP app.
# Usage: ./inventory.sh /path/to/app
set -euo pipefail

APP_DIR="${1:-.}"

echo "=== Website Modules ==="
echo ""
for boot in "$APP_DIR"/app/Website/*/Boot.php; do
    [ -f "$boot" ] || continue
    mod=$(basename "$(dirname "$boot")")
    # Extract domain patterns from $domains array
    domains=$(grep -E "'/\^" "$boot" 2>/dev/null | sed "s/.*'\(.*\)'.*/\1/" | tr '\n' ' ' || echo "(no domain pattern)")
    # Extract event class names from $listens array
    listens=$(grep '::class' "$boot" 2>/dev/null | grep -oE '[A-Za-z]+::class' | sed 's/::class//' | tr '\n' ', ' | sed 's/,$//' || echo "none")
    printf "%-15s domains: %s\n" "$mod" "$domains"
    printf "%-15s listens: %s\n" "" "$listens"
    echo ""
done

echo "=== Mod Modules ==="
echo ""
for boot in "$APP_DIR"/app/Mod/*/Boot.php; do
    [ -f "$boot" ] || continue
    mod=$(basename "$(dirname "$boot")")
    listens=$(grep '::class' "$boot" 2>/dev/null | grep -oE '[A-Za-z]+::class' | sed 's/::class//' | tr '\n' ', ' | sed 's/,$//' || echo "none")
    printf "%-15s listens: %s\n" "$mod" "$listens"
done

echo ""
echo "=== Service Providers ==="
echo ""
for boot in "$APP_DIR"/app/Service/*/Boot.php; do
    [ -f "$boot" ] || continue
    mod=$(basename "$(dirname "$boot")")
    code=$(grep -oE "'code'\s*=>\s*'[^']+'" "$boot" 2>/dev/null | head -1 || echo "")
    printf "%-15s %s\n" "$mod" "$code"
done

echo ""
echo "=== Boot.php Provider List ==="
grep '::class' "$APP_DIR/app/Boot.php" 2>/dev/null | grep -v '//' | sed 's/^[[:space:]]*/  /' | sed 's/,$//'
