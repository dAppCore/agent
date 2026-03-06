#!/bin/bash

# Exit on error
set -e

# --- Argument Parsing ---
BUMP_TYPE=""
PREVIEW=false

# Loop through all arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        patch|minor|major)
            if [ -n "$BUMP_TYPE" ]; then
                echo "Error: Only one version bump type (patch, minor, major) can be specified." >&2
                exit 1
            fi
            BUMP_TYPE="$1"
            ;;
        --preview)
            PREVIEW=true
            ;;
        *)
            # Stop parsing at the first non-recognized argument, assuming it's for another script
            break
            ;;
    esac
    shift
done

# --- Validation ---
if [ -z "$BUMP_TYPE" ]; then
    echo "Usage: /core:release [patch|minor|major] [--preview]"
    echo "Error: A version bump type must be specified." >&2
    exit 1
fi

echo "Configuration:"
echo "  Bump Type: $BUMP_TYPE"
echo "  Preview Mode: $PREVIEW"
echo "-------------------------"

# --- Version Calculation ---
PACKAGE_JSON_PATH="package.json"

if [ ! -f "$PACKAGE_JSON_PATH" ]; then
    echo "Error: package.json not found in the current directory." >&2
    exit 1
fi

CURRENT_VERSION=$(jq -r '.version' "$PACKAGE_JSON_PATH")
echo "Current version: $CURRENT_VERSION"

# Version bumping logic
IFS='.' read -r -a VERSION_PARTS <<< "$CURRENT_VERSION"
MAJOR=${VERSION_PARTS[0]}
MINOR=${VERSION_PARTS[1]}
PATCH=${VERSION_PARTS[2]}

case $BUMP_TYPE in
    patch)
        PATCH=$((PATCH + 1))
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"
echo "New version: $NEW_VERSION"

# --- Changelog Generation ---
# Get the latest tag, or the first commit if no tags exist
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)

# Get commit messages since the last tag
COMMIT_LOG=$(git log "$LATEST_TAG"..HEAD --pretty=format:"%s")

# Prepare changelog entry
CHANGELOG_ENTRY="## [$NEW_VERSION] - $(date +%Y-%m-%d)\n\n"
ADDED=""
FIXED=""
OTHER=""

while IFS= read -r line; do
    if [[ "$line" == feat* ]]; then
        ADDED="$ADDED- ${line#*: }\n"
    elif [[ "$line" == fix* ]]; then
        FIXED="$FIXED- ${line#*: }\n"
    else
        OTHER="$OTHER- $line\n"
    fi
done <<< "$COMMIT_LOG"

if [ -n "$ADDED" ]; then
    CHANGELOG_ENTRY="${CHANGELOG_ENTRY}### Added\n$ADDED\n"
fi

if [ -n "$FIXED" ]; then
    CHANGELOG_ENTRY="${CHANGELOG_ENTRY}### Fixed\n$FIXED\n"
fi

if [ -n "$OTHER" ]; then
    CHANGELOG_ENTRY="${CHANGELOG_ENTRY}### Other\n$OTHER\n"
fi

echo -e "\nGenerated CHANGELOG entry:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━\n$CHANGELOG_ENTRY━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# --- Execution ---
GEMINI_EXTENSION_PATH="gemini-extension.json"
CHANGELOG_PATH="CHANGELOG.md"

if [ "$PREVIEW" = true ]; then
    echo -e "\nPreview mode enabled. The following actions would be taken:"
    echo "  - Bump version in $PACKAGE_JSON_PATH to $NEW_VERSION"
    echo "  - Bump version in $GEMINI_EXTENSION_PATH to $NEW_VERSION (if it exists)"
    echo "  - Prepend the generated entry to $CHANGELOG_PATH"
    echo "  - git add (conditionally including $GEMINI_EXTENSION_PATH if it exists)"
    echo "  - git commit -m \"chore(release): v$NEW_VERSION\""
    echo "  - git tag v$NEW_VERSION"
    echo "  - git push origin HEAD --follow-tags"
    exit 0
fi

read -p "Proceed with release? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Release cancelled."
    exit 1
fi

echo "Proceeding with release..."

# Update package.json
jq --arg v "$NEW_VERSION" '.version = $v' "$PACKAGE_JSON_PATH" > tmp.$$.json && mv tmp.$$.json "$PACKAGE_JSON_PATH"
echo "Updated $PACKAGE_JSON_PATH"

# Update gemini-extension.json
if [ -f "$GEMINI_EXTENSION_PATH" ]; then
    jq --arg v "$NEW_VERSION" '.version = $v' "$GEMINI_EXTENSION_PATH" > tmp.$$.json && mv tmp.$$.json "$GEMINI_EXTENSION_PATH"
    echo "Updated $GEMINI_EXTENSION_PATH"
fi

# Update CHANGELOG.md
if [ ! -f "$CHANGELOG_PATH" ]; then
    echo -e "# Changelog\n\nAll notable changes to this project will be documented in this file.\n" > "$CHANGELOG_PATH"
fi
echo -e "$CHANGELOG_ENTRY$(cat $CHANGELOG_PATH)" > "$CHANGELOG_PATH"
echo "Updated $CHANGELOG_PATH"

# Git operations
echo "Committing changes..."
FILES_TO_ADD=("$PACKAGE_JSON_PATH" "$CHANGELOG_PATH")
if [ -f "$GEMINI_EXTENSION_PATH" ]; then
    FILES_TO_ADD+=("$GEMINI_EXTENSION_PATH")
fi
git add "${FILES_TO_ADD[@]}"
git commit -m "chore(release): v$NEW_VERSION"

echo "Creating git tag..."
git tag "v$NEW_VERSION"

echo "Pushing changes and tag..."
git push origin HEAD --follow-tags

echo "Release complete."
