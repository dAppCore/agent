#!/bin/bash

TARGET_PATH=$1
# The second argument can be a path to scan for API endpoints.
SCAN_PATH=$2

if [ -z "$TARGET_PATH" ]; then
  echo "Usage: doc-api.sh <TargetPath> [ScanPath]" >&2
  exit 1
fi

# Default to scanning the 'src' directory if no path is provided.
if [ -z "$SCAN_PATH" ]; then
  SCAN_PATH="src"
fi

SWAGGER_PHP_PATH="${TARGET_PATH}/vendor/bin/swagger-php"
FULL_SCAN_PATH="${TARGET_PATH}/${SCAN_PATH}"

if [ ! -d "$FULL_SCAN_PATH" ]; then
    echo "Error: Scan directory does not exist at '$FULL_SCAN_PATH'." >&2
    exit 1
fi

if [ -f "$SWAGGER_PHP_PATH" ]; then
  echo "Found swagger-php. Generating OpenAPI spec from '$FULL_SCAN_PATH'..."
  "$SWAGGER_PHP_PATH" "$FULL_SCAN_PATH"
else
  echo "Error: 'swagger-php' not found at '$SWAGGER_PHP_PATH'." >&2
  echo "Please ensure it is installed in your project's dev dependencies." >&2
  exit 1
fi
