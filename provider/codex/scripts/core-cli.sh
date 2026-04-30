#!/bin/bash
set -euo pipefail

cat <<'MAP'
Core CLI mapping:
- go test -> core go test
- go build -> core build
- go fmt -> core go fmt
- composer test -> core php test
- ./vendor/bin/pint -> core php fmt
MAP
