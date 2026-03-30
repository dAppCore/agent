---
name: flow-gather-training-data
description: Use when capturing training data from completed flows. Records structural signals (IDs, timestamps, SHAs) to JSONL journals for model training.
---

# Flow: Gather Training Data

Continuously capture PR/issue state observations for training the agentic orchestrator model.

---

## Purpose

Build a time-series dataset of:
1. **Input signals** - PR state, CI status, review counts, timing
2. **Actions taken** - what the orchestrator decided
3. **Outcomes** - did it work? how long to resolution?

This enables training a model to predict correct actions from signals alone.

---

## Infrastructure

### InfluxDB Setup

```bash
# Install (Ubuntu 24.04)
curl -sL https://repos.influxdata.com/influxdata-archive.key | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/influxdata-archive.gpg
echo "deb [signed-by=/etc/apt/trusted.gpg.d/influxdata-archive.gpg] https://repos.influxdata.com/ubuntu noble stable" | sudo tee /etc/apt/sources.list.d/influxdata.list
sudo apt-get update && sudo apt-get install -y influxdb2 influxdb2-cli

# Start service
sudo systemctl enable influxdb --now

# Initial setup (interactive)
influx setup \
  --org agentic \
  --bucket training \
  --username claude \
  --password <password> \
  --force

# Create API token for writes
influx auth create --org agentic --write-bucket training --description "training-data-capture"
```

Store the token in `~/.influx_token` (chmod 600).

### Schema (InfluxDB Line Protocol)

```
# Measurement: pr_observation
pr_observation,repo=dappcore/core,pr=315,author=jules[bot] \
  merge_state="CLEAN",mergeable=true,is_draft=false,\
  checks_total=8i,checks_passing=8i,checks_failing=0i,\
  reviews_approved=1i,reviews_changes_requested=0i,\
  threads_total=5i,threads_unresolved=0i,\
  pr_age_hours=48i,last_push_hours=2i,\
  conflict_attempts=0i,review_fix_attempts=0i \
  1707123600000000000

# Measurement: action_taken
action_taken,repo=dappcore/core,pr=315 \
  action="wait",reason="auto-merge enabled, checks passing" \
  1707123600000000000

# Measurement: outcome
outcome,repo=dappcore/core,pr=315 \
  result="success",detail="merged via auto-merge",resolution_hours=0.5 \
  1707125400000000000
```

---

## Capture Script

Location: `~/infra/tasks-agentic/training-data/capture-to-influx.sh`

```bash
#!/bin/bash
# capture-to-influx.sh - Capture PR states to InfluxDB
set -euo pipefail

INFLUX_HOST="${INFLUX_HOST:-http://localhost:8086}"
INFLUX_ORG="${INFLUX_ORG:-agentic}"
INFLUX_BUCKET="${INFLUX_BUCKET:-training}"
INFLUX_TOKEN="${INFLUX_TOKEN:-$(cat ~/.influx_token 2>/dev/null)}"
REPO="${1:-dappcore/core}"

capture_pr_to_influx() {
    local repo=$1
    local pr=$2
    local timestamp
    timestamp=$(date +%s%N)

    # Get PR data
    local data
    data=$(gh pr view "$pr" --repo "$repo" --json \
        number,mergeable,mergeStateStatus,statusCheckRollup,\
latestReviews,reviewDecision,labels,author,createdAt,updatedAt,\
commits,autoMergeRequest,isDraft 2>/dev/null)

    # Extract fields
    local merge_state=$(echo "$data" | jq -r '.mergeStateStatus // "UNKNOWN"')
    local mergeable=$(echo "$data" | jq -r 'if .mergeable == "MERGEABLE" then "true" else "false" end')
    local is_draft=$(echo "$data" | jq -r '.isDraft // false')
    local checks_total=$(echo "$data" | jq '[.statusCheckRollup[]? | select(.name != null)] | length')
    local checks_passing=$(echo "$data" | jq '[.statusCheckRollup[]? | select(.conclusion == "SUCCESS")] | length')
    local checks_failing=$(echo "$data" | jq '[.statusCheckRollup[]? | select(.conclusion == "FAILURE")] | length')
    local reviews_approved=$(echo "$data" | jq '[.latestReviews[]? | select(.state == "APPROVED")] | length')
    local reviews_changes=$(echo "$data" | jq '[.latestReviews[]? | select(.state == "CHANGES_REQUESTED")] | length')
    local author=$(echo "$data" | jq -r '.author.login // "unknown"')
    local auto_merge=$(echo "$data" | jq -r 'if .autoMergeRequest != null then "true" else "false" end')

    # Calculate ages
    local created=$(echo "$data" | jq -r '.createdAt')
    local updated=$(echo "$data" | jq -r '.updatedAt')
    # NOTE: date -d is GNU (Linux). On macOS use: date -j -f "%Y-%m-%dT%H:%M:%SZ" "$created" +%s
    local pr_age_hours=$(( ($(date +%s) - $(date -d "$created" +%s)) / 3600 ))
    local last_activity_hours=$(( ($(date +%s) - $(date -d "$updated" +%s)) / 3600 ))

    # Build line protocol
    local line="pr_observation,repo=${repo//\//_},pr=${pr},author=${author} "
    line+="merge_state=\"${merge_state}\","
    line+="mergeable=${mergeable},"
    line+="is_draft=${is_draft},"
    line+="checks_total=${checks_total}i,"
    line+="checks_passing=${checks_passing}i,"
    line+="checks_failing=${checks_failing}i,"
    line+="reviews_approved=${reviews_approved}i,"
    line+="reviews_changes_requested=${reviews_changes}i,"
    line+="auto_merge_enabled=${auto_merge},"
    line+="pr_age_hours=${pr_age_hours}i,"
    line+="last_activity_hours=${last_activity_hours}i "
    line+="${timestamp}"

    # Write to InfluxDB
    curl -s -XPOST "${INFLUX_HOST}/api/v2/write?org=${INFLUX_ORG}&bucket=${INFLUX_BUCKET}&precision=ns" \
        -H "Authorization: Token ${INFLUX_TOKEN}" \
        -H "Content-Type: text/plain" \
        --data-raw "$line"

    echo "Captured PR #${pr}"
}

# Capture all open PRs
for pr in $(gh pr list --repo "$REPO" --state open --json number --jq '.[].number'); do
    capture_pr_to_influx "$REPO" "$pr"
done
```

---

## Cron Schedule

```bash
# Add to crontab -e
# Capture every 15 minutes
*/15 * * * * /home/claude/infra/tasks-agentic/training-data/capture-to-influx.sh dappcore/core >> /home/claude/logs/training-capture.log 2>&1

# Also capture PHP repos hourly (lower priority)
0 * * * * /home/claude/infra/tasks-agentic/training-data/capture-to-influx.sh dappcore/core-php >> /home/claude/logs/training-capture.log 2>&1
0 * * * * /home/claude/infra/tasks-agentic/training-data/capture-to-influx.sh dappcore/core-mcp >> /home/claude/logs/training-capture.log 2>&1
0 * * * * /home/claude/infra/tasks-agentic/training-data/capture-to-influx.sh dappcore/core-api >> /home/claude/logs/training-capture.log 2>&1
```

---

## Recording Actions & Outcomes

### When Orchestrator Takes Action

After any orchestration action, record it:

```bash
record_action() {
    local repo=$1 pr=$2 action=$3 reason=$4
    local timestamp=$(date +%s%N)
    local line="action_taken,repo=${repo//\//_},pr=${pr} action=\"${action}\",reason=\"${reason}\" ${timestamp}"

    curl -s -XPOST "${INFLUX_HOST}/api/v2/write?org=${INFLUX_ORG}&bucket=${INFLUX_BUCKET}&precision=ns" \
        -H "Authorization: Token ${INFLUX_TOKEN}" \
        --data-raw "$line"
}

# Examples:
record_action "dappcore/core" 315 "wait" "auto-merge enabled, all checks passing"
record_action "dappcore/core" 307 "request_review_fix" "unresolved threads, attempt 1"
record_action "dappcore/core" 319 "resolve_conflict" "conflict_attempts >= 2, manual resolution"
```

### When PR Resolves

When a PR merges, closes, or is escalated:

```bash
record_outcome() {
    local repo=$1 pr=$2 result=$3 detail=$4 resolution_hours=$5
    local timestamp=$(date +%s%N)
    local line="outcome,repo=${repo//\//_},pr=${pr} result=\"${result}\",detail=\"${detail}\",resolution_hours=${resolution_hours} ${timestamp}"

    curl -s -XPOST "${INFLUX_HOST}/api/v2/write?org=${INFLUX_ORG}&bucket=${INFLUX_BUCKET}&precision=ns" \
        -H "Authorization: Token ${INFLUX_TOKEN}" \
        --data-raw "$line"
}

# Examples:
record_outcome "dappcore/core" 315 "success" "merged via auto-merge" 0.5
record_outcome "dappcore/core" 307 "success" "merged after 2 review fix requests" 4.2
record_outcome "dappcore/core" 291 "escalated" "conflict unresolvable after manual attempt" 72.0
```

---

## Query Examples

### Flux queries for analysis

```flux
// All observations for a PR over time
from(bucket: "training")
  |> range(start: -7d)
  |> filter(fn: (r) => r._measurement == "pr_observation")
  |> filter(fn: (r) => r.pr == "315")
  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")

// Action success rate by type
from(bucket: "training")
  |> range(start: -30d)
  |> filter(fn: (r) => r._measurement == "outcome")
  |> filter(fn: (r) => r._field == "result")
  |> group(columns: ["action"])
  |> count()

// Average resolution time by action type
from(bucket: "training")
  |> range(start: -30d)
  |> filter(fn: (r) => r._measurement == "outcome")
  |> filter(fn: (r) => r._field == "resolution_hours")
  |> group(columns: ["action"])
  |> mean()
```

---

## Export for Training

```bash
# Export to JSONL for model training
influx query '
from(bucket: "training")
  |> range(start: -90d)
  |> filter(fn: (r) => r._measurement == "pr_observation")
  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
' --raw | jq -c '.' > training-export.jsonl
```

---

## Integration with issue-epic.md

The `issue-epic` flow should call `record_action` at each decision point:

1. **Step 3 (CI Gate)** - After checking checks: `record_action $REPO $PR "wait" "CI running"`
2. **Step 5 (Fix Review)** - After sending fix request: `record_action $REPO $PR "request_review_fix" "unresolved threads"`
3. **Step 7 (Update Branch)** - After conflict request: `record_action $REPO $PR "request_conflict_fix" "merge conflict detected"`
4. **Step 8 (Merge)** - When PR merges: `record_outcome $REPO $PR "success" "merged" $hours`

---

*Created: 2026-02-05*
*Part of: agentic pipeline training infrastructure*
