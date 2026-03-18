#!/bin/bash
# Local agent dispatch without MCP/core-agent.
# Usage: local-dispatch.sh <repo> <task> [agent] [persona-path]
#
# Creates workspace, clones repo, runs agent, captures output.

set -euo pipefail

REPO="${1:?repo required}"
TASK="${2:?task required}"
AGENT="${3:-claude:haiku}"
PERSONA="${4:-}"

CODE_PATH="${CODE_PATH:-$HOME/Code}"
WS_ROOT="$CODE_PATH/.core/workspace"
PROMPTS="$CODE_PATH/core/agent/pkg/prompts/lib"
TIMESTAMP=$(date +%s)
WS_DIR="$WS_ROOT/${REPO}-${TIMESTAMP}"

# Parse agent:model
IFS=':' read -r AGENT_TYPE MODEL <<< "$AGENT"
MODEL="${MODEL:-haiku}"

# Map to CLI + model flag
case "$AGENT_TYPE" in
  claude)
    case "$MODEL" in
      haiku)  CLI="claude"; MODEL_FLAG="--model claude-haiku-4-5-20251001" ;;
      sonnet) CLI="claude"; MODEL_FLAG="--model claude-sonnet-4-5-20241022" ;;
      opus)   CLI="claude"; MODEL_FLAG="" ;;
      *)      CLI="claude"; MODEL_FLAG="--model $MODEL" ;;
    esac
    ;;
  gemini) CLI="gemini"; MODEL_FLAG="-p" ;;
  codex)  CLI="codex"; MODEL_FLAG="exec -p" ;;
  *)      echo "Unknown agent: $AGENT_TYPE"; exit 1 ;;
esac

# Create workspace
mkdir -p "$WS_DIR"

# Clone repo
REPO_PATH="$CODE_PATH/core/$REPO"
if [ ! -d "$REPO_PATH" ]; then
  echo "ERROR: $REPO_PATH not found"
  exit 1
fi
git clone "$REPO_PATH" "$WS_DIR/src" 2>/dev/null

# Build prompt
PROMPT="$TASK"

# Add persona if provided
if [ -n "$PERSONA" ] && [ -f "$PROMPTS/persona/$PERSONA.md" ]; then
  PERSONA_CONTENT=$(cat "$PROMPTS/persona/$PERSONA.md")
  PROMPT="$PERSONA_CONTENT

---

$TASK"
fi

# Dispatch
LOG_FILE="$WS_DIR/agent-${AGENT}.log"
echo "Dispatching $AGENT to $WS_DIR..."

case "$AGENT_TYPE" in
  claude)
    cd "$WS_DIR/src"
    claude --dangerously-skip-permissions $MODEL_FLAG -p "$PROMPT" > "$LOG_FILE" 2>&1 &
    PID=$!
    ;;
  gemini)
    cd "$WS_DIR/src"
    gemini -p "$PROMPT" > "$LOG_FILE" 2>&1 &
    PID=$!
    ;;
  codex)
    cd "$WS_DIR/src"
    codex exec -p "$PROMPT" > "$LOG_FILE" 2>&1 &
    PID=$!
    ;;
esac

echo "{\"workspace\":\"$WS_DIR\",\"pid\":$PID,\"agent\":\"$AGENT\",\"repo\":\"$REPO\"}"
