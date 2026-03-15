#!/bin/bash
# Session start: Load OpenBrain context + recent scratchpad
#
# 1. Query OpenBrain for session history + roadmap + project context
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

# Helper: query OpenBrain and return JSON
recall() {
    local query="$1"
    local top_k="${2:-3}"
    local extra="${3:-}"  # optional extra JSON fields (project, type filter)

    local body="{\"query\": \"${query}\", \"top_k\": ${top_k}, \"agent_id\": \"cladius\"${extra}}"

    curl -s --max-time 8 "${BRAIN_URL}/v1/brain/recall" \
        -X POST \
        -H 'Content-Type: application/json' \
        -H 'Accept: application/json' \
        -H "Authorization: Bearer ${BRAIN_KEY}" \
        -d "$body" 2>/dev/null
}

# Helper: format memories as markdown (truncate to N chars)
format_memories() {
    local max_len="${1:-500}"
    python3 -c "
import json, sys
data = json.load(sys.stdin)
for m in data.get('memories', []):
    t = m.get('type', '?')
    p = m.get('project', '?')
    score = m.get('score', 0)
    content = m.get('content', '')[:${max_len}]
    created = m.get('created_at', '')[:10]
    print(f'**[{t}]** ({p}, {created}, score:{score:.2f}):')
    print(f'{content}')
    print()
" 2>/dev/null
}

# Helper: count memories in JSON
count_memories() {
    python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d.get('memories',[])))" 2>/dev/null || echo "0"
}

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

    echo "[SessionStart] OpenBrain: querying memories (project: ${PROJECT:-global})..." >&2

    # 1. Session history — what happened recently?
    #    Session summaries are stored as type:decision with dated content
    SESSIONS=$(recall "session summary day completed built implemented fixed pushed agentic dispatch IDE OpenBrain" 3 ", \"type\": \"decision\"")

    # 2. Design priorities + roadmap — what's planned next?
    #    Roadmap items are stored as type:plan with actionable next steps
    ROADMAP=$(recall "next session priorities design plans wire dispatch conventions pending work roadmap" 3 ", \"type\": \"plan\"")

    # 3. Project-specific context (if we're in a project dir)
    PROJECT_CTX=""
    if [[ -n "$PROJECT" ]]; then
        PROJECT_CTX=$(recall "architecture design decisions dependencies conventions implementation status for ${PROJECT}" 3 ", \"project\": \"${PROJECT}\", \"type\": \"convention\"")
    fi

    # --- Output to stdout ---
    SESSION_COUNT=$(echo "$SESSIONS" | count_memories)
    ROADMAP_COUNT=$(echo "$ROADMAP" | count_memories)

    if [[ "$SESSION_COUNT" -gt 0 ]]; then
        echo ""
        echo "## OpenBrain — Recent Sessions"
        echo ""
        echo "$SESSIONS" | format_memories 600
    fi

    if [[ "$ROADMAP_COUNT" -gt 0 ]]; then
        echo ""
        echo "## OpenBrain — Roadmap & Priorities"
        echo ""
        echo "$ROADMAP" | format_memories 600
    fi

    if [[ -n "$PROJECT" && -n "$PROJECT_CTX" ]]; then
        PROJECT_COUNT=$(echo "$PROJECT_CTX" | count_memories)
        if [[ "$PROJECT_COUNT" -gt 0 ]]; then
            echo ""
            echo "## OpenBrain — ${PROJECT} Context"
            echo ""
            echo "$PROJECT_CTX" | format_memories 400
        fi
    fi

    TOTAL=$((SESSION_COUNT + ROADMAP_COUNT + ${PROJECT_COUNT:-0}))
    echo "[SessionStart] OpenBrain: ${TOTAL} memories loaded (${SESSION_COUNT} sessions, ${ROADMAP_COUNT} roadmap, ${PROJECT_COUNT:-0} project)" >&2
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
