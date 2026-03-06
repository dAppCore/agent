#!/bin/bash

# Patterns for detecting secrets
PATTERNS=(
  # API keys (e.g., sk_live_..., ghp_..., etc.)
  "[a-zA-Z0-9]{32,}"
  # AWS keys
  "AKIA[0-9A-Z]{16}"
  # Private keys
  "-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----"
  # Passwords in config
  "(password|passwd|pwd)\s*[=:]\s*['\"][^'\"]+['\"]"
  # Tokens
  "(token|secret|key)\s*[=:]\s*['\"][^'\"]+['\"]"
)

# Exceptions for fake secrets
EXCEPTIONS=(
  "password123"
  "your-api-key-here"
  "xxx"
  "test"
  "example"
)

# File to check is passed as the first argument
FILE_PATH=$1

# Function to check for secrets
check_secrets() {
  local input_source="$1"
  local file_path="$2"
  local line_num=0
  while IFS= read -r line; do
    line_num=$((line_num + 1))
    for pattern in "${PATTERNS[@]}"; do
      if echo "$line" | grep -qE "$pattern"; then
        # Check for exceptions
        is_exception=false
        for exception in "${EXCEPTIONS[@]}"; do
          if echo "$line" | grep -qF "$exception"; then
            is_exception=true
            break
          fi
        done

        if [ "$is_exception" = false ]; then
          echo "⚠️ Potential secret detected!"
          echo "File: $file_path"
          echo "Line: $line_num"
          echo ""
          echo "Found: $line"
          echo ""
          echo "This looks like a production secret."
          echo "Use environment variables instead."
          echo ""

          # Propose a fix (example for a PHP config file)
          if [[ "$file_path" == *.php ]]; then
            echo "'stripe' => ["
            echo "    'secret' => env('STRIPE_SECRET'),  // ✓"
            echo "]"
          fi
          exit 1
        fi
      fi
    done
  done < "$input_source"
}

check_secrets "/dev/stdin" "$FILE_PATH"

exit 0
