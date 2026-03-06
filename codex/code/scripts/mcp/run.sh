#!/bin/bash
#
# MCP Server script for the core-claude plugin.
# This script reads a JSON MCP request from stdin, executes the corresponding
# core CLI command, and prints a JSON response to stdout.
#

set -e

# Read the entire input from stdin
request_json=$(cat)

# --- Input Validation ---
if ! echo "$request_json" | jq . > /dev/null 2>&1; then
    echo '{"status": "error", "message": "Invalid JSON request."}'
    exit 1
fi

# --- Request Parsing ---
tool_name=$(echo "$request_json" | jq -r '.tool_name')
params=$(echo "$request_json" | jq '.parameters')

# --- Command Routing ---
case "$tool_name" in
    "core_go_test")
        filter=$(echo "$params" | jq -r '.filter // ""')
        coverage=$(echo "$params" | jq -r '.coverage // false')

        # Build the command
        cmd_args=("go" "test")
        [ -n "$filter" ] && cmd_args+=("--filter=$filter")
        [ "$coverage" = "true" ] && cmd_args+=("--coverage")
        ;;

    "core_dev_health")
        cmd_args=("dev" "health")
        ;;

    "core_dev_commit")
        message=$(echo "$params" | jq -r '.message // ""')
        if [ -z "$message" ]; then
            echo '{"status": "error", "message": "Missing required parameter: message"}'
            exit 1
        fi

        cmd_args=("dev" "commit" "-m" "$message")

        repos=$(echo "$params" | jq -r '.repos // "[]"')
        if [ "$(echo "$repos" | jq 'length')" -gt 0 ]; then
            # Read repos into a bash array
            mapfile -t repo_array < <(echo "$repos" | jq -r '.[]')
            cmd_args+=("${repo_array[@]}")
        fi
        ;;

    *)
        echo "{\"status\": \"error\", \"message\": \"Unknown tool: $tool_name\"}"
        exit 1
        ;;
esac

# --- Command Execution ---
# The 'core' command is expected to be in the PATH of the execution environment.
output=$(core "${cmd_args[@]}" 2>&1)
exit_code=$?

# --- Response Formatting ---
if [ $exit_code -eq 0 ]; then
    status="success"
else
    status="error"
fi

# Default response is just the raw output
result_json=$(jq -n --arg raw "$output" '{raw: $raw}')

# Structured Response Parsing
if [ "$tool_name" = "core_go_test" ]; then
    if [ "$status" = "success" ]; then
        # Use awk for more robust parsing of the test output.
        # This is less brittle than grepping for exact lines.
        outcome=$(printf "%s" "$output" | awk '/^PASS$/ {print "PASS"}')
        coverage=$(printf "%s" "$output" | awk '/coverage:/ {print $2}')
        summary=$(printf "%s" "$output" | awk '/^ok\s/ {print $0}')

        result_json=$(jq -n \
            --arg outcome "${outcome:-UNKNOWN}" \
            --arg coverage "${coverage:--}" \
            --arg summary "${summary:--}" \
            --arg raw_output "$output" \
            '{
                outcome: $outcome,
                coverage: $coverage,
                summary: $summary,
                raw_output: $raw_output
            }')
    else
        # In case of failure, the output is less predictable.
        # We'll grab what we can, but the raw output is most important.
        outcome=$(printf "%s" "$output" | awk '/^FAIL$/ {print "FAIL"}')
        summary=$(printf "%s" "$output" | awk '/^FAIL\s/ {print $0}')
        result_json=$(jq -n \
            --arg outcome "${outcome:-FAIL}" \
            --arg summary "${summary:--}" \
            --arg raw_output "$output" \
            '{
                outcome: $outcome,
                summary: $summary,
                raw_output: $raw_output
            }')
    fi
elif [ "$tool_name" = "core_dev_health" ]; then
    if [ "$status" = "success" ]; then
        # Safely parse the "key: value" output into a JSON array of objects.
        # This uses jq to be robust against special characters in the output.
        result_json=$(printf "%s" "$output" | jq -R 'capture("(?<name>[^:]+):\\s*(?<status>.*)")' | jq -s '{services: .}')
    else
        # On error, just return the raw output
        result_json=$(jq -n --arg error "$output" '{error: $error}')
    fi
elif [ "$tool_name" = "core_dev_commit" ]; then
    if [ "$status" = "success" ]; then
        result_json=$(jq -n --arg message "$output" '{message: $message}')
    else
        result_json=$(jq -n --arg error "$output" '{error: $error}')
    fi
fi

response=$(jq -n --arg status "$status" --argjson result "$result_json" '{status: $status, result: $result}')
echo "$response"

exit 0
