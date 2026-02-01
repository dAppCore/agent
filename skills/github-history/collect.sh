#!/usr/bin/env bash
# GitHub History Collector v2
# Usage: ./collect.sh <target> [--org] [--issues-only] [--prs-only]
#
# Supports:
#   Single repo:  ./collect.sh LetheanNetwork/lthn-app-vpn
#   Single org:   ./collect.sh LetheanNetwork --org
#   Batch orgs:   ./collect.sh "LetheanNetwork,graft-project,oxen-io" --org
#   Batch repos:  ./collect.sh "owner/repo1,owner/repo2"
#
# Output structure:
#   repo/{org}/{repo}/Issue/001.md, 002.md, ...
#   repo/{org}/{repo}/PR/001.md, 002.md, ...
#
# Rate limiting:
#   --check-rate   Just show current rate limit status and exit
#   Auto-pauses at 25% remaining (75% used) until reset+10s (preserves GraphQL quota)

set -e

# GitHub API allows 5000 requests/hour authenticated
# 0.05s = 20 req/sec = safe margin, bump to 0.1 if rate limited
DELAY=0.05
OUTPUT_BASE="./repo"

# Rate limit protection - check every N calls, pause if under 25% (75% used)
API_CALL_COUNT=0
RATE_CHECK_INTERVAL=100

check_rate_limit() {
    local rate_json=$(gh api rate_limit 2>/dev/null)
    if [ -z "$rate_json" ]; then
        echo "  [Rate check failed, continuing...]"
        return
    fi

    local remaining=$(echo "$rate_json" | jq -r '.resources.core.remaining')
    local limit=$(echo "$rate_json" | jq -r '.resources.core.limit')
    local reset=$(echo "$rate_json" | jq -r '.resources.core.reset')

    local percent=$((remaining * 100 / limit))

    echo ""
    echo ">>> Rate check: ${percent}% remaining ($remaining/$limit)"

    if [ "$percent" -lt 25 ]; then
        local now=$(date +%s)
        local wait_time=$((reset - now + 10))

        if [ "$wait_time" -gt 0 ]; then
            local resume_time=$(date -d "@$((reset + 10))" '+%H:%M:%S' 2>/dev/null || date -r "$((reset + 10))" '+%H:%M:%S' 2>/dev/null || echo "reset+10s")
            echo ">>> Under 25% - pausing ${wait_time}s until $resume_time"
            echo ">>> (GraphQL quota preserved for other tools)"
            sleep "$wait_time"
            echo ">>> Resuming collection..."
        fi
    else
        echo ">>> Above 25% - continuing..."
    fi
    echo ""
}

track_api_call() {
    API_CALL_COUNT=$((API_CALL_COUNT + 1))

    if [ $((API_CALL_COUNT % RATE_CHECK_INTERVAL)) -eq 0 ]; then
        check_rate_limit
    fi
}

# Parse URL into org/repo
parse_github_url() {
    local url="$1"
    url="${url#https://github.com/}"
    url="${url#http://github.com/}"
    url="${url%/}"
    echo "$url"
}

# Collect single repo
collect_repo() {
    local repo="$1"  # format: org/repo-name
    local org=$(dirname "$repo")
    local repo_name=$(basename "$repo")

    local issue_dir="$OUTPUT_BASE/$org/$repo_name/Issue"
    local pr_dir="$OUTPUT_BASE/$org/$repo_name/PR"
    local json_dir="$OUTPUT_BASE/$org/$repo_name/.json"

    mkdir -p "$issue_dir" "$pr_dir" "$json_dir"

    echo "=== Collecting: $repo ==="
    echo "    Output: $OUTPUT_BASE/$org/$repo_name/"

    # Collect Issues
    if [ "$SKIP_ISSUES" != "1" ]; then
        echo "Fetching issues..."
        if ! gh issue list --repo "$repo" --state all --limit 500 \
            --json number,title,state,author,labels,createdAt,closedAt,body \
            > "$json_dir/issues-list.json" 2>/dev/null; then
            echo "  (issues disabled or not accessible)"
            echo "[]" > "$json_dir/issues-list.json"
        fi
        track_api_call

        local issue_count=$(jq length "$json_dir/issues-list.json")
        echo "  Found $issue_count issues"

        # Fetch each issue
        local seq=0
        for github_num in $(jq -r '.[].number' "$json_dir/issues-list.json" | sort -n); do
            seq=$((seq + 1))
            local seq_padded=$(printf '%03d' $seq)

            # Skip if already fetched
            if [ -f "$json_dir/issue-$github_num.json" ] && [ -f "$issue_dir/$seq_padded.md" ]; then
                echo "  Skipping issue #$github_num (already exists)"
                continue
            fi

            echo "  Fetching issue #$github_num -> $seq_padded.md"
            gh issue view "$github_num" --repo "$repo" \
                --json number,title,state,author,labels,createdAt,closedAt,body,comments \
                > "$json_dir/issue-$github_num.json"
            track_api_call

            # Convert to markdown with sequential filename
            convert_issue "$json_dir/issue-$github_num.json" "$issue_dir/$seq_padded.md" "$github_num"
            sleep $DELAY
        done

        generate_issue_index "$issue_dir"
    fi

    # Collect PRs
    if [ "$SKIP_PRS" != "1" ]; then
        echo "Fetching PRs..."
        if ! gh pr list --repo "$repo" --state all --limit 500 \
            --json number,title,state,author,createdAt,closedAt,mergedAt,body \
            > "$json_dir/prs-list.json" 2>/dev/null; then
            echo "  (PRs disabled or not accessible)"
            echo "[]" > "$json_dir/prs-list.json"
        fi
        track_api_call

        local pr_count=$(jq length "$json_dir/prs-list.json")
        echo "  Found $pr_count PRs"

        # Fetch each PR
        local seq=0
        for github_num in $(jq -r '.[].number' "$json_dir/prs-list.json" | sort -n); do
            seq=$((seq + 1))
            local seq_padded=$(printf '%03d' $seq)

            # Skip if already fetched
            if [ -f "$json_dir/pr-$github_num.json" ] && [ -f "$pr_dir/$seq_padded.md" ]; then
                echo "  Skipping PR #$github_num (already exists)"
                continue
            fi

            echo "  Fetching PR #$github_num -> $seq_padded.md"
            gh pr view "$github_num" --repo "$repo" \
                --json number,title,state,author,createdAt,closedAt,mergedAt,body,comments,reviews \
                > "$json_dir/pr-$github_num.json" 2>/dev/null || true
            track_api_call

            # Convert to markdown with sequential filename
            convert_pr "$json_dir/pr-$github_num.json" "$pr_dir/$seq_padded.md" "$github_num"
            sleep $DELAY
        done

        generate_pr_index "$pr_dir"
    fi
}

# Collect all repos in org
collect_org() {
    local org="$1"

    echo "=== Collecting all repos from org: $org ==="

    # Get repo list (1 API call)
    local repos
    repos=$(gh repo list "$org" --limit 500 --json nameWithOwner -q '.[].nameWithOwner')
    track_api_call

    while read -r repo; do
        [ -n "$repo" ] || continue
        collect_repo "$repo"
        sleep $DELAY
    done <<< "$repos"
}

# Convert issue JSON to markdown
convert_issue() {
    local json_file="$1"
    local output_file="$2"
    local github_num="$3"

    local title=$(jq -r '.title' "$json_file")
    local state=$(jq -r '.state' "$json_file")
    local author=$(jq -r '.author.login' "$json_file")
    local created=$(jq -r '.createdAt' "$json_file" | cut -d'T' -f1)
    local closed=$(jq -r '.closedAt // "N/A"' "$json_file" | cut -d'T' -f1)
    local body=$(jq -r '.body // "No description"' "$json_file")
    local labels=$(jq -r '[.labels[].name] | join(", ")' "$json_file")
    local comment_count=$(jq '.comments | length' "$json_file")

    # Score reception
    local score="UNKNOWN"
    local reason=""

    if [ "$state" = "CLOSED" ]; then
        if echo "$labels" | grep -qi "wontfix\|invalid\|duplicate\|won't fix"; then
            score="DISMISSED"
            reason="Labeled as wontfix/invalid/duplicate"
        elif [ "$comment_count" -eq 0 ]; then
            score="IGNORED"
            reason="Closed with no discussion"
        else
            score="ADDRESSED"
            reason="Closed after discussion"
        fi
    else
        if [ "$comment_count" -eq 0 ]; then
            score="STALE"
            reason="Open with no response"
        else
            score="ACTIVE"
            reason="Open with discussion"
        fi
    fi

    cat > "$output_file" << ISSUE_EOF
# Issue #$github_num: $title

## Reception Score

| Score | Reason |
|-------|--------|
| **$score** | $reason |

---

## Metadata

| Field | Value |
|-------|-------|
| GitHub # | $github_num |
| State | $state |
| Author | @$author |
| Created | $created |
| Closed | $closed |
| Labels | $labels |
| Comments | $comment_count |

---

## Original Post

**Author:** @$author

$body

---

## Discussion Thread

ISSUE_EOF

    jq -r '.comments[] | "### Comment by @\(.author.login)\n\n**Date:** \(.createdAt | split("T")[0])\n\n\(.body)\n\n---\n"' "$json_file" >> "$output_file" 2>/dev/null || true
}

# Convert PR JSON to markdown
convert_pr() {
    local json_file="$1"
    local output_file="$2"
    local github_num="$3"

    [ -f "$json_file" ] || return

    local title=$(jq -r '.title' "$json_file")
    local state=$(jq -r '.state' "$json_file")
    local author=$(jq -r '.author.login' "$json_file")
    local created=$(jq -r '.createdAt' "$json_file" | cut -d'T' -f1)
    local merged=$(jq -r '.mergedAt // "N/A"' "$json_file" | cut -d'T' -f1)
    local body=$(jq -r '.body // "No description"' "$json_file")

    local score="UNKNOWN"
    local reason=""

    if [ "$state" = "MERGED" ] || { [ "$merged" != "N/A" ] && [ "$merged" != "null" ]; }; then
        score="MERGED"
        reason="Contribution accepted"
    elif [ "$state" = "CLOSED" ]; then
        score="REJECTED"
        reason="PR closed without merge"
    else
        score="PENDING"
        reason="Still open"
    fi

    cat > "$output_file" << PR_EOF
# PR #$github_num: $title

## Reception Score

| Score | Reason |
|-------|--------|
| **$score** | $reason |

---

## Metadata

| Field | Value |
|-------|-------|
| GitHub # | $github_num |
| State | $state |
| Author | @$author |
| Created | $created |
| Merged | $merged |

---

## Description

$body

---

## Reviews & Comments

PR_EOF

    jq -r '.comments[]? | "### Comment by @\(.author.login)\n\n\(.body)\n\n---\n"' "$json_file" >> "$output_file" 2>/dev/null || true
    jq -r '.reviews[]? | "### Review by @\(.author.login) [\(.state)]\n\n\(.body // "No comment")\n\n---\n"' "$json_file" >> "$output_file" 2>/dev/null || true
}

# Generate Issue index
generate_issue_index() {
    local dir="$1"

    cat > "$dir/INDEX.md" << 'INDEX_HEADER'
# Issues Index

## Reception Score Legend

| Score | Meaning | Action |
|-------|---------|--------|
| ADDRESSED | Closed after discussion | Review if actually fixed |
| DISMISSED | Labeled wontfix/invalid | **RECLAIM candidate** |
| IGNORED | Closed, no response | **RECLAIM candidate** |
| STALE | Open, no replies | Needs attention |
| ACTIVE | Open with discussion | In progress |

---

## Issues

| Seq | GitHub # | Title | Score |
|-----|----------|-------|-------|
INDEX_HEADER

    for file in "$dir"/[0-9]*.md; do
        [ -f "$file" ] || continue
        local seq=$(basename "$file" .md)
        local github_num=$(sed -n 's/^# Issue #\([0-9]*\):.*/\1/p' "$file")
        local title=$(head -1 "$file" | sed 's/^# Issue #[0-9]*: //')
        local score=$(sed -n '/\*\*[A-Z]/s/.*\*\*\([A-Z]*\)\*\*.*/\1/p' "$file" | head -1)
        echo "| [$seq]($seq.md) | #$github_num | $title | $score |" >> "$dir/INDEX.md"
    done

    echo "  Created Issue/INDEX.md"
}

# Generate PR index
generate_pr_index() {
    local dir="$1"

    cat > "$dir/INDEX.md" << 'INDEX_HEADER'
# Pull Requests Index

## Reception Score Legend

| Score | Meaning | Action |
|-------|---------|--------|
| MERGED | PR accepted | Done |
| REJECTED | PR closed unmerged | Review why |
| PENDING | PR still open | Needs review |

---

## Pull Requests

| Seq | GitHub # | Title | Score |
|-----|----------|-------|-------|
INDEX_HEADER

    for file in "$dir"/[0-9]*.md; do
        [ -f "$file" ] || continue
        local seq=$(basename "$file" .md)
        local github_num=$(sed -n 's/^# PR #\([0-9]*\):.*/\1/p' "$file")
        local title=$(head -1 "$file" | sed 's/^# PR #[0-9]*: //')
        local score=$(sed -n '/\*\*[A-Z]/s/.*\*\*\([A-Z]*\)\*\*.*/\1/p' "$file" | head -1)
        echo "| [$seq]($seq.md) | #$github_num | $title | $score |" >> "$dir/INDEX.md"
    done

    echo "  Created PR/INDEX.md"
}

# Show rate limit status
show_rate_status() {
    local rate_json=$(gh api rate_limit 2>/dev/null)
    if [ -z "$rate_json" ]; then
        echo "Failed to fetch rate limit"
        exit 1
    fi

    echo "=== GitHub API Rate Limit Status ==="
    echo ""
    echo "Core (REST API):"
    echo "  Remaining: $(echo "$rate_json" | jq -r '.resources.core.remaining') / $(echo "$rate_json" | jq -r '.resources.core.limit')"
    local core_reset=$(echo "$rate_json" | jq -r '.resources.core.reset')
    echo "  Reset: $(date -d "@$core_reset" '+%H:%M:%S' 2>/dev/null || date -r "$core_reset" '+%H:%M:%S' 2>/dev/null || echo "$core_reset")"
    echo ""
    echo "GraphQL:"
    echo "  Remaining: $(echo "$rate_json" | jq -r '.resources.graphql.remaining') / $(echo "$rate_json" | jq -r '.resources.graphql.limit')"
    local gql_reset=$(echo "$rate_json" | jq -r '.resources.graphql.reset')
    echo "  Reset: $(date -d "@$gql_reset" '+%H:%M:%S' 2>/dev/null || date -r "$gql_reset" '+%H:%M:%S' 2>/dev/null || echo "$gql_reset")"
    echo ""
    echo "Search:"
    echo "  Remaining: $(echo "$rate_json" | jq -r '.resources.search.remaining') / $(echo "$rate_json" | jq -r '.resources.search.limit')"
    echo ""
}

# Main
main() {
    local targets=""
    local is_org=0
    SKIP_ISSUES=0
    SKIP_PRS=0

    # Parse args
    for arg in "$@"; do
        case "$arg" in
            --org) is_org=1 ;;
            --issues-only) SKIP_PRS=1 ;;
            --prs-only) SKIP_ISSUES=1 ;;
            --delay=*) DELAY="${arg#*=}" ;;
            --check-rate) show_rate_status; exit 0 ;;
            https://*|http://*) targets="$arg" ;;
            -*) ;; # ignore unknown flags
            *) targets="$arg" ;;
        esac
    done

    if [ -z "$targets" ]; then
        echo "Usage: $0 <target> [--org] [--issues-only] [--prs-only] [--delay=0.05] [--check-rate]"
        echo ""
        echo "Options:"
        echo "  --check-rate   Show rate limit status (Core/GraphQL/Search) and exit"
        echo "  --delay=N      Delay between requests (default: 0.05s)"
        echo ""
        echo "Rate limiting: Auto-pauses at 25% remaining (75% used) until reset+10s"
        echo ""
        echo "Target formats:"
        echo "  Single repo:  LetheanNetwork/lthn-app-vpn"
        echo "  Single org:   LetheanNetwork --org"
        echo "  Batch orgs:   \"LetheanNetwork,graft-project,oxen-io\" --org"
        echo "  Batch repos:  \"owner/repo1,owner/repo2\""
        echo ""
        echo "Output: repo/{org}/{repo}/Issue/  repo/{org}/{repo}/PR/"
        echo ""
        echo "Full registry list (copy-paste ready):"
        echo ""
        echo "  # Lethean ecosystem"
        echo "  $0 \"LetheanNetwork,letheanVPN,LetheanMovement\" --org"
        echo ""
        echo "  # CryptoNote projects"
        echo "  $0 \"monero-project,haven-protocol-org,hyle-team,zanoio\" --org"
        echo "  $0 \"kevacoin-project,scala-network,deroproject\" --org"
        echo "  $0 \"Karbovanets,wownero,turtlecoin\" --org"
        echo "  $0 \"masari-project,aeonix,nerva-project\" --org"
        echo "  $0 \"ConcealNetwork,ryo-currency,sumoprojects\" --org"
        echo "  $0 \"bcndev,electroneum\" --org"
        echo ""
        echo "  # Dead/salvage priority"
        echo "  $0 \"graft-project,graft-community,oxen-io,loki-project\" --org"
        echo ""
        echo "  # Non-CN reference projects"
        echo "  $0 \"theQRL,hyperswarm,holepunchto,openhive-network,octa-space\" --org"
        exit 1
    fi

    # Handle comma-separated list
    IFS=',' read -ra TARGET_LIST <<< "$targets"

    for target in "${TARGET_LIST[@]}"; do
        # Trim whitespace
        target=$(echo "$target" | xargs)
        local parsed=$(parse_github_url "$target")

        if [ "$is_org" = "1" ]; then
            collect_org "$parsed"
        else
            collect_repo "$parsed"
        fi
    done

    echo ""
    echo "=== Collection Complete ==="
    echo "Output: $OUTPUT_BASE/"
}

main "$@"
