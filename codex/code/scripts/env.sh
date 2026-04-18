#!/bin/bash
# Environment management script for /core:env command

set -e

# Function to mask sensitive values
mask_sensitive_value() {
    local key="$1"
    local value="$2"
    if [[ "$key" =~ (_SECRET|_KEY|_PASSWORD|_TOKEN)$ ]]; then
        if [ -z "$value" ]; then
            echo "***not set***"
        else
            echo "***set***"
        fi
    else
        echo "$value"
    fi
}

# The subcommand is the first argument
SUBCOMMAND="$1"

case "$SUBCOMMAND" in
    "")
        # Default command: Show env vars
        if [ ! -f ".env" ]; then
            echo ".env file not found."
            exit 1
        fi
        while IFS= read -r line || [[ -n "$line" ]]; do
            # Skip comments and empty lines
            if [[ "$line" =~ ^\s*#.*$ || -z "$line" ]]; then
                continue
            fi
            # Extract key and value
            key=$(echo "$line" | cut -d '=' -f 1)
            value=$(echo "$line" | cut -d '=' -f 2-)
            masked_value=$(mask_sensitive_value "$key" "$value")
            echo "$key=$masked_value"
        done < ".env"
        ;;
    check)
        # Subcommand: check
        if [ ! -f ".env.example" ]; then
            echo ".env.example file not found."
            exit 1
        fi

        # Create an associative array of env vars
        declare -A env_vars
        if [ -f ".env" ]; then
            while IFS= read -r line || [[ -n "$line" ]]; do
                if [[ ! "$line" =~ ^\s*# && "$line" =~ = ]]; then
                    key=$(echo "$line" | cut -d '=' -f 1)
                    value=$(echo "$line" | cut -d '=' -f 2-)
                    env_vars["$key"]="$value"
                fi
            done < ".env"
        fi

        echo "Environment Check"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo

        errors=0
        warnings=0

        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ -z "$line" || "$line" =~ ^\s*# ]]; then
                continue
            fi

            example_key=$(echo "$line" | cut -d '=' -f 1)
            example_value=$(echo "$line" | cut -d '=' -f 2-)

            if [[ ${env_vars[$example_key]+_} ]]; then
                # Key exists in .env
                env_value="${env_vars[$example_key]}"
                if [ -n "$env_value" ]; then
                    echo "✓ $example_key=$(mask_sensitive_value "$example_key" "$env_value")"
                else
                    # Key exists but value is empty
                    if [ -z "$example_value" ]; then
                        echo "✗ $example_key missing (required, no default)"
                        ((errors++))
                    else
                        echo "⚠ $example_key missing (default: $example_value)"
                        ((warnings++))
                    fi
                fi
            else
                # Key does not exist in .env
                if [ -z "$example_value" ]; then
                    echo "✗ $example_key missing (required, no default)"
                    ((errors++))
                else
                    echo "⚠ $example_key missing (default: $example_value)"
                    ((warnings++))
                fi
            fi
        done < ".env.example"

        echo
        if [ "$errors" -gt 0 ] || [ "$warnings" -gt 0 ]; then
            echo "$errors errors, $warnings warnings"
        else
            echo "✓ All checks passed."
        fi
        ;;
    diff)
        # Subcommand: diff
        if [ ! -f ".env.example" ]; then
            echo ".env.example file not found."
            exit 1
        fi

        # Create associative arrays for both files
        declare -A env_vars
        if [ -f ".env" ]; then
            while IFS= read -r line || [[ -n "$line" ]]; do
                if [[ ! "$line" =~ ^\s*# && "$line" =~ = ]]; then
                    key=$(echo "$line" | cut -d '=' -f 1)
                    value=$(echo "$line" | cut -d '=' -f 2-)
                    env_vars["$key"]="$value"
                fi
            done < ".env"
        fi

        declare -A example_vars
        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ ! "$line" =~ ^\s*# && "$line" =~ = ]]; then
                key=$(echo "$line" | cut -d '=' -f 1)
                value=$(echo "$line" | cut -d '=' -f 2-)
                example_vars["$key"]="$value"
            fi
        done < ".env.example"

        echo "Environment Diff"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo

        # Check for modifications and deletions
        for key in "${!example_vars[@]}"; do
            example_value="${example_vars[$key]}"
            if [[ ${env_vars[$key]+_} ]]; then
                # Key exists in .env
                env_value="${env_vars[$key]}"
                if [ "$env_value" != "$example_value" ]; then
                    echo "~ $key: $(mask_sensitive_value "$key" "$example_value") -> $(mask_sensitive_value "$key" "$env_value")"
                fi
            else
                # Key does not exist in .env
                echo "- $key: $(mask_sensitive_value "$key" "$example_value")"
            fi
        done

        # Check for additions
        for key in "${!env_vars[@]}"; do
            if [[ ! ${example_vars[$key]+_} ]]; then
                echo "+ $key: $(mask_sensitive_value "$key" "${env_vars[$key]}")"
            fi
        done
        ;;
    sync)
        # Subcommand: sync
        if [ ! -f ".env.example" ]; then
            echo ".env.example file not found."
            exit 1
        fi

        # Create an associative array of env vars
        declare -A env_vars
        if [ -f ".env" ]; then
            while IFS= read -r line || [[ -n "$line" ]]; do
                if [[ ! "$line" =~ ^\s*# && "$line" =~ = ]]; then
                    key=$(echo "$line" | cut -d '=' -f 1)
                    value=$(echo "$line" | cut -d '=' -f 2-)
                    env_vars["$key"]="$value"
                fi
            done < ".env"
        fi

        while IFS= read -r line || [[ -n "$line" ]]; do
            if [[ -z "$line" || "$line" =~ ^\s*# ]]; then
                continue
            fi

            example_key=$(echo "$line" | cut -d '=' -f 1)
            example_value=$(echo "$line" | cut -d '=' -f 2-)

            if [[ ! ${env_vars[$example_key]+_} ]]; then
                # Key does not exist in .env, so add it
                echo "$example_key=$example_value" >> ".env"
                echo "Added: $example_key"
            fi
        done < ".env.example"

        echo "Sync complete."
        ;;
    *)
        echo "Unknown subcommand: $SUBCOMMAND"
        exit 1
        ;;
esac
