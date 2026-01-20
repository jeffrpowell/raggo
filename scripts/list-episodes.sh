#!/bin/bash
set -euo pipefail

FEED_HASH="${1:-}"

if [ -z "$FEED_HASH" ]; then
    echo "Usage: $0 <feed_hash>"
    echo ""
    echo "Lists all episodes from a manifest"
    echo ""
    echo "Available manifests:"
    find data/manifests -name "*.jsonl" -exec basename {} \; 2>/dev/null || true
    exit 1
fi

MANIFEST="data/manifests/${FEED_HASH}.jsonl"

if [ ! -f "$MANIFEST" ]; then
    echo "Manifest not found: $MANIFEST"
    exit 1
fi

echo "Episodes in manifest: $MANIFEST"
echo ""

jq -r '"\(.episode_id) | \(.title) | \(.published_at)"' "$MANIFEST" | \
    column -t -s '|' -N "Episode ID,Title,Published"
