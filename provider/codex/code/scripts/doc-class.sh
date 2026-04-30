#!/bin/bash

CLASS_NAME=$1
TARGET_PATH=$2

if [ -z "$CLASS_NAME" ] || [ -z "$TARGET_PATH" ]; then
  echo "Usage: doc-class.sh <ClassName> <TargetPath>" >&2
  exit 1
fi

# Find the file in the target path
FILE_PATH=$(find "$TARGET_PATH" -type f -name "${CLASS_NAME}.php")

if [ -z "$FILE_PATH" ]; then
  echo "Error: File for class '$CLASS_NAME' not found in '$TARGET_PATH'." >&2
  exit 1
fi

if [ $(echo "$FILE_PATH" | wc -l) -gt 1 ]; then
  echo "Error: Multiple files found for class '$CLASS_NAME':" >&2
  echo "$FILE_PATH" >&2
  exit 1
fi

# --- PARSING ---
SCRIPT_DIR=$(dirname "$0")
# Use the new PHP parser to get a JSON representation of the class.
# The `jq` tool is used to parse the JSON. It's a common dependency.
PARSED_JSON=$(php "${SCRIPT_DIR}/doc-class-parser.php" "$FILE_PATH")

if [ $? -ne 0 ]; then
    echo "Error: PHP parser failed." >&2
    echo "$PARSED_JSON" >&2
    exit 1
fi

# --- MARKDOWN GENERATION ---
CLASS_NAME=$(echo "$PARSED_JSON" | jq -r '.className')
CLASS_DESCRIPTION=$(echo "$PARSED_JSON" | jq -r '.description')

echo "# $CLASS_NAME"
echo ""
echo "$CLASS_DESCRIPTION"
echo ""
echo "## Methods"
echo ""

# Iterate over each method in the JSON
echo "$PARSED_JSON" | jq -c '.methods[]' | while read -r METHOD_JSON; do
    METHOD_NAME=$(echo "$METHOD_JSON" | jq -r '.name')
    # This is a bit fragile, but it's the best we can do for now
    # to get the full signature.
    METHOD_SIGNATURE=$(grep "function ${METHOD_NAME}" "$FILE_PATH" | sed -e 's/.*public function //' -e 's/{//' | xargs)

    echo "### $METHOD_SIGNATURE"

    # Method description
    METHOD_DESCRIPTION=$(echo "$METHOD_JSON" | jq -r '.description')
    if [ -n "$METHOD_DESCRIPTION" ]; then
        echo ""
        echo "$METHOD_DESCRIPTION"
    fi

    # Parameters
    PARAMS_JSON=$(echo "$METHOD_JSON" | jq -c '.params | to_entries')
    if [ "$PARAMS_JSON" != "[]" ]; then
        echo ""
        echo "**Parameters:**"
        echo "$PARAMS_JSON" | jq -c '.[]' | while read -r PARAM_JSON; do
            PARAM_NAME=$(echo "$PARAM_JSON" | jq -r '.key')
            PARAM_TYPE=$(echo "$PARAM_JSON" | jq -r '.value.type')
            PARAM_REQUIRED=$(echo "$PARAM_JSON" | jq -r '.value.required')
            PARAM_DESC=$(echo "$PARAM_JSON" | jq -r '.value.description')

            REQUIRED_TEXT=""
            if [ "$PARAM_REQUIRED" = "true" ]; then
                REQUIRED_TEXT=", required"
            fi

            echo "- \`$PARAM_NAME\` ($PARAM_TYPE$REQUIRED_TEXT) $PARAM_DESC"
        done
    fi

    # Return type
    RETURN_JSON=$(echo "$METHOD_JSON" | jq -c '.return')
    if [ "$RETURN_JSON" != "null" ]; then
        RETURN_TYPE=$(echo "$RETURN_JSON" | jq -r '.type')
        RETURN_DESC=$(echo "$RETURN_JSON" | jq -r '.description')
        echo ""
        if [ -n "$RETURN_DESC" ]; then
            echo "**Returns:** \`$RETURN_TYPE\` $RETURN_DESC"
        else
            echo "**Returns:** \`$RETURN_TYPE\`"
        fi
    fi
    echo ""
done

exit 0
