#!/bin/bash

# Default options
CLEAN_DEPS=false
CLEAN_CACHE_ONLY=false
DRY_RUN=false

# Parse arguments
for arg in "$@"
do
    case $arg in
        --deps)
        CLEAN_DEPS=true
        shift
        ;;
        --cache)
        CLEAN_CACHE_ONLY=true
        shift
        ;;
        --dry-run)
        DRY_RUN=true
        shift
        ;;
    esac
done

# --- Configuration ---
CACHE_PATHS=(
  "storage/framework/cache/*"
  "bootstrap/cache/*"
  ".phpunit.cache"
)

BUILD_PATHS=(
  "public/build/*"
  "public/hot"
)

DEP_PATHS=(
  "vendor"
  "node_modules"
)

# --- Logic ---
total_freed=0

delete_path() {
  local path_pattern=$1
  local size_bytes=0
  local size_human=""

  # Use a subshell to avoid affecting the main script's globbing settings
  (
    shopt -s nullglob
    local files=( $path_pattern )

    if [ ${#files[@]} -eq 0 ]; then
        return # No files matched the glob
    fi

    # Calculate total size for all matched files
    for file in "${files[@]}"; do
        if [ -e "$file" ]; then
            size_bytes=$((size_bytes + $(du -sb "$file" | cut -f1)))
        fi
    done
  )

  total_freed=$((total_freed + size_bytes))
  size_human=$(echo "$size_bytes" | awk '{
      if ($1 >= 1024*1024*1024) { printf "%.2f GB", $1/(1024*1024*1024) }
      else if ($1 >= 1024*1024) { printf "%.2f MB", $1/(1024*1024) }
      else if ($1 >= 1024) { printf "%.2f KB", $1/1024 }
      else { printf "%d Bytes", $1 }
  }')


  if [ "$DRY_RUN" = true ]; then
    echo "  ✓ (dry run) $path_pattern ($size_human)"
  else
    # Suppress "no such file or directory" errors if glob doesn't match anything
    rm -rf $path_pattern 2>/dev/null
    echo "  ✓ $path_pattern ($size_human)"
  fi
}


echo "Cleaning project..."
echo ""

if [ "$CLEAN_CACHE_ONLY" = true ]; then
  echo "Cache:"
  for path in "${CACHE_PATHS[@]}"; do
    delete_path "$path"
  done
else
  echo "Cache:"
  for path in "${CACHE_PATHS[@]}"; do
    delete_path "$path"
  done
  echo ""
  echo "Build:"
  for path in "${BUILD_PATHS[@]}"; do
    delete_path "$path"
  done
fi

if [ "$CLEAN_DEPS" = true ]; then
  if [ "$DRY_RUN" = false ]; then
    echo ""
    read -p "Delete vendor/ and node_modules/? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
  fi
  echo ""
  echo "Dependencies (--deps):"
  for path in "${DEP_PATHS[@]}"; do
    delete_path "$path"
  done
fi

# Final summary
if [ "$total_freed" -gt 0 ]; then
    total_freed_human=$(echo "$total_freed" | awk '{
        if ($1 >= 1024*1024*1024) { printf "%.2f GB", $1/(1024*1024*1024) }
        else if ($1 >= 1024*1024) { printf "%.2f MB", $1/(1024*1024) }
        else if ($1 >= 1024) { printf "%.2f KB", $1/1024 }
        else { printf "%d Bytes", $1 }
    }')
    echo ""
    echo "Total freed: $total_freed_human"
fi
