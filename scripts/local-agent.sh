#!/bin/bash
# Local agent wrapper — runs Ollama model on workspace files
# Usage: local-agent.sh <prompt>
#
# Reads PROMPT.md, CLAUDE.md, TODO.md, PLAN.md from current directory,
# combines them into a single prompt, sends to Ollama, outputs result.

set -e

PROMPT="$1"
MODEL="${LOCAL_MODEL:-hf.co/unsloth/Qwen3-Coder-Next-GGUF:UD-IQ4_NL}"
CTX_SIZE="${LOCAL_CTX:-16384}"

# Build context from workspace files
CONTEXT=""

if [ -f "CLAUDE.md" ]; then
    CONTEXT="${CONTEXT}

=== PROJECT CONVENTIONS (CLAUDE.md) ===
$(cat CLAUDE.md)
"
fi

if [ -f "PLAN.md" ]; then
    CONTEXT="${CONTEXT}

=== WORK PLAN (PLAN.md) ===
$(cat PLAN.md)
"
fi

if [ -f "TODO.md" ]; then
    CONTEXT="${CONTEXT}

=== TASK (TODO.md) ===
$(cat TODO.md)
"
fi

if [ -f "CONTEXT.md" ]; then
    CONTEXT="${CONTEXT}

=== PRIOR KNOWLEDGE (CONTEXT.md) ===
$(head -200 CONTEXT.md)
"
fi

if [ -f "CONSUMERS.md" ]; then
    CONTEXT="${CONTEXT}

=== CONSUMERS (CONSUMERS.md) ===
$(cat CONSUMERS.md)
"
fi

if [ -f "RECENT.md" ]; then
    CONTEXT="${CONTEXT}

=== RECENT CHANGES (RECENT.md) ===
$(cat RECENT.md)
"
fi

# List all source files for the model to review
FILES=""
if [ -d "." ]; then
    FILES=$(find . -name "*.go" -o -name "*.php" -o -name "*.ts" | grep -v vendor | grep -v node_modules | grep -v ".git" | sort)
fi

# Build the full prompt
FULL_PROMPT="${CONTEXT}

=== INSTRUCTIONS ===
${PROMPT}

=== SOURCE FILES IN THIS REPO ===
${FILES}

Review each source file listed above. Read them one at a time and report your findings.
For each file, use: cat <filename> to read it, then analyse it according to the instructions.
"

# Call Ollama API (non-streaming for clean output)
RESPONSE=$(curl -s http://localhost:11434/api/generate \
    -d "$(python3 -c "
import json
print(json.dumps({
    'model': '${MODEL}',
    'prompt': $(python3 -c "import json,sys; print(json.dumps(sys.stdin.read()))" <<< "$FULL_PROMPT"),
    'stream': False,
    'options': {
        'temperature': 0.1,
        'num_ctx': ${CTX_SIZE},
        'top_p': 0.95,
        'top_k': 40
    }
}))
")" 2>/dev/null)

# Extract and output the response
echo "$RESPONSE" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    print(d.get('response', 'Error: no response'))
except:
    print('Error: failed to parse response')
"
