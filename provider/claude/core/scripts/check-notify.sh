#!/bin/bash
# Lightweight inbox notification check for PostToolUse hook.
# Reads a marker file written by the monitor subsystem.
# If marker exists, outputs the notification and removes the file.
# Zero API calls — just a file stat.

NOTIFY_FILE="/tmp/claude-inbox-notify"

if [ -f "$NOTIFY_FILE" ]; then
    cat "$NOTIFY_FILE"
    rm -f "$NOTIFY_FILE"
fi
