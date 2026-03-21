#!/usr/bin/env bash
# Generate job list for proxy-based collection
# Usage: ./generate-jobs.sh <source> <target> [options] > jobs.txt

set -e

SOURCE="$1"
TARGET="$2"
shift 2 || true

# Defaults
LIMIT=1000
PAGES=100

# Parse options
for arg in "$@"; do
    case "$arg" in
        --limit=*) LIMIT="${arg#*=}" ;;
        --pages=*) PAGES="${arg#*=}" ;;
    esac
done

# Output header
echo "# Job list generated $(date +%Y-%m-%d\ %H:%M)"
echo "# Source: $SOURCE | Target: $TARGET"
echo "# Format: URL|FILENAME|TYPE|METADATA"
echo "#"

case "$SOURCE" in

    bitcointalk|btt)
        # Extract topic ID
        TOPIC_ID=$(echo "$TARGET" | grep -oE '[0-9]+' | head -1)
        echo "# BitcoinTalk topic: $TOPIC_ID"
        echo "#"

        # Generate page URLs (20 posts per page)
        for ((i=0; i<PAGES*20; i+=20)); do
            echo "https://bitcointalk.org/index.php?topic=${TOPIC_ID}.${i}|btt-${TOPIC_ID}-p${i}.html|bitcointalk|page=$((i/20)),offset=$i"
        done
        ;;

    reddit)
        # Handle r/subreddit or full URL
        SUBREDDIT=$(echo "$TARGET" | sed 's|.*/r/||' | sed 's|/.*||')
        echo "# Reddit: r/$SUBREDDIT"
        echo "#"

        # Subreddit pages (top, new, hot)
        for sort in "top" "new" "hot"; do
            echo "https://old.reddit.com/r/${SUBREDDIT}/${sort}/.json?limit=100|reddit-${SUBREDDIT}-${sort}.json|reddit|sort=$sort"
        done

        # If it's a specific thread
        if [[ "$TARGET" =~ comments/([a-z0-9]+) ]]; then
            THREAD_ID="${BASH_REMATCH[1]}"
            echo "https://old.reddit.com/r/${SUBREDDIT}/comments/${THREAD_ID}.json|reddit-thread-${THREAD_ID}.json|reddit|thread=$THREAD_ID"
        fi
        ;;

    wayback|archive)
        # Clean domain
        DOMAIN=$(echo "$TARGET" | sed 's|https\?://||' | sed 's|/.*||')
        echo "# Wayback Machine: $DOMAIN"
        echo "#"

        # CDX API to get all snapshots
        echo "https://web.archive.org/cdx/search/cdx?url=${DOMAIN}/*&output=json&limit=${LIMIT}|wayback-${DOMAIN}-cdx.json|wayback-index|domain=$DOMAIN"

        # Common important pages
        for path in "" "index.html" "about" "roadmap" "team" "whitepaper" "faq"; do
            echo "https://web.archive.org/web/2020/${DOMAIN}/${path}|wayback-${DOMAIN}-2020-${path:-index}.html|wayback|year=2020,path=$path"
            echo "https://web.archive.org/web/2021/${DOMAIN}/${path}|wayback-${DOMAIN}-2021-${path:-index}.html|wayback|year=2021,path=$path"
            echo "https://web.archive.org/web/2022/${DOMAIN}/${path}|wayback-${DOMAIN}-2022-${path:-index}.html|wayback|year=2022,path=$path"
        done
        ;;

    medium)
        # Handle @author or publication
        AUTHOR=$(echo "$TARGET" | sed 's|.*/||' | sed 's|^@||')
        echo "# Medium: @$AUTHOR"
        echo "#"

        # Medium RSS feed (easier to parse)
        echo "https://medium.com/feed/@${AUTHOR}|medium-${AUTHOR}-feed.xml|medium-rss|author=$AUTHOR"

        # Profile page
        echo "https://medium.com/@${AUTHOR}|medium-${AUTHOR}-profile.html|medium|author=$AUTHOR"
        ;;

    twitter|x)
        USERNAME=$(echo "$TARGET" | sed 's|.*/||' | sed 's|^@||')
        echo "# Twitter/X: @$USERNAME"
        echo "# Note: Twitter requires auth - use nitter or API"
        echo "#"

        # Nitter instances (public, no auth)
        echo "https://nitter.net/${USERNAME}|twitter-${USERNAME}.html|nitter|user=$USERNAME"
        echo "https://nitter.net/${USERNAME}/with_replies|twitter-${USERNAME}-replies.html|nitter|user=$USERNAME,type=replies"
        ;;

    *)
        echo "# ERROR: Unknown source '$SOURCE'" >&2
        echo "# Supported: bitcointalk, reddit, wayback, medium, twitter" >&2
        exit 1
        ;;
esac
