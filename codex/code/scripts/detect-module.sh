#!/bin/bash
#
# Detects the current module and sets environment variables for other tools.
# Intended to be run once per session via a hook.

# --- Detection Logic ---
MODULE_NAME=""
MODULE_TYPE="unknown"

# 1. Check for composer.json (PHP)
if [ -f "composer.json" ]; then
    MODULE_TYPE="php"
    # Use jq, but check if it is installed first
    if command -v jq >/dev/null 2>&1; then
        MODULE_NAME=$(jq -r ".name // empty" composer.json)
    fi
fi

# 2. Check for go.mod (Go)
if [ -f "go.mod" ]; then
    MODULE_TYPE="go"
    MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')
fi

# 3. If name is still empty, try git remote
if [ -z "$MODULE_NAME" ] || [ "$MODULE_NAME" = "unknown" ]; then
    if git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
        GIT_REMOTE=$(git remote get-url origin 2>/dev/null)
        if [ -n "$GIT_REMOTE" ]; then
            MODULE_NAME=$(basename "$GIT_REMOTE" .git)
        fi
    fi
fi

# 4. As a last resort, use the current directory name
if [ -z "$MODULE_NAME" ] || [ "$MODULE_NAME" = "unknown" ]; then
    MODULE_NAME=$(basename "$PWD")
fi


# --- Store Context ---
# Create a file with the context variables to be sourced by other scripts.
mkdir -p .claude-plugin/.tmp
CONTEXT_FILE=".claude-plugin/.tmp/module_context.sh"

echo "export CLAUDE_CURRENT_MODULE=\"$MODULE_NAME\"" > "$CONTEXT_FILE"
echo "export CLAUDE_MODULE_TYPE=\"$MODULE_TYPE\"" >> "$CONTEXT_FILE"

# --- User-facing Message ---
# Print a confirmation message to stderr.
echo "Workspace context loaded: Module='$MODULE_NAME', Type='$MODULE_TYPE'" >&2
