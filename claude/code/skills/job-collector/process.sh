#!/usr/bin/env bash
# Process downloaded files into markdown
# Usage: ./process.sh <source> <downloads-dir> [--output=DIR]

set -e

SOURCE="$1"
DOWNLOADS="$2"
shift 2 || true

OUTPUT="./processed"

for arg in "$@"; do
    case "$arg" in
        --output=*) OUTPUT="${arg#*=}" ;;
    esac
done

mkdir -p "$OUTPUT/posts"

echo "=== Processing $SOURCE files from $DOWNLOADS ==="

case "$SOURCE" in

    bitcointalk|btt)
        echo "Processing BitcoinTalk pages..."
        POST_NUM=0

        for file in "$DOWNLOADS"/btt-*.html; do
            [ -f "$file" ] || continue
            echo "  Processing: $(basename "$file")"

            python3 << PYEOF
import re
import html
import os

html_content = open('$file', 'r', encoding='utf-8', errors='ignore').read()

# Extract thread title from first page
title_match = re.search(r'<title>([^<]+)</title>', html_content)
title = title_match.group(1) if title_match else "Unknown Thread"
title = title.replace(' - Bitcoin Forum', '').strip()

with open('$OUTPUT/.thread_title', 'w') as f:
    f.write(title)

# Pattern for posts
post_blocks = re.findall(r'<div class="post"[^>]*id="msg(\d+)"[^>]*>(.*?)</div>\s*(?:<div class="moderatorbar"|<div class="signature">)', html_content, re.DOTALL)

for msg_id, content in post_blocks:
    # Clean content
    content = re.sub(r'<br\s*/?>', '\n', content)
    content = re.sub(r'<[^>]+>', '', content)
    content = html.unescape(content).strip()

    if content:
        post_num = $POST_NUM + 1
        $POST_NUM = post_num

        with open(f'$OUTPUT/posts/POST-{post_num:04d}.md', 'w') as f:
            f.write(f"# Post #{post_num}\\n\\n")
            f.write(f"Message ID: {msg_id}\\n\\n")
            f.write(f"---\\n\\n")
            f.write(content)
            f.write("\\n")

        print(f"    POST-{post_num:04d}.md")

print(f"TOTAL:{$POST_NUM}")
PYEOF
        done

        # Generate index
        TITLE=$(cat "$OUTPUT/.thread_title" 2>/dev/null || echo "BitcoinTalk Thread")
        TOTAL=$(ls "$OUTPUT/posts/"POST-*.md 2>/dev/null | wc -l)

        cat > "$OUTPUT/INDEX.md" << EOF
# $TITLE

Archived from BitcoinTalk

| Posts | $(echo $TOTAL) |
|-------|------|

## Posts

EOF
        for f in "$OUTPUT/posts/"POST-*.md; do
            [ -f "$f" ] || continue
            NUM=$(basename "$f" .md | sed 's/POST-0*//')
            echo "- [Post #$NUM](posts/$(basename $f))" >> "$OUTPUT/INDEX.md"
        done
        ;;

    reddit)
        echo "Processing Reddit JSON..."

        for file in "$DOWNLOADS"/reddit-*.json; do
            [ -f "$file" ] || continue
            echo "  Processing: $(basename "$file")"

            python3 << PYEOF
import json
import os

data = json.load(open('$file', 'r'))

# Handle different Reddit JSON structures
posts = []
if isinstance(data, list) and len(data) > 0:
    if 'data' in data[0]:
        # Thread format
        posts = data[0]['data']['children']
    else:
        posts = data
elif isinstance(data, dict) and 'data' in data:
    posts = data['data']['children']

for i, post_wrapper in enumerate(posts):
    post = post_wrapper.get('data', post_wrapper)

    title = post.get('title', post.get('body', '')[:50])
    author = post.get('author', 'unknown')
    score = post.get('score', 0)
    body = post.get('selftext', post.get('body', ''))
    created = post.get('created_utc', 0)

    filename = f'$OUTPUT/posts/REDDIT-{i+1:04d}.md'
    with open(filename, 'w') as f:
        f.write(f"# {title}\\n\\n")
        f.write(f"| Author | u/{author} |\\n")
        f.write(f"|--------|----------|\\n")
        f.write(f"| Score | {score} |\\n\\n")
        f.write(f"---\\n\\n")
        f.write(body or "(no content)")
        f.write("\\n")

    print(f"    REDDIT-{i+1:04d}.md - {title[:40]}...")
PYEOF
        done
        ;;

    wayback)
        echo "Processing Wayback Machine files..."

        for file in "$DOWNLOADS"/wayback-*.html; do
            [ -f "$file" ] || continue
            BASENAME=$(basename "$file" .html)
            echo "  Processing: $BASENAME"

            # Extract text content
            python3 << PYEOF
import re
import html

content = open('$file', 'r', encoding='utf-8', errors='ignore').read()

# Remove scripts and styles
content = re.sub(r'<script[^>]*>.*?</script>', '', content, flags=re.DOTALL)
content = re.sub(r'<style[^>]*>.*?</style>', '', content, flags=re.DOTALL)

# Extract title
title_match = re.search(r'<title>([^<]+)</title>', content)
title = html.unescape(title_match.group(1)) if title_match else "$BASENAME"

# Get body text
body_match = re.search(r'<body[^>]*>(.*?)</body>', content, re.DOTALL)
if body_match:
    body = body_match.group(1)
    body = re.sub(r'<[^>]+>', ' ', body)
    body = html.unescape(body)
    body = re.sub(r'\s+', ' ', body).strip()
else:
    body = "(could not extract body)"

with open('$OUTPUT/posts/$BASENAME.md', 'w') as f:
    f.write(f"# {title}\\n\\n")
    f.write(f"Source: Wayback Machine\\n\\n")
    f.write(f"---\\n\\n")
    f.write(body[:5000])  # Limit length
    f.write("\\n")

print(f"    $BASENAME.md")
PYEOF
        done
        ;;

    medium)
        echo "Processing Medium files..."

        # Handle RSS feed
        for file in "$DOWNLOADS"/medium-*-feed.xml; do
            [ -f "$file" ] || continue
            echo "  Processing RSS: $(basename "$file")"

            python3 << PYEOF
import xml.etree.ElementTree as ET
import html
import re

tree = ET.parse('$file')
root = tree.getroot()

channel = root.find('channel')
items = channel.findall('item') if channel else root.findall('.//item')

for i, item in enumerate(items):
    title = item.findtext('title', 'Untitled')
    author = item.findtext('{http://purl.org/dc/elements/1.1/}creator', 'Unknown')
    date = item.findtext('pubDate', '')
    content = item.findtext('{http://purl.org/rss/1.0/modules/content/}encoded', '')

    # Clean content
    content = re.sub(r'<[^>]+>', '', content)
    content = html.unescape(content)

    filename = f'$OUTPUT/posts/MEDIUM-{i+1:04d}.md'
    with open(filename, 'w') as f:
        f.write(f"# {title}\\n\\n")
        f.write(f"| Author | {author} |\\n")
        f.write(f"|--------|----------|\\n")
        f.write(f"| Date | {date} |\\n\\n")
        f.write(f"---\\n\\n")
        f.write(content[:10000])
        f.write("\\n")

    print(f"    MEDIUM-{i+1:04d}.md - {title[:40]}...")
PYEOF
        done
        ;;

    *)
        echo "ERROR: Unknown source '$SOURCE'"
        echo "Supported: bitcointalk, reddit, wayback, medium"
        exit 1
        ;;
esac

echo ""
echo "=== Processing Complete ==="
echo "Output: $OUTPUT/"
