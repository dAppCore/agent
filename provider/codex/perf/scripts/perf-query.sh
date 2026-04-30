#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 \"<query>\""
  echo "Required environment variables: DB_HOST, DB_USER, DB_PASS, DB_NAME"
  exit 1
fi

if ! command -v mysql &> /dev/null; then
    echo "mysql command could not be found. Please install the MySQL client."
    exit 1
fi

if [ -z "$DB_HOST" ] || [ -z "$DB_USER" ] || [ -z "$DB_PASS" ] || [ -z "$DB_NAME" ]; then
  echo "Error: Missing required database environment variables."
  echo "Please set DB_HOST, DB_USER, DB_PASS, and DB_NAME."
  exit 1
fi

QUERY=$1

echo "Analyzing query: $QUERY"

EXPLAIN_OUTPUT=$(mysql -h"$DB_HOST" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "EXPLAIN $QUERY" --batch 2>/dev/null)

if [ $? -ne 0 ]; then
  echo "Error executing EXPLAIN on the query. Please check your query and database credentials."
  exit 1
fi

echo "--- EXPLAIN Output ---"
echo "$EXPLAIN_OUTPUT"
echo "----------------------"

SUGGESTIONS=""

# suggestion 1: Full table scan
if echo "$EXPLAIN_OUTPUT" | awk 'NR > 1' | awk '{print $5}' | grep -q "ALL"; then
    TABLE=$(echo "$EXPLAIN_OUTPUT" | awk 'NR > 1 && $5 == "ALL" {print $3}')
    SUGGESTIONS+="  - Consider adding an index to the join condition or WHERE clause for table '$TABLE' to avoid a full table scan.\n"
fi

# suggestion 2: Using filesort
if echo "$EXPLAIN_OUTPUT" | awk 'NR > 1' | awk '{print $10}' | grep -q "filesort"; then
    SUGGESTIONS+="  - 'Using filesort' indicates an inefficient sort. Consider adding an index on the columns used in the ORDER BY clause.\n"
fi

# suggestion 3: Using temporary
if echo "$EXPLAIN_OUTPUT" | awk 'NR > 1' | awk '{print $10}' | grep -q "temporary"; then
    SUGGESTIONS+="  - 'Using temporary' indicates the creation of a temporary table, which can be slow. This might be improved by adding an index.\n"
fi


if [ -z "$SUGGESTIONS" ]; then
  echo "No obvious performance issues found."
else
  echo "Actionable Suggestions:"
  echo -e "$SUGGESTIONS"
fi
