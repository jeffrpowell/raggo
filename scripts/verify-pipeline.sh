#!/bin/bash
set -euo pipefail

FEED_HASH="${1:-}"

if [ -z "$FEED_HASH" ]; then
    echo "Usage: $0 <feed_hash>"
    echo ""
    echo "Verifies pipeline completion for a given feed hash"
    exit 1
fi

echo "Verifying pipeline for feed: $FEED_HASH"
echo ""

MANIFEST="data/manifests/${FEED_HASH}.jsonl"

if [ ! -f "$MANIFEST" ]; then
    echo "❌ Manifest not found: $MANIFEST"
    exit 1
fi

EPISODE_COUNT=$(wc -l < "$MANIFEST")
echo "📋 Episodes in manifest: $EPISODE_COUNT"

AUDIO_COUNT=$(find data/audio -name "*.mp3" -o -name "*.m4a" | wc -l)
echo "🎵 Audio files downloaded: $AUDIO_COUNT"

TRANSCRIPT_COUNT=$(find data/transcripts -name "*.json" ! -name "*.normalized.json" | wc -l)
echo "📝 Raw transcripts: $TRANSCRIPT_COUNT"

NORMALIZED_COUNT=$(find data/transcripts -name "*.normalized.json" | wc -l)
echo "✨ Normalized transcripts: $NORMALIZED_COUNT"

CHUNK_COUNT=$(find data/chunks -name "*.jsonl" | wc -l)
echo "🧩 Chunk files: $CHUNK_COUNT"

EMBEDDING_COUNT=$(find data/embeddings -name "*.jsonl" | wc -l)
echo "🔢 Embedding files: $EMBEDDING_COUNT"

INDEX_COUNT=$(find data/index -name "*.done" | wc -l)
echo "✅ Indexed episodes: $INDEX_COUNT"

echo ""

if [ "$INDEX_COUNT" -eq "$EPISODE_COUNT" ]; then
    echo "✅ Pipeline complete! All $EPISODE_COUNT episodes indexed."
    exit 0
else
    echo "⚠️  Pipeline incomplete: $INDEX_COUNT/$EPISODE_COUNT episodes indexed"
    exit 1
fi
