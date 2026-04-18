#!/bin/bash

set -e

if [ -f "go.mod" ]; then
  PROJECT_TYPE="go"
elif [ -f "composer.json" ]; then
  PROJECT_TYPE="php"
else
  echo "Error: Unable to determine project type. No go.mod or composer.json found."
  exit 1
fi

echo "Detected project type: $PROJECT_TYPE"

case $PROJECT_TYPE in
  "go")
    if [ ! -f "mem.prof" ]; then
      echo "Error: Memory profile 'mem.prof' not found."
      echo "Please run '/core:perf test' on your Go project first to generate the memory profile."
      exit 1
    fi
    echo "Analyzing Go memory profile..."
    go tool pprof -top mem.prof
    ;;
  "php")
    if ! php -m | grep -q 'Xdebug'; then
      echo "Xdebug is not installed. Please install it to use the PHP memory profiler."
      exit 1
    fi

    if [ -z "$1" ]; then
      echo "Usage for PHP: $0 <path_to_php_script>"
      exit 1
    fi

    PHP_SCRIPT=$1
    if [ ! -f "$PHP_SCRIPT" ]; then
      echo "Error: File not found: $PHP_SCRIPT"
      exit 1
    fi

    echo "Generating memory profile for $PHP_SCRIPT..."
    # Generate a unique filename for the profile output
    PROFILE_OUTPUT="cachegrind.out.$$"
    XDEBUG_MODE=profile php -d xdebug.profiler_output_name="$PROFILE_OUTPUT" "$PHP_SCRIPT" > /dev/null 2>&1

    if [ ! -f "$PROFILE_OUTPUT" ]; then
        echo "Error: Memory profile could not be generated. Check your Xdebug configuration."
        exit 1
    fi

    echo "Analyzing memory profile..."
    # Parse the cachegrind file to find functions with high memory usage
    awk '
        /^fn=/ {
            current_function = substr($0, 4)
        }
        /^[0-9]/ {
            # Column 2 is self-inclusive memory cost
            memory_cost = $2
            functions[current_function] += memory_cost
        }
        END {
            for (func in functions) {
                printf "%12d %s\n", functions[func], func
            }
        }
    ' "$PROFILE_OUTPUT" | sort -nr | head -n 10 > top_memory_functions.log

    echo "--- Top 10 Memory-Consuming Functions (in bytes) ---"
    cat top_memory_functions.log
    echo "----------------------------------------------------"

    echo "Actionable Suggestions:"
    echo "  - Review the functions listed above. High memory usage may indicate large arrays, unreleased resources, or inefficient data structures."
    echo "  - For processing large files or database results, consider using generators to reduce memory footprint."

    # Clean up the generated files
    rm "$PROFILE_OUTPUT" top_memory_functions.log
    ;;
esac
