#!/bin/bash

# This script generates a TypeScript/JavaScript API client or an OpenAPI spec
# from a Laravel routes file. It works by running a PHP script to parse the
# routes into JSON, and then uses jq to transform the JSON into the desired
# output format.

# Path to the PHP script that parses the Laravel routes.
PHP_SCRIPT="$(dirname "$0")/../php/generate.php"

# Run the PHP script and capture the JSON output.
ROUTES_JSON=$(php "$PHP_SCRIPT")

# --- Argument Parsing ---
# Initialize flags for the different output formats.
TS=false
JS=false
OPENAPI=false

# Loop through the command-line arguments to determine which output format
# to generate.
for arg in "$@"; do
    case $arg in
        --ts)
            TS=true
            shift # Remove --ts from the list of arguments
            ;;
        --js)
            JS=true
            shift # Remove --js from the list of arguments
            ;;
        --openapi)
            OPENAPI=true
            shift # Remove --openapi from the list of arguments
            ;;
    esac
done

# Default to TypeScript if no language is specified. This ensures that the
# script always generates at least one output format.
if [ "$JS" = false ] && [ "$OPENAPI" = false ]; then
    TS=true
fi

# --- TypeScript Client Generation ---
if [ "$TS" = true ]; then
    # Start by creating the api.ts file and adding the header.
    echo "// Generated from routes/api.php" > api.ts
    echo "export const api = {" >> api.ts

    # Use jq to transform the JSON into a TypeScript client.
    echo "$ROUTES_JSON" | jq -r '
        [group_by(.uri | split("/")[1]) | .[] | {
            key: .[0].uri | split("/")[1],
            value: .
        }] | from_entries | to_entries | map(
            "  \(.key): {\n" +
            (.value | map(
                "    \(.action_name): (" +
                (.parameters | map("\(.): number") | join(", ")) +
                (if (.method == "POST" or .method == "PUT") and (.parameters | length > 0) then ", " else "" end) +
                (if .method == "POST" or .method == "PUT" then "data: any" else "" end) +
                ") => fetch(`/\(.uri | gsub("{"; "${") | gsub("}"; "}"))`, {" +
                (if .method != "GET" then "\n      method: \"\(.method)\"," else "" end) +
                (if .method == "POST" or .method == "PUT" then "\n      body: JSON.stringify(data)" else "" end) +
                "\n    }),"
            ) | join("\n")) +
            "\n  },"
        ) | join("\n")
    ' >> api.ts
    echo "};" >> api.ts
fi

# --- JavaScript Client Generation ---
if [ "$JS" = true ]; then
    # Start by creating the api.js file and adding the header.
    echo "// Generated from routes/api.php" > api.js
    echo "export const api = {" >> api.js

    # The jq filter for JavaScript is similar to the TypeScript filter, but
    # it doesn't include type annotations.
    echo "$ROUTES_JSON" | jq -r '
        [group_by(.uri | split("/")[1]) | .[] | {
            key: .[0].uri | split("/")[1],
            value: .
        }] | from_entries | to_entries | map(
            "  \(.key): {\n" +
            (.value | map(
                "    \(.action_name): (" +
                (.parameters | join(", ")) +
                (if (.method == "POST" or .method == "PUT") and (.parameters | length > 0) then ", " else "" end) +
                (if .method == "POST" or .method == "PUT" then "data" else "" end) +
                ") => fetch(`/\(.uri | gsub("{"; "${") | gsub("}"; "}"))`, {" +
                (if .method != "GET" then "\n      method: \"\(.method)\"," else "" end) +
                (if .method == "POST" or .method == "PUT" then "\n      body: JSON.stringify(data)" else "" end) +
                "\n    }),"
            ) | join("\n")) +
            "\n  },"
        ) | join("\n")
    ' >> api.js
    echo "};" >> api.js
fi

# --- OpenAPI Spec Generation ---
if [ "$OPENAPI" = true ]; then
    # Start by creating the openapi.yaml file and adding the header.
    echo "openapi: 3.0.0" > openapi.yaml
    echo "info:" >> openapi.yaml
    echo "  title: API" >> openapi.yaml
    echo "  version: 1.0.0" >> openapi.yaml
    echo "paths:" >> openapi.yaml

    # The jq filter for OpenAPI generates a YAML file with the correct structure.
    # It groups the routes by URI, and then for each URI, it creates a path
    # entry with the correct HTTP methods.
    echo "$ROUTES_JSON" | jq -r '
        group_by(.uri) | .[] |
        "  /\(.[0].uri):\n" +
        (map("    " + (.method | ascii_downcase | split("|")[0]) + ":\n" +
        "      summary: \(.action)\n" +
        "      responses:\n" +
        "        \"200\":\n" +
        "          description: OK") | join("\n"))
    ' >> openapi.yaml
fi
