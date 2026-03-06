#!/bin/bash
set -e # Exit on error

# --- Configuration ---
# Set to 1 to enable debug messages
DEBUG=0
debug() {
  if [ "$DEBUG" -eq 1 ]; then
    echo "DEBUG: $@" >&2
  fi
}

# --- Argument Parsing ---
COVERAGE=false
FILTER=""
ALL=false
# All other arguments are treated as files/directories to test
PATHS=()

while (( "$#" )); do
  case "$1" in
    --coverage)
      COVERAGE=true
      shift
      ;;
    --filter)
      if [ -n "$2" ] && [ "${2:0:1}" != "-" ]; then
        FILTER="$2"
        shift 2
      else
        echo "Error: Argument for --filter is missing" >&2
        exit 1
      fi
      ;;
    --all)
      ALL=true
      shift
      ;;
    -*)
      echo "Error: Unknown option $1" >&2
      exit 1
      ;;
    *) # preserve positional arguments as paths
      PATHS+=("$1")
      shift
      ;;
  esac
done

debug "Coverage: $COVERAGE"
debug "Filter: $FILTER"
debug "All: $ALL"
debug "Paths: ${PATHS[*]}"

# --- Project Detection ---
is_php() { [ -f "composer.json" ]; }
is_go() { [ -f "go.mod" ]; }

# --- Test Execution Functions ---
run_php_tests() {
  local args=("$@")
  local cmd=("core" "php" "test")
  if [ "$COVERAGE" = true ]; then
    cmd+=("--coverage")
  fi
  if [ -n "$FILTER" ]; then
    cmd+=("--filter" "$FILTER")
  fi
  if [ ${#args[@]} -gt 0 ]; then
    cmd+=("${args[@]}")
  fi
  echo "Running: ${cmd[*]}"
  "${cmd[@]}"
}

run_go_tests() {
  local args=("$@")
  local cmd_base=("core" "go")
  local cmd_action="test"

  if [ "$COVERAGE" = true ]; then
    cmd_action="cov"
  fi

  local cmd=("$cmd_base" "$cmd_action")

  if [ -n "$FILTER" ]; then
    cmd+=("--run" "$FILTER")
  fi

  if [ ${#args[@]} -gt 0 ]; then
     # For Go, convert file paths to package paths (./path/to/pkg)
    local pkgs=()
    for p in "${args[@]}"; do
      pkg=$(dirname "$p")
      # Add ./ prefix if not present and avoid duplicates
      if [[ ! " ${pkgs[@]} " =~ " ./$pkg " ]]; then
         pkgs+=("./$pkg")
      fi
    done
    cmd+=("${pkgs[@]}")
  else
    # If no paths specified, run for all packages
    cmd+=("./...")
  fi

  echo "Running: ${cmd[*]}"
  "${cmd[@]}"
}

# --- Main Logic ---

# If specific paths are provided, use them
if [ ${#PATHS[@]} -gt 0 ]; then
  echo "Running tests for specified paths..."
  if is_php; then run_php_tests "${PATHS[@]}";
  elif is_go; then run_go_tests "${PATHS[@]}";
  else echo "No PHP or Go project detected." >&2; exit 1; fi
  exit 0
fi

# If --all or --filter is used without paths, run against the whole project
if [ "$ALL" = true ] || [ -n "$FILTER" ]; then
  echo "Running all tests (--all or --filter specified)..."
  if is_php; then run_php_tests;
  elif is_go; then run_go_tests;
  else echo "No PHP or Go project detected." >&2; exit 1; fi
  exit 0
fi

# --- Smart Detection (default behavior) ---
echo "No specific options provided. Detecting changes since last commit..."
changed_files=$(git diff --name-only HEAD~1 HEAD)

if [ -z "$changed_files" ]; then
  echo "No changed files detected. Running all tests."
  if is_php; then run_php_tests;
  elif is_go; then run_go_tests;
  else echo "No PHP or Go project detected." >&2; exit 1; fi
  exit 0
fi

echo -e "Detected changed files:\n$changed_files"
test_files=()

for file in $changed_files; do
  if is_php && [[ "$file" == src/*.php ]]; then
    test_file=$(echo "$file" | sed 's|^src/|tests/|' | sed 's/\.php/Test.php/')
    [ -f "$test_file" ] && test_files+=("$test_file")
  elif is_go && [[ "$file" == *.go ]] && [[ "$file" != *_test.go ]]; then
    test_file="${file%.go}_test.go"
    [ -f "$test_file" ] && test_files+=("$test_file")
  fi
done

if [ ${#test_files[@]} -eq 0 ]; then
  echo "Could not map changed files to any existing tests. Running all tests as a fallback."
  if is_php; then run_php_tests;
  elif is_go; then run_go_tests;
  else echo "No PHP or Go project detected." >&2; exit 1; fi
else
  echo "Running tests for the following files:"
  printf " - %s\n" "${test_files[@]}"
  if is_php; then run_php_tests "${test_files[@]}";
  elif is_go; then run_go_tests "${test_files[@]}";
  else echo "No PHP or Go project detected." >&2; exit 1; fi
fi
