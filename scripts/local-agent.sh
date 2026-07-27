#!/usr/bin/env bash
# SPDX-License-Identifier: EUPL-1.2
#
# Lightweight local-agent wrapper.
#
# Profiles:
#   gemma-main    -> MLX VLM 26B-A4B main lane on :8001
#   gemma-helper  -> MLX VLM E4B helper lane on :8005
#   qwen36        -> OpenAI-compatible Qwen3.6 27B coding lane on :8003
#   qwen36-moe    -> OpenAI-compatible Qwen3.6 35B-A3B MoE lane on :8008
#   ollama-qwen   -> legacy Ollama Qwen GGUF path
#
# Usage:
#   scripts/local-agent.sh --profile gemma-helper "summarise this workspace"
#   LOCAL_AGENT_PROFILE=qwen36 scripts/local-agent.sh "review the plan"
#   scripts/local-agent.sh --backend ollama --model hf.co/... "prompt"

set -euo pipefail

PROFILE="${LOCAL_AGENT_PROFILE:-gemma-helper}"
BACKEND="${LOCAL_AGENT_BACKEND:-}"
MODEL="${LOCAL_MODEL:-}"
SMALL_MODEL="${LOCAL_SMALL_MODEL:-}"
BASE_URL="${LOCAL_BASE_URL:-}"
API_KEY="${LOCAL_API_KEY:-sk-local}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
TEMPERATURE="${LOCAL_TEMPERATURE:-0.1}"
MAX_TOKENS="${LOCAL_MAX_TOKENS:-2048}"
CTX_SIZE="${LOCAL_CTX:-16384}"
ENABLE_THINKING="${LOCAL_ENABLE_THINKING:-false}"
FILE_LIMIT="${LOCAL_FILE_LIMIT:-800}"
DRY_RUN=0

usage() {
    cat <<'EOF'
usage: scripts/local-agent.sh [options] <prompt>

Options:
  --profile NAME      gemma-main, gemma-helper, qwen36, qwen36-moe, ollama-qwen
  --backend NAME      openai or ollama
  --base-url URL      OpenAI-compatible base URL, e.g. http://127.0.0.1:8005/v1
  --model NAME        Model name exposed by the local server
  --max-tokens N      Completion token limit
  --ctx N             Ollama context size
  --file-limit N      Max source file paths to include in the prompt, 0 = all
  --dry-run           Print resolved target and context size without calling a model
  -h, --help          Show this help

Environment mirrors the options:
  LOCAL_AGENT_PROFILE, LOCAL_AGENT_BACKEND, LOCAL_BASE_URL, LOCAL_MODEL,
  LOCAL_MAX_TOKENS, LOCAL_TEMPERATURE, LOCAL_ENABLE_THINKING, LOCAL_CTX,
  LOCAL_FILE_LIMIT.
EOF
}

apply_profile() {
    case "$PROFILE" in
        gemma-main|main26)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8001/v1}"
            MODEL="${MODEL:-mlx-community/gemma-4-26b-a4b-it-4bit}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/gemma-4-e4b-it-mxfp8}"
            ;;
        gemma-helper|gemma-e4b|helper-e4b)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8005/v1}"
            MODEL="${MODEL:-mlx-community/gemma-4-e4b-it-mxfp8}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/gemma-4-e4b-it-mxfp8}"
            ;;
        gemma-e2b|helper-e2b)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8004/v1}"
            MODEL="${MODEL:-mlx-community/gemma-4-e2b-it-4bit}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/gemma-4-e2b-it-4bit}"
            ;;
        qwen36|qwen3.6|qwen36-mlx|qwen36-27b|qwen36-coder)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8003/v1}"
            MODEL="${MODEL:-mlx-community/Qwen3.6-27B-4bit}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/gemma-4-e4b-it-mxfp8}"
            ;;
        qwen36-27b-mxfp8|qwen36-mxfp8)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8006/v1}"
            MODEL="${MODEL:-mlx-community/Qwen3.6-27B-mxfp8}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/gemma-4-e4b-it-mxfp8}"
            ;;
        qwen36-moe|qwen36-35b|qwen36-35b-a3b)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8008/v1}"
            MODEL="${MODEL:-mlx-community/Qwen3.6-35B-A3B-4bit}"
            SMALL_MODEL="${SMALL_MODEL:-mlx-community/Qwen3.6-27B-4bit}"
            ;;
        ollama-qwen|qwen-ollama|ollama)
            BACKEND="${BACKEND:-ollama}"
            MODEL="${MODEL:-hf.co/unsloth/Qwen3-Coder-Next-GGUF:UD-IQ4_NL}"
            ;;
        *)
            BACKEND="${BACKEND:-openai}"
            BASE_URL="${BASE_URL:-http://127.0.0.1:8000/v1}"
            MODEL="${MODEL:-$PROFILE}"
            ;;
    esac
}

append_file() {
    local title="$1"
    local path="$2"
    local limit="${3:-0}"

    if [[ ! -f "$path" ]]; then
        return
    fi

    CONTEXT="${CONTEXT}

=== ${title} (${path}) ===
"
    if [[ "$limit" == "0" ]]; then
        CONTEXT="${CONTEXT}$(cat "$path")
"
    else
        CONTEXT="${CONTEXT}$(head -n "$limit" "$path")
"
    fi
}

collect_files() {
    local files
    files="$(find . \
        \( -name "*.go" -o -name "*.php" -o -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.py" -o -name "*.md" \) \
        -not -path "*/vendor/*" \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -not -path "*/.core/*" \
        | sort)"
    if [[ "$FILE_LIMIT" == "0" ]]; then
        printf "%s\n" "$files"
    else
        local rows=()
        local index=0
        mapfile -t rows <<<"$files"
        for path in "${rows[@]}"; do
            if [[ "$index" -ge "$FILE_LIMIT" ]]; then
                break
            fi
            printf "%s\n" "$path"
            index=$((index + 1))
        done
    fi
}

build_prompt() {
    local prompt="$1"
    CONTEXT=""

    append_file "PROJECT CONVENTIONS" "AGENTS.md"
    append_file "PROJECT CONVENTIONS" "CLAUDE.md"
    append_file "ENTRY POINTS" "llm.txt"
    append_file "WORK PLAN" "PLAN.md"
    append_file "TASK" "TODO.md"
    append_file "PRIOR KNOWLEDGE" "CONTEXT.md" 200
    append_file "CONSUMERS" "CONSUMERS.md"
    append_file "RECENT CHANGES" "RECENT.md"

    FILES="$(collect_files)"

    FULL_PROMPT="${CONTEXT}

=== INSTRUCTIONS ===
${prompt}

=== LOCAL AGENT CONTRACT ===
You are a local helper model. Keep the main agent's context small: inspect the provided project context, identify the exact files or commands needed, and return a compact result. If external tools are needed, describe the requested tool call precisely instead of pretending it was run.

=== SOURCE FILES IN THIS REPO ===
${FILES}
"
}

openai_payload() {
    python3 -c '
import json
import sys

model, max_tokens, temperature, enable_thinking = sys.argv[1:5]
prompt = sys.stdin.read()
payload = {
    "model": model,
    "messages": [{"role": "user", "content": prompt}],
    "max_tokens": int(max_tokens),
    "temperature": float(temperature),
    "enable_thinking": enable_thinking.lower() in ("1", "true", "yes", "on"),
}
print(json.dumps(payload))
' "$MODEL" "$MAX_TOKENS" "$TEMPERATURE" "$ENABLE_THINKING" <<<"$FULL_PROMPT"
}

ollama_payload() {
    python3 -c '
import json
import sys

model, ctx_size, temperature = sys.argv[1:4]
prompt = sys.stdin.read()
payload = {
    "model": model,
    "prompt": prompt,
    "stream": False,
    "keep_alive": "5m",
    "options": {
        "temperature": float(temperature),
        "num_ctx": int(ctx_size),
        "top_p": 0.95,
        "top_k": 40,
    },
}
print(json.dumps(payload))
' "$MODEL" "$CTX_SIZE" "$TEMPERATURE" <<<"$FULL_PROMPT"
}

print_openai_response() {
    python3 -c '
import json
import sys

try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    print("Error: failed to parse response")
    raise SystemExit(1)

if "error" in data:
    print(json.dumps(data["error"], indent=2, sort_keys=True))
    raise SystemExit(1)

choices = data.get("choices") or []
message = (choices[0].get("message") if choices else {}) or {}
content = message.get("content")
if content:
    print(content)
else:
    print(json.dumps(data, indent=2, sort_keys=True))
'
}

print_ollama_response() {
    python3 -c '
import json
import sys

try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    print("Error: failed to parse response")
    raise SystemExit(1)

print(data.get("response", "Error: no response"))
'
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --profile)
            PROFILE="$2"
            shift 2
            ;;
        --backend)
            BACKEND="$2"
            shift 2
            ;;
        --base-url)
            BASE_URL="$2"
            shift 2
            ;;
        --model)
            MODEL="$2"
            shift 2
            ;;
        --max-tokens)
            MAX_TOKENS="$2"
            shift 2
            ;;
        --ctx)
            CTX_SIZE="$2"
            shift 2
            ;;
        --file-limit)
            FILE_LIMIT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        -*)
            echo "unknown option: $1" >&2
            usage >&2
            exit 2
            ;;
        *)
            break
            ;;
    esac
done

if [[ $# -eq 0 ]]; then
    usage >&2
    exit 2
fi

PROMPT="$*"
apply_profile
build_prompt "$PROMPT"

if [[ "$DRY_RUN" == "1" ]]; then
    echo "profile=${PROFILE}"
    echo "backend=${BACKEND}"
    echo "base_url=${BASE_URL:-}"
    echo "model=${MODEL}"
    echo "small_model=${SMALL_MODEL:-}"
    echo "prompt_chars=${#FULL_PROMPT}"
    echo "files=$(printf "%s\n" "$FILES" | sed '/^$/d' | wc -l | tr -d ' ')"
    exit 0
fi

case "$BACKEND" in
    openai)
        if [[ -z "$BASE_URL" ]]; then
            echo "LOCAL_BASE_URL or --base-url is required for openai backend" >&2
            exit 2
        fi
        curl -s "${BASE_URL%/}/chat/completions" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${API_KEY}" \
            -d "$(openai_payload)" \
            | print_openai_response
        ;;
    ollama)
        curl -s "${OLLAMA_URL%/}/api/generate" \
            -H "Content-Type: application/json" \
            -d "$(ollama_payload)" \
            | print_ollama_response
        ;;
    *)
        echo "unknown backend: ${BACKEND}" >&2
        exit 2
        ;;
esac
