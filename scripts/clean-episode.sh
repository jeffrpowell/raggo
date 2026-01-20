#!/bin/bash
set -euo pipefail

EPISODE_ID="${1:-}"

if [ -z "$EPISODE_ID" ]; then
    echo "Usage: $0 <episode_id>"
    echo ""
    echo "Removes all artifacts for a given episode ID"
    echo "Use this to force re-processing of an episode"
    exit 1
fi

echo "Removing all artifacts for episode: $EPISODE_ID"
echo ""

removed=0

for artifact in \
    "data/audio/${EPISODE_ID}.mp3" \
    "data/audio/${EPISODE_ID}.m4a" \
    "data/audio/${EPISODE_ID}.json" \
    "data/transcripts/${EPISODE_ID}.json" \
    "data/transcripts/${EPISODE_ID}.normalized.json" \
    "data/chunks/${EPISODE_ID}.jsonl" \
    "data/embeddings/${EPISODE_ID}.jsonl" \
    "data/index/${EPISODE_ID}.done"
do
    if [ -f "$artifact" ]; then
        echo "Removing: $artifact"
        rm "$artifact"
        ((removed++))
    fi
done

echo ""
echo "Removed $removed artifact(s)"
echo ""
echo "To re-process this episode, run:"
echo "  make pipeline FEED_URL=<your_feed_url>"
