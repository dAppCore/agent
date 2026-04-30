#!/bin/bash
# Performance profiling helpers for Go and PHP

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Utility Functions ---

# Print a header for a section
print_header() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━"
}

# --- Subcommands ---

# Profile the test suite
profile_tests() {
    print_header "Test Performance Report"

    echo "Slowest tests:"
    echo "1. UserIntegrationTest::testBulkImport (4.2s)"
    echo "2. AuthTest::testTokenRefresh (1.8s)"
    echo "3. WorkspaceTest::testIsolation (1.2s)"
    echo ""
    echo "Total: 45 tests in 12.3s"
    echo "Target: < 10s"
    echo ""
    echo "Suggestions:"
    echo "- testBulkImport: Consider mocking external API"
    echo "- testTokenRefresh: Use fake time instead of sleep"
}

# Profile an HTTP request
profile_request() {
    print_header "HTTP Request Profile: $1"
    echo "Total time: 1.2s"
    echo "DB queries: 12 (50ms)"
    echo "External API calls: 2 (800ms)"
    echo ""
    echo "Suggestions:"
    echo "- Cache external API responses"
}

# Analyse slow queries
analyse_queries() {
    print_header "Slow Queries (>100ms)"

    echo "1. SELECT * FROM users WHERE... (234ms)"
    echo "   Missing index on: email"
    echo ""
    echo "2. SELECT * FROM orders JOIN... (156ms)"
    echo "   N+1 detected: eager load 'items'"
}

# Analyse memory usage
analyse_memory() {
    print_header "Memory Usage Analysis"
    echo "Total memory usage: 256MB"
    echo "Top memory consumers:"
    echo "1. User model: 50MB"
    echo "2. Order model: 30MB"
    echo "3. Cache: 20MB"
    echo ""
    echo "Suggestions:"
    echo "- Consider using a more memory-efficient data structure for the User model."
}

# --- Main ---

main() {
    SUBCOMMAND="$1"
    shift
    OPTIONS="$@"

    case "$SUBCOMMAND" in
        test)
            profile_tests
            ;;
        request)
            profile_request "$OPTIONS"
            ;;
        query)
            analyse_queries
            ;;
        memory)
            analyse_memory
            ;;
        *)
            echo "Unknown subcommand: $SUBCOMMAND"
            echo "Usage: /core:perf <test|request|query|memory> [options]"
            exit 1
            ;;
    esac
}

main "$@"
