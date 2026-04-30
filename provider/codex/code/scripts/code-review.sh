#!/bin/bash
# Core code review script

# --- Result Variables ---
conventions_result=""
debug_result=""
test_coverage_result=""
secrets_result=""
error_handling_result=""
docs_result=""
intensive_security_result=""
suggestions=()

# --- Check Functions ---

check_conventions() {
  # Placeholder for project convention checks (e.g., linting)
  conventions_result="✓ Conventions: UK English, strict types (Placeholder)"
}

check_debug() {
  local diff_content=$1
  if echo "$diff_content" | grep -q -E 'console\.log|print_r|var_dump'; then
    debug_result="⚠ No debug statements: Found debug statements."
    suggestions+=("Remove debug statements before merging.")
  else
    debug_result="✓ No debug statements"
  fi
}

check_test_coverage() {
  local diff_content=$1
  # This is a simple heuristic and not a replacement for a full test coverage suite.
  # It checks if any new files are tests, or if test files were modified.
  if echo "$diff_content" | grep -q -E '\+\+\+ b/(tests?|specs?)/'; then
    test_coverage_result="✓ Test files modified: Yes"
  else
    test_coverage_result="⚠ Test files modified: No"
    suggestions+=("Consider adding tests for new functionality.")
  fi
}

check_secrets() {
  local diff_content=$1
  if echo "$diff_content" | grep -q -i -E 'secret|password|api_key|token'; then
    secrets_result="⚠ No secrets detected: Potential hardcoded secrets found."
    suggestions+=("Review potential hardcoded secrets for security.")
  else
    secrets_result="✓ No secrets detected"
  fi
}

intensive_security_check() {
  local diff_content=$1
  if echo "$diff_content" | grep -q -E 'eval|dangerouslySetInnerHTML'; then
    intensive_security_result="⚠ Intensive security scan: Unsafe functions may be present."
    suggestions+=("Thoroughly audit the use of unsafe functions.")
  else
    intensive_security_result="✓ Intensive security scan: No obvious unsafe functions found."
  fi
}

check_error_handling() {
    local diff_content=$1
    # Files with new functions/methods but no error handling
    local suspicious_files=$(echo "$diff_content" | grep -E '^\+\+\+ b/' | sed 's/^\+\+\+ b\///' | while read -r file; do
        # Heuristic: if a file has added lines with 'function' or '=>' but no 'try'/'catch', it's suspicious.
        added_logic=$(echo "$diff_content" | grep -E "^\+.*(function|\=>)" | grep "$file")
        added_error_handling=$(echo "$diff_content" | grep -E "^\+.*(try|catch|throw)" | grep "$file")

        if [ -n "$added_logic" ] && [ -z "$added_error_handling" ]; then
            line_number=$(echo "$diff_content" | grep -nE "^\+.*(function|\=>)" | grep "$file" | cut -d: -f1 | head -n 1)
            echo "$file:$line_number"
        fi
    done)

    if [ -n "$suspicious_files" ]; then
        error_handling_result="⚠ Missing error handling"
        for file_line in $suspicious_files; do
            suggestions+=("Consider adding error handling in $file_line.")
        done
    else
        error_handling_result="✓ Error handling present"
    fi
}

check_docs() {
    local diff_content=$1
    if echo "$diff_content" | grep -q -E '\+\+\+ b/(README.md|docs?)/'; then
        docs_result="✓ Documentation updated"
    else
        docs_result="⚠ Documentation updated: No changes to documentation files detected."
        suggestions+=("Update documentation if the changes affect public APIs or user behavior.")
    fi
}

# --- Output Function ---

print_results() {
  local title="Code Review"
  if [ -n "$range_arg" ]; then
    title="$title: $range_arg"
  else
    local branch_name=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
    if [ -n "$branch_name" ]; then
        title="$title: $branch_name branch"
    else
        title="$title: Staged changes"
    fi
  fi

  echo "$title"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""

  # Print checklist
  echo "$conventions_result"
  echo "$debug_result"
  echo "$test_coverage_result"
  echo "$secrets_result"
  echo "$error_handling_result"
  echo "$docs_result"
  if [ -n "$intensive_security_result" ]; then
    echo "$intensive_security_result"
  fi
  echo ""

  # Print suggestions if any
  if [ ${#suggestions[@]} -gt 0 ]; then
    echo "Suggestions:"
    for i in "${!suggestions[@]}"; do
      echo "$((i+1)). ${suggestions[$i]}"
    done
    echo ""
  fi

  echo "Overall: Approve with suggestions"
}

# --- Main Logic ---
security_mode=false
range_arg=""

for arg in "$@"; do
  case $arg in
    --security)
      security_mode=true
      ;;
    *)
      if [ -n "$range_arg" ]; then echo "Error: Multiple range arguments." >&2; exit 1; fi
      range_arg="$arg"
      ;;
  esac
done

diff_output=""
if [ -z "$range_arg" ]; then
  diff_output=$(git diff --staged)
  if [ $? -ne 0 ]; then echo "Error: git diff --staged failed." >&2; exit 1; fi
  if [ -z "$diff_output" ]; then echo "No staged changes to review."; exit 0; fi
elif [[ "$range_arg" == \#* ]]; then
  pr_number="${range_arg#?}"
  if ! command -v gh &> /dev/null; then echo "Error: 'gh' not found." >&2; exit 1; fi
  diff_output=$(gh pr diff "$pr_number")
  if [ $? -ne 0 ]; then echo "Error: gh pr diff failed. Is the PR number valid?" >&2; exit 1; fi
elif [[ "$range_arg" == *..* ]]; then
  diff_output=$(git diff "$range_arg")
  if [ $? -ne 0 ]; then echo "Error: git diff failed. Is the commit range valid?" >&2; exit 1; fi
else
  echo "Unsupported argument: $range_arg" >&2
  exit 1
fi

# Run checks
check_conventions
check_debug "$diff_output"
check_test_coverage "$diff_output"
check_error_handling "$diff_output"
check_docs "$diff_output"
check_secrets "$diff_output"

if [ "$security_mode" = true ]; then
  intensive_security_check "$diff_output"
fi

# Print the final formatted report
print_results
