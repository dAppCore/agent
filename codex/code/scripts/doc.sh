#!/bin/bash

# Default path is the current directory
TARGET_PATH="."
ARGS=()

# Parse --path argument
# This allows testing by pointing the command to a mock project directory.
for arg in "$@"; do
  case $arg in
    --path=*)
      TARGET_PATH="${arg#*=}"
      ;;
    *)
      ARGS+=("$arg")
      ;;
  esac
done

# The subcommand is the first positional argument
SUBCOMMAND="${ARGS[0]}"
# The second argument is the name for class/module
NAME="${ARGS[1]}"
# The third argument is the optional path for api
SCAN_PATH="${ARGS[2]}"

# Get the directory where this script is located to call sub-scripts
SCRIPT_DIR=$(dirname "$0")

case "$SUBCOMMAND" in
  class)
    if [ -z "$NAME" ]; then
      echo "Error: Missing class name." >&2
      echo "Usage: /core:doc class <ClassName>" >&2
      exit 1
    fi
    "${SCRIPT_DIR}/doc-class.sh" "$NAME" "$TARGET_PATH"
    ;;
  module)
    if [ -z "$NAME" ]; then
      echo "Error: Missing module name." >&2
      echo "Usage: /core:doc module <ModuleName>" >&2
      exit 1
    fi
    "${SCRIPT_DIR}/doc-module.sh" "$NAME" "$TARGET_PATH"
    ;;
  api)
    "${SCRIPT_DIR}/doc-api.sh" "$TARGET_PATH" "$SCAN_PATH"
    ;;
  changelog)
    "${SCRIPT_DIR}/doc-changelog.sh" "$TARGET_PATH"
    ;;
  *)
    echo "Error: Unknown subcommand '$SUBCOMMAND'." >&2
    echo "Usage: /core:doc [class|module|api|changelog] [name]" >&2
    exit 1
    ;;
esac
