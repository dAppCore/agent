#!/bin/bash

TARGET_PATH=$1

if [ -z "$TARGET_PATH" ]; then
  echo "Usage: doc-changelog.sh <TargetPath>" >&2
  exit 1
fi

# We must be in the target directory for git commands to work correctly.
cd "$TARGET_PATH"

# Get the latest tag. If no tags, this will be empty.
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
# Get the date of the latest tag.
TAG_DATE=$(git log -1 --format=%ai "$LATEST_TAG" 2>/dev/null | cut -d' ' -f1)

# Set the version to the latest tag, or "Unreleased" if no tags exist.
VERSION="Unreleased"
if [ -n "$LATEST_TAG" ]; then
  VERSION="$LATEST_TAG"
fi

# Get the current date in YYYY-MM-DD format.
CURRENT_DATE=$(date +%F)
DATE_TO_SHOW=$CURRENT_DATE
if [ -n "$TAG_DATE" ]; then
    DATE_TO_SHOW="$TAG_DATE"
fi

echo "# Changelog"
echo ""
echo "## [$VERSION] - $DATE_TO_SHOW"
echo ""

# Get the commit history. If there's a tag, get commits since the tag. Otherwise, get all.
if [ -n "$LATEST_TAG" ]; then
  COMMIT_RANGE="${LATEST_TAG}..HEAD"
else
  COMMIT_RANGE="HEAD"
fi

# Use git log to get commits, then awk to categorize and format them.
# Categories are based on the commit subject prefix (e.g., "feat:", "fix:").
git log --no-merges --pretty="format:%s" "$COMMIT_RANGE" | awk '
  BEGIN {
    FS = ": ";
    print_added = 0;
    print_fixed = 0;
  }
  /^feat:/ {
    if (!print_added) {
      print "### Added";
      print_added = 1;
    }
    print "- " $2;
  }
  /^fix:/ {
    if (!print_fixed) {
      print "";
      print "### Fixed";
      print_fixed = 1;
    }
    print "- " $2;
  }
'
