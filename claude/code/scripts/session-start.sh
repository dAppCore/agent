#!/bin/bash
# Session start: Load OpenBrain context + recent scratchpad
#
# 1. Query OpenBrain for project-relevant memories
# 2. Read local scratchpad if recent (<3h)
# 3. Output to stdout → injected into Claude's context

BRAIN_URL="${CORE_BRAIN_URL:-https://api.lthn.sh}"
BRAIN_KEY="${CORE_BRAIN_KEY:-}"
BRAIN_KEY_FILE="${HOME}/.claude/brain.key"
STATE_FILE="${HOME}/.claude/sessions/scratchpad.md"
THREE_HOURS=10800

# Load API key from file if not in env
if [[ -z "$BRAIN_KEY" && -f "$BRAIN_KEY_FILE" ]]; then
    BRAIN_KEY=$(cat "$BRAIN_KEY_FILE" 2>/dev/null | tr -d '[:space:]')
fi

# --- OpenBrain Recall ---
if [[ -n "$BRAIN_KEY" ]]; then
    # Detect project from CWD
    PROJECT=""
    CWD=$(pwd)
    case "$CWD" in
        */core/go-*)   PROJECT=$(basename "$CWD" | sed 's/^go-//') ;;
        */core/php-*)  PROJECT=$(basename "$CWD" | sed 's/^php-//') ;;
        */core/*)      PROJECT=$(basename "$CWD") ;;
        */host-uk/*)   PROJECT=$(basename "$CWD") ;;
        */lthn/*)      PROJECT=$(basename "$CWD") ;;
        */snider/*)    PROJECT=$(basename "$CWD") ;;
    esac

    echo "[SessionStart] OpenBrain: querying memories..." >&2

    # 1. Recent session summaries (what did we do recently?)
    RECENT=$(curl -s --max-time 5 "${BRAIN_URL}/v1/brain/recall" \
        -X POST \
        -H 'Content-Type: application/json' \
        -H 'Accept: application/json' \
        -H "Authorization: Bearer ${BRAIN_KEY}" \
        -d "{\"query\": \"session summary milestone recent work completed\", \"top_k\": 3, \"agent_id\": \"cladius\"}" 2>/dev/null)

    # 2. Project-specific context (if we're in a project dir)
    PROJECT_CTX=""
    if [[ -n "$PROJECT" ]]; then
        PROJECT_CTX=$(curl -s --max-time 5 "${BRAIN_URL}/v1/brain/recall" \
            -X POST \
            -H 'Content-Type: application/json' \
            -H 'Accept: application/json' \
            -H "Authorization: Bearer ${BRAIN_KEY}" \
            -d "{\"query\": \"architecture decisions conventions for ${PROJECT}\", \"top_k\": 3, \"agent_id\": \"cladius\", \"project\": \"${PROJECT}\"}" 2>/dev/null)
    fi

    # Output to stdout (injected into context)
    RECENT_COUNT=$(echo "$RECENT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d.get('memories',[])))" 2>/dev/null || echo "0")

    if [[ "$RECENT_COUNT" -gt 0 ]]; then
        echo ""
        echo "## OpenBrain — Recent Activity"
        echo ""
        echo "$RECENT" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for m in data.get('memories', []):
    t = m.get('type', '?')
    p = m.get('project', '?')
    content = m.get('content', '')[:300]
    print(f'**[{t}]** ({p}): {content}')
    print()
" 2>/dev/null
    fi

    if [[ -n "$PROJECT" && -n "$PROJECT_CTX" ]]; then
        PROJECT_COUNT=$(echo "$PROJECT_CTX" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d.get('memories',[])))" 2>/dev/null || echo "0")
        if [[ "$PROJECT_COUNT" -gt 0 ]]; then
            echo ""
            echo "## OpenBrain — ${PROJECT} Context"
            echo ""
            echo "$PROJECT_CTX" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for m in data.get('memories', []):
    t = m.get('type', '?')
    content = m.get('content', '')[:300]
    print(f'**[{t}]**: {content}')
    print()
" 2>/dev/null
        fi
    fi

    echo "[SessionStart] OpenBrain: ${RECENT_COUNT} recent + ${PROJECT_COUNT:-0} project memories loaded" >&2
else
    echo "[SessionStart] OpenBrain: no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)" >&2
fi

# --- Local Scratchpad ---
if [[ -f "$STATE_FILE" ]]; then
    FILE_TS=$(grep -E '^timestamp:' "$STATE_FILE" 2>/dev/null | cut -d' ' -f2)
    NOW=$(date '+%s')

    if [[ -n "$FILE_TS" ]]; then
        AGE=$((NOW - FILE_TS))
        if [[ $AGE -lt $THREE_HOURS ]]; then
            echo "[SessionStart] Scratchpad: $(($AGE / 60)) min old" >&2
            echo ""
            echo "## Recent Scratchpad ($(($AGE / 60)) min ago)"
            echo ""
            cat "$STATE_FILE"
        else
            rm -f "$STATE_FILE"
            echo "[SessionStart] Scratchpad: >3h old, cleared" >&2
        fi
    else
        rm -f "$STATE_FILE"
    fi
fi

exit 0
