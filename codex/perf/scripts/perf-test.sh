#!/bin/bash

set -e

if [ -f "go.mod" ]; then
  PROJECT_TYPE="go"
elif [ -f "composer.json" ]; then
  PROJECT_TYPE="php"
else
  echo "Error: Unable to determine project type. No go.mod or composer.json found."
  exit 1
fi

echo "Detected project type: $PROJECT_TYPE"

case $PROJECT_TYPE in
  "go")
    echo "Running Go test profiling..."
    go test -v -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. 2>&1 | tee test_output.log

    echo "Analyzing test performance..."
    grep "--- PASS" test_output.log | awk '{print $4, $3}' | sort -nr | head -n 10 > slowest_tests.log

    echo "Slowest tests:"
    cat slowest_tests.log

    echo ""
    echo "Actionable Suggestions:"
    awk '$1 > 2.0 {print "  - The test \""$2"\" took " $1 "s to run. Consider using mocks for external dependencies to speed it up."}' slowest_tests.log
    ;;
  "php")
    if ! php -m | grep -q 'Xdebug'; then
      echo "Xdebug is not installed. Please install it to use the PHP test profiler."
      exit 1
    fi

    echo "Running PHP test profiling..."
    if [ -f "vendor/bin/pest" ]; then
      vendor/bin/pest --log-junit report.xml
    elif [ -f "vendor/bin/phpunit" ]; then
      vendor/bin/phpunit --log-junit report.xml
    else
        echo "Error: No pest or phpunit executable found."
        exit 1
    fi

    if ! command -v xmlstarlet &> /dev/null; then
      echo "xmlstarlet could not be found. Please install it to use this feature."
      echo "On Debian/Ubuntu: sudo apt-get install xmlstarlet"
      echo "On macOS (Homebrew): brew install xmlstarlet"
      exit 1
    fi

    echo "Analyzing test performance..."
    xmlstarlet sel -t -m "//testcase" -v "@time" -o " " -v "@name" -n report.xml | sort -nr | head -n 10 > slowest_tests.log

    echo "Slowest tests:"
    cat slowest_tests.log

    echo ""
    echo "Actionable Suggestions:"
    awk '$1 > 2.0 {print "  - The test \""$2"\" took " $1 "s to run. Consider using mocks for external dependencies to speed it up."}' slowest_tests.log
    ;;
esac
