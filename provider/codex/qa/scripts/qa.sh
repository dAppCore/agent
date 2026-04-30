#!/bin/bash
# Core QA command logic

# --- Flags ---
FIX=false
QUICK=false
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --fix)
      FIX=true
      shift
      ;;
    --quick)
      QUICK=true
      shift
      ;;
    *)
      # Unknown arg, shift past it
      shift
      ;;
  esac
done

# --- Project Detection ---
PROJECT_TYPE=""
if [ -f "go.mod" ]; then
  PROJECT_TYPE="go"
elif [ -f "composer.json" ]; then
  PROJECT_TYPE="php"
else
  echo "Could not determine project type (go.mod or composer.json not found)."
  exit 1
fi

# --- QA Functions ---
run_qa() {
  if [ "$PROJECT_TYPE" = "go" ]; then
    core go qa
  else
    core php qa
  fi
}

run_lint() {
  if [ "$PROJECT_TYPE" = "go" ]; then
    core go lint
  else
    core php pint --test
  fi
}

run_fix() {
  if [ "$PROJECT_TYPE" = "go" ]; then
    core go fmt
  else
    core php pint
  fi
}

# --- Main Logic ---
if [ "$QUICK" = true ]; then
  echo "Running in --quick mode (lint only)..."
  run_lint
  exit $?
fi

echo "Running QA for $PROJECT_TYPE project..."
MAX_ITERATIONS=3
for i in $(seq 1 $MAX_ITERATIONS); do
  echo "--- Iteration $i ---"
  run_qa
  EXIT_CODE=$?

  if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ QA Passed"
    exit 0
  fi

  if [ "$FIX" = false ]; then
    echo "✗ QA Failed"
    exit $EXIT_CODE
  fi

  echo "QA failed. Attempting to fix..."
  run_fix
done

echo "✗ QA failed after $MAX_ITERATIONS iterations."
exit 1
