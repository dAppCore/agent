#!/bin/bash
set -euo pipefail

cat <<'SAFE'
Safety guardrails:
- Avoid rm -rf / rm -r (except node_modules, vendor, .cache)
- Avoid sed -i
- Avoid xargs with file operations
- Avoid mv/cp with wildcards
SAFE
