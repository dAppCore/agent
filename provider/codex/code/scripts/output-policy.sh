#!/bin/bash
# Hook Output Policy - Expose vs Hide
#
# EXPOSE (additionalContext):
#   - Errors that need fixing
#   - Failures that block progress
#   - Security warnings
#   - Breaking changes
#
# HIDE (suppressOutput):
#   - Success confirmations
#   - Verbose progress output
#   - Repetitive status messages
#   - Debug information
#
# Usage:
#   source output-policy.sh
#   expose_error "Test failed: $error"
#   expose_warning "Debug statements found"
#   hide_success
#   pass_through "$input"

# Expose an error to Claude (always visible)
expose_error() {
    local message="$1"
    local context="$2"

    cat << EOF
{
  "hookSpecificOutput": {
    "additionalContext": "## ❌ Error\n\n$message${context:+\n\n$context}"
  }
}
EOF
}

# Expose a warning to Claude (visible, but not blocking)
expose_warning() {
    local message="$1"
    local context="$2"

    cat << EOF
{
  "hookSpecificOutput": {
    "additionalContext": "## ⚠️ Warning\n\n$message${context:+\n\n$context}"
  }
}
EOF
}

# Expose informational context (visible when relevant)
expose_info() {
    local message="$1"

    cat << EOF
{
  "hookSpecificOutput": {
    "additionalContext": "$message"
  }
}
EOF
}

# Hide output (success, no action needed)
hide_success() {
    echo '{"suppressOutput": true}'
}

# Pass through without modification (neutral)
pass_through() {
    echo "$1"
}

# Aggregate multiple issues into a summary
aggregate_issues() {
    local issues=("$@")
    local count=${#issues[@]}

    if [[ $count -eq 0 ]]; then
        hide_success
        return
    fi

    local summary=""
    local shown=0
    local max_shown=5

    for issue in "${issues[@]}"; do
        if [[ $shown -lt $max_shown ]]; then
            summary+="- $issue\n"
            ((shown++))
        fi
    done

    if [[ $count -gt $max_shown ]]; then
        summary+="\n... and $((count - max_shown)) more"
    fi

    expose_warning "$count issues found:" "$summary"
}
