#!/usr/bin/env bash
# BitcoinTalk Thread Collector
# Usage: ./collect.sh <topic-id-or-url> [--pages=N] [--output=DIR]

set -e

DELAY=2  # Be respectful to BTT servers
MAX_PAGES=0  # 0 = all pages
OUTPUT_BASE="."

# Parse topic ID from URL or direct input
parse_topic_id() {
    local input="$1"
    if [[ "$input" =~ topic=([0-9]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "$input" | grep -oE '[0-9]+'
    fi
}

# Fetch a single page
fetch_page() {
    local topic_id="$1"
    local offset="$2"
    local output_file="$3"

    local url="https://bitcointalk.org/index.php?topic=${topic_id}.${offset}"
    echo "  Fetching: $url"

    curl -s -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" \
        -H "Accept: text/html" \
        "$url" > "$output_file"

    sleep $DELAY
}

# Check if page has posts
page_has_posts() {
    local html_file="$1"
    grep -q 'class="post"' "$html_file" 2>/dev/null
}

# Get last page number from first page
get_last_page() {
    local html_file="$1"
    # Look for navigation like "Pages: [1] 2 3 ... 50"
    local max_page=$(grep -oE 'topic=[0-9]+\.[0-9]+' "$html_file" | \
        sed 's/.*\.//' | sort -rn | head -1)
    echo "${max_page:-0}"
}

# Extract posts from HTML (simplified - works for basic extraction)
extract_posts_simple() {
    local html_file="$1"
    local output_dir="$2"
    local post_offset="$3"

    # Use Python for reliable HTML parsing
    python3 << PYEOF
import re
import html
import os
from datetime import datetime

html_content = open('$html_file', 'r', encoding='utf-8', errors='ignore').read()

# Pattern to find posts - BTT structure
post_pattern = r'<td class="td_headerandpost">(.*?)</td>\s*</tr>\s*</table>\s*</td>\s*</tr>'
author_pattern = r'<a href="https://bitcointalk\.org/index\.php\?action=profile;u=\d+"[^>]*>([^<]+)</a>'
date_pattern = r'<div class="smalltext">([A-Za-z]+ \d+, \d+, \d+:\d+:\d+ [AP]M)</div>'
post_content_pattern = r'<div class="post"[^>]*>(.*?)</div>\s*(?:<div class="moderatorbar"|</td>)'

posts = re.findall(post_pattern, html_content, re.DOTALL)
post_num = $post_offset

for post_html in posts:
    post_num += 1

    # Extract author
    author_match = re.search(author_pattern, post_html)
    author = author_match.group(1) if author_match else "Unknown"

    # Extract date
    date_match = re.search(date_pattern, post_html)
    date_str = date_match.group(1) if date_match else "Unknown date"

    # Extract content
    content_match = re.search(post_content_pattern, post_html, re.DOTALL)
    if content_match:
        content = content_match.group(1)
        # Clean HTML
        content = re.sub(r'<br\s*/?>', '\n', content)
        content = re.sub(r'<[^>]+>', '', content)
        content = html.unescape(content)
        content = content.strip()
    else:
        content = "(Could not extract content)"

    # Determine post type/score
    score = "COMMUNITY"
    if post_num == 1:
        score = "ANN"
    elif re.search(r'\[UPDATE\]|\[RELEASE\]|\[ANNOUNCEMENT\]', content, re.I):
        score = "UPDATE"
    elif '?' in content[:200]:
        score = "QUESTION"

    # Write post file
    filename = f"$output_dir/POST-{post_num:04d}.md"
    with open(filename, 'w') as f:
        f.write(f"# Post #{post_num}\n\n")
        f.write(f"## Metadata\n\n")
        f.write(f"| Field | Value |\n")
        f.write(f"|-------|-------|\n")
        f.write(f"| Author | {author} |\n")
        f.write(f"| Date | {date_str} |\n")
        f.write(f"| Type | **{score}** |\n\n")
        f.write(f"---\n\n")
        f.write(f"## Content\n\n")
        f.write(content)
        f.write(f"\n")

    print(f"  Created POST-{post_num:04d}.md ({score}) by {author}")

print(f"EXTRACTED:{post_num}")
PYEOF
}

# Main collection function
collect_thread() {
    local topic_id="$1"
    local output_dir="$OUTPUT_BASE/bitcointalk-$topic_id"

    mkdir -p "$output_dir/pages" "$output_dir/posts"

    echo "=== Collecting BitcoinTalk Topic: $topic_id ==="

    # Fetch first page to get thread info
    fetch_page "$topic_id" 0 "$output_dir/pages/page-0.html"

    # Extract thread title
    local title=$(grep -oP '<title>\K[^<]+' "$output_dir/pages/page-0.html" | head -1)
    echo "Thread: $title"

    # Get total pages
    local last_offset=$(get_last_page "$output_dir/pages/page-0.html")
    local total_pages=$(( (last_offset / 20) + 1 ))
    echo "Total pages: $total_pages"

    if [ "$MAX_PAGES" -gt 0 ] && [ "$MAX_PAGES" -lt "$total_pages" ]; then
        total_pages=$MAX_PAGES
        echo "Limiting to: $total_pages pages"
    fi

    # Extract posts from first page
    local post_count=0
    local result=$(extract_posts_simple "$output_dir/pages/page-0.html" "$output_dir/posts" 0)
    post_count=$(echo "$result" | grep "EXTRACTED:" | cut -d: -f2)

    # Fetch remaining pages
    for (( page=1; page<total_pages; page++ )); do
        local offset=$((page * 20))
        fetch_page "$topic_id" "$offset" "$output_dir/pages/page-$offset.html"

        if ! page_has_posts "$output_dir/pages/page-$offset.html"; then
            echo "  No more posts found, stopping."
            break
        fi

        result=$(extract_posts_simple "$output_dir/pages/page-$offset.html" "$output_dir/posts" "$post_count")
        post_count=$(echo "$result" | grep "EXTRACTED:" | cut -d: -f2)
    done

    # Generate index
    generate_index "$output_dir" "$title" "$topic_id" "$post_count"

    echo ""
    echo "=== Collection Complete ==="
    echo "Posts: $post_count"
    echo "Output: $output_dir/"
}

# Generate index file
generate_index() {
    local output_dir="$1"
    local title="$2"
    local topic_id="$3"
    local post_count="$4"

    cat > "$output_dir/INDEX.md" << EOF
# BitcoinTalk Thread Archive

## Thread Info

| Field | Value |
|-------|-------|
| Title | $title |
| Topic ID | $topic_id |
| URL | https://bitcointalk.org/index.php?topic=$topic_id.0 |
| Posts Archived | $post_count |
| Collected | $(date +%Y-%m-%d) |

---

## Post Type Legend

| Type | Meaning |
|------|---------|
| ANN | Original announcement |
| UPDATE | Official team update |
| QUESTION | Community question |
| ANSWER | Team response |
| COMMUNITY | General discussion |
| CONCERN | Raised issue/criticism |

---

## Posts

| # | Author | Date | Type |
|---|--------|------|------|
EOF

    for file in "$output_dir/posts/"POST-*.md; do
        [ -f "$file" ] || continue
        local num=$(basename "$file" .md | sed 's/POST-0*//')
        local author=$(grep "| Author |" "$file" | sed 's/.*| Author | \(.*\) |/\1/')
        local date=$(grep "| Date |" "$file" | sed 's/.*| Date | \(.*\) |/\1/')
        local type=$(sed -n '/| Type |/s/.*\*\*\([A-Z]*\)\*\*.*/\1/p' "$file")
        echo "| [$num](posts/POST-$(printf "%04d" $num).md) | $author | $date | $type |" >> "$output_dir/INDEX.md"
    done

    echo "  Created INDEX.md"
}

# Parse arguments
main() {
    local topic_input=""

    for arg in "$@"; do
        case "$arg" in
            --pages=*) MAX_PAGES="${arg#*=}" ;;
            --output=*) OUTPUT_BASE="${arg#*=}" ;;
            --delay=*) DELAY="${arg#*=}" ;;
            *) topic_input="$arg" ;;
        esac
    done

    if [ -z "$topic_input" ]; then
        echo "Usage: $0 <topic-id-or-url> [--pages=N] [--output=DIR] [--delay=2]"
        echo ""
        echo "Examples:"
        echo "  $0 2769739"
        echo "  $0 https://bitcointalk.org/index.php?topic=2769739.0"
        echo "  $0 2769739 --pages=5 --output=./lethean-ann"
        exit 1
    fi

    local topic_id=$(parse_topic_id "$topic_input")

    if [ -z "$topic_id" ]; then
        echo "Error: Could not parse topic ID from: $topic_input"
        exit 1
    fi

    collect_thread "$topic_id"
}

main "$@"
