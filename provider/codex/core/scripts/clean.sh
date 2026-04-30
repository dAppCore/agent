#!/bin/bash

# core:clean script
# Cleans generated files, caches, and build artifacts.

# --- Configuration ---
CACHE_PATHS=(
  "storage/framework/cache"
  "bootstrap/cache"
  ".phpunit.cache"
)
BUILD_PATHS=(
  "public/build"
  "public/hot"
)
DEP_PATHS=(
  "vendor"
  "node_modules"
)

# --- Argument Parsing ---
CLEAN_CACHE=false
CLEAN_BUILD=false
CLEAN_DEPS=false
FORCE=false
DRY_RUN=false
ACTION_SPECIFIED=false

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --cache) CLEAN_CACHE=true; ACTION_SPECIFIED=true; shift ;;
        --deps) CLEAN_DEPS=true; ACTION_SPECIFIED=true; shift ;;
        --force) FORCE=true; shift ;;
        --dry-run) DRY_RUN=true; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
done

if [ "$ACTION_SPECIFIED" = false ]; then
  CLEAN_CACHE=true
  CLEAN_BUILD=true
fi

if [ "$DRY_RUN" = true ]; then
  FORCE=false
fi

# --- Functions ---
get_size() {
  du -sb "$1" 2>/dev/null | cut -f1
}

format_size() {
  local size=$1
  if [ -z "$size" ] || [ "$size" -eq 0 ]; then
    echo "0 B"
    return
  fi
  if (( size < 1024 )); then
    echo "${size} B"
  elif (( size < 1048576 )); then
    echo "$((size / 1024)) KB"
  else
    echo "$((size / 1048576)) MB"
  fi
}

# --- Main Logic ---
TOTAL_FREED=0
echo "Cleaning core-tenant..."
echo

# Cache cleanup
if [ "$CLEAN_CACHE" = true ]; then
  echo "Cache:"
  for path in "${CACHE_PATHS[@]}"; do
    if [ -e "$path" ]; then
      SIZE=$(get_size "$path")
      if [ "$DRY_RUN" = true ]; then
        echo "  - $path ($(format_size "$SIZE")) would be cleared"
      else
        rm -rf "${path:?}"/*
        echo "  ✓ $path cleared"
        TOTAL_FREED=$((TOTAL_FREED + SIZE))
      fi
    fi
  done
  echo
fi

# Build cleanup
if [ "$CLEAN_BUILD" = true ]; then
  echo "Build:"
  for path in "${BUILD_PATHS[@]}"; do
    if [ -e "$path" ]; then
      SIZE=$(get_size "$path")
      if [ "$DRY_RUN" = true ]; then
        echo "  - $path ($(format_size "$SIZE")) would be deleted"
      else
        rm -rf "$path"
        echo "  ✓ $path deleted"
        TOTAL_FREED=$((TOTAL_FREED + SIZE))
      fi
    fi
  done
  echo
fi

# Dependency cleanup
if [ "$CLEAN_DEPS" = true ]; then
  DEPS_SIZE=0
  DEPS_TO_DELETE=()
  for path in "${DEP_PATHS[@]}"; do
      if [ -d "$path" ]; then
          DEPS_TO_DELETE+=("$path")
          SIZE=$(get_size "$path")
          DEPS_SIZE=$((DEPS_SIZE + SIZE))
      fi
  done

  echo "Dependencies:"
  if [ ${#DEPS_TO_DELETE[@]} -eq 0 ]; then
      echo "  No dependency directories found."
  elif [ "$FORCE" = false ] || [ "$DRY_RUN" = true ]; then
      echo "This is a dry-run. Use --deps --force to delete."
      for path in "${DEPS_TO_DELETE[@]}"; do
          echo "  - $path ($(format_size "$(get_size "$path")")) would be deleted"
      done
  else
      echo "The following directories will be permanently deleted:"
      for path in "${DEPS_TO_DELETE[@]}"; do
          echo "  - $path ($(format_size "$(get_size "$path")"))"
      done
      echo
      read -p "Are you sure? [y/N] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
          for path in "${DEPS_TO_DELETE[@]}"; do
              rm -rf "$path"
              echo "  ✓ $path deleted."
          done
          TOTAL_FREED=$((TOTAL_FREED + DEPS_SIZE))
      else
          echo "Aborted by user."
      fi
  fi
  echo
fi

echo "Total freed: $(format_size "$TOTAL_FREED")"
