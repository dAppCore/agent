#!/usr/bin/env bash
# Hook dispatcher - source this in collectors
# Usage: source ./hooks/dispatch.sh

HOOKS_DIR="$(dirname "${BASH_SOURCE[0]}")"
HOOKS_JSON="$HOOKS_DIR/hooks.json"

# Dispatch a hook event
# dispatch_hook <event> <arg1> [arg2] ...
dispatch_hook() {
    local event="$1"
    shift
    local args=("$@")

    if [ ! -f "$HOOKS_JSON" ]; then
        return 0
    fi

    # Get handlers for this event (requires jq)
    if ! command -v jq &> /dev/null; then
        echo "Warning: jq not installed, hooks disabled" >&2
        return 0
    fi

    local handlers
    handlers=$(jq -r ".hooks[\"$event\"][]? | select(.enabled == true) | @json" "$HOOKS_JSON" 2>/dev/null)

    if [ -z "$handlers" ]; then
        return 0
    fi

    echo "$handlers" | while read -r handler_json; do
        local name pattern handler_script priority
        name=$(echo "$handler_json" | jq -r '.name')
        pattern=$(echo "$handler_json" | jq -r '.pattern // ""')
        handler_script=$(echo "$handler_json" | jq -r '.handler')

        # Check pattern match if pattern exists
        if [ -n "$pattern" ] && [ -n "${args[0]}" ]; then
            if ! echo "${args[0]}" | grep -qE "$pattern"; then
                continue
            fi
        fi

        # Execute handler
        local full_path="$HOOKS_DIR/$handler_script"
        if [ -x "$full_path" ]; then
            echo "[hook] $name: ${args[*]}" >&2
            "$full_path" "${args[@]}"
        elif [ -f "$full_path" ]; then
            echo "[hook] $name: ${args[*]}" >&2
            bash "$full_path" "${args[@]}"
        fi
    done
}

# Register a new hook dynamically
# register_hook <event> <name> <pattern> <handler>
register_hook() {
    local event="$1"
    local name="$2"
    local pattern="$3"
    local handler="$4"

    if ! command -v jq &> /dev/null; then
        echo "Error: jq required for hook registration" >&2
        return 1
    fi

    local new_hook
    new_hook=$(jq -n \
        --arg name "$name" \
        --arg pattern "$pattern" \
        --arg handler "$handler" \
        '{name: $name, pattern: $pattern, handler: $handler, priority: 50, enabled: true}')

    # Add to hooks.json
    jq ".hooks[\"$event\"] += [$new_hook]" "$HOOKS_JSON" > "$HOOKS_JSON.tmp" \
        && mv "$HOOKS_JSON.tmp" "$HOOKS_JSON"
}
