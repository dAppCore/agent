#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <url>"
  exit 1
fi

URL=$1

echo "Profiling request to: $URL"

OUTPUT=$(curl -w "time_namelookup=%{time_namelookup}\ntime_connect=%{time_connect}\ntime_appconnect=%{time_appconnect}\ntime_pretransfer=%{time_pretransfer}\ntime_redirect=%{time_redirect}\ntime_starttransfer=%{time_starttransfer}\ntime_total=%{time_total}" -o /dev/null -s "$URL")

# Extract values
get_value() {
    echo "$OUTPUT" | grep "$1" | cut -d'=' -f2
}

TIME_NAMELOOKUP=$(get_value time_namelookup)
TIME_CONNECT=$(get_value time_connect)
TIME_STARTTRANSFER=$(get_value time_starttransfer)

echo "--- Timing Metrics ---"
echo "DNS Lookup:      ${TIME_NAMELOOKUP}s"
echo "Connect:         ${TIME_CONNECT}s"
echo "Start Transfer:  ${TIME_STARTTRANSFER}s"
echo "----------------------"

SUGGESTIONS=""

# Suggestion 1: High DNS lookup time
if (( $(echo "$TIME_NAMELOOKUP > 0.1" | bc -l) )); then
    SUGGESTIONS+="  - DNS lookup took over 100ms. Consider using a faster DNS provider or checking your network configuration.\n"
fi

# Suggestion 2: High connect time
if (( $(echo "$TIME_CONNECT > 0.2" | bc -l) )); then
    SUGGESTIONS+="  - Connection time is over 200ms. If this is a remote server, consider using a CDN. If it's local, check for network latency or server load.\n"
fi

# Suggestion 3: High start transfer time (Time To First Byte)
if (( $(echo "$TIME_STARTTRANSFER > 0.5" | bc -l) )); then
    SUGGESTIONS+="  - Time To First Byte (TTFB) is over 500ms. This indicates a slow backend. Profile your application code to identify and optimize bottlenecks.\n"
fi

if [ -z "$SUGGESTIONS" ]; then
  echo "No obvious performance issues found."
else
  echo "Actionable Suggestions:"
  echo -e "$SUGGESTIONS"
fi
