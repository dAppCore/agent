#!/bin/bash
set -e

# --- Argument Parsing ---
ARG="${1:-}"
PREVIEW=false
BUMP_LEVEL=""

if [[ "$ARG" == "--preview" ]]; then
  PREVIEW=true
  # Default to minor for preview, but allow specifying a level, e.g. --preview major
  BUMP_LEVEL="${2:-minor}"
else
  BUMP_LEVEL="$ARG"
fi

if [[ ! "$BUMP_LEVEL" =~ ^(patch|minor|major)$ ]]; then
  echo "Usage: /core:release <patch|minor|major|--preview> [level]"
  exit 1
fi

# --- Project Detection ---
CURRENT_VERSION=""
PROJECT_TYPE=""
VERSION_FILE=""
MODULE_NAME=""

if [ -f "composer.json" ]; then
  PROJECT_TYPE="php"
  VERSION_FILE="composer.json"
  MODULE_NAME=$(grep '"name":' "$VERSION_FILE" | sed -E 's/.*"name": "([^"]+)".*/\1/')
  CURRENT_VERSION=$(grep '"version":' "$VERSION_FILE" | sed -E 's/.*"version": "([^"]+)".*/\1/')
elif [ -f "go.mod" ]; then
  PROJECT_TYPE="go"
  VERSION_FILE="go.mod"
  MODULE_NAME=$(grep 'module' "$VERSION_FILE" | awk '{print $2}')
  CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
else
  echo "Error: No composer.json or go.mod found in the current directory."
  exit 1
fi

if [ -z "$CURRENT_VERSION" ]; then
    echo "Error: Could not determine current version for project type '$PROJECT_TYPE'."
    exit 1
fi

# --- Version Bumping ---
bump_version() {
  local version=$1
  local level=$2
  local parts=(${version//./ })
  local major=${parts[0]}
  local minor=${parts[1]}
  local patch=${parts[2]}

  case $level in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
  esac
  echo "$major.$minor.$patch"
}

NEW_VERSION=$(bump_version "$CURRENT_VERSION" "$BUMP_LEVEL")

# --- Changelog Generation ---
LAST_TAG="v$CURRENT_VERSION"
COMMITS=$(git log "$LAST_TAG..HEAD" --no-merges --pretty=format:"%s")

# Check if there are any commits since the last tag
if [ -z "$COMMITS" ]; then
  echo "No changes since the last release ($LAST_TAG). Nothing to do."
  exit 0
fi

declare -A changes
while IFS= read -r commit; do
  if [[ "$commit" =~ ^(feat|fix|docs)(\(.*\))?:\ .* ]]; then
    type=$(echo "$commit" | sed -E 's/^(feat|fix|docs).*/\1/')
    message=$(echo "$commit" | sed -E 's/^(feat|fix|docs)(\(.*\))?:\ //')
    case $type in
      feat) changes["Added"]+="- $message\n";;
      fix)  changes["Fixed"]+="- $message\n";;
      docs) changes["Documentation"]+="- $message\n";;
    esac
  fi
done <<< "$COMMITS"

CHANGELOG_ENTRY="## [$NEW_VERSION] - $(date +%Y-%m-%d)\n\n"
for type in Added Fixed Documentation; do
  if [ -n "${changes[$type]}" ]; then
    CHANGELOG_ENTRY+="### $type\n${changes[$type]}\n"
  fi
done

# --- Display Plan ---
echo "Preparing release: $MODULE_NAME v$CURRENT_VERSION → v$NEW_VERSION"
echo ""
echo "Changes since $LAST_TAG:"
echo "$COMMITS" | sed 's/^/- /'
echo ""
echo "Generated CHANGELOG entry:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "$CHANGELOG_ENTRY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# --- Execution ---
if [ "$PREVIEW" = true ]; then
  echo "Running in preview mode. No files will be changed and no tags will be pushed."
  exit 0
fi

echo "Proceed with release? [y/N]"
read -r confirmation

if [[ ! "$confirmation" =~ ^[yY]$ ]]; then
    echo "Release cancelled."
    exit 1
fi

# 1. Update version file
if [ "$PROJECT_TYPE" == "php" ]; then
    sed -i -E "s/(\"version\": *)\"[^\"]+\"/\1\"$NEW_VERSION\"/" "$VERSION_FILE"
    echo "Updated $VERSION_FILE to v$NEW_VERSION"
fi

# 2. Update CHANGELOG.md
if [ ! -f "CHANGELOG.md" ]; then
  echo "# Changelog" > CHANGELOG.md
  echo "" >> CHANGELOG.md
fi
# Prepend the new entry
NEW_CHANGELOG_CONTENT=$(echo -e "$CHANGELOG_ENTRY" && cat CHANGELOG.md)
echo -e "$NEW_CHANGELOG_CONTENT" > CHANGELOG.md
echo "Updated CHANGELOG.md"

# 3. Commit the changes
git add "$VERSION_FILE" CHANGELOG.md
git commit -m "chore(release): version $NEW_VERSION"

# 4. Create and push git tag
NEW_TAG="v$NEW_VERSION"
git tag "$NEW_TAG"
echo "Created new git tag: $NEW_TAG"

# 5. Push tag and changes
git push origin "$NEW_TAG"
git push
echo "Pushed tag and commit to remote."

# 6. Trigger CI release (placeholder)
