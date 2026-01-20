#!/bin/bash
set -euo pipefail

EPISODE_ID="${1:-}"

if [ -z "$EPISODE_ID" ]; then
    echo "Usage: $0 <episode_id>"
    echo ""
    echo "Inspects all artifacts for a given episode ID"
    exit 1
fi

echo "Episode: $EPISODE_ID"
echo "========================================"
echo ""

if [ -f "data/audio/${EPISODE_ID}.json" ]; then
    echo "📊 Audio Metadata:"
    jq '{file_path, file_size, duration}' "data/audio/${EPISODE_ID}.json"
    echo ""
fi

if [ -f "data/transcripts/${EPISODE_ID}.json" ]; then
    echo "📝 Transcript:"
    jq '{model, model_version, segment_count: (.segments | length)}' "data/transcripts/${EPISODE_ID}.json"
    echo ""
    echo "First segment:"
    jq '.segments[0]' "data/transcripts/${EPISODE_ID}.json"
    echo ""
fi

if [ -f "data/transcripts/${EPISODE_ID}.normalized.json" ]; then
    echo "✨ Normalized Transcript:"
    jq '{segment_count: (.segments | length), normalized_at}' "data/transcripts/${EPISODE_ID}.normalized.json"
    echo ""
fi

if [ -f "data/chunks/${EPISODE_ID}.jsonl" ]; then
    CHUNK_COUNT=$(wc -l < "data/chunks/${EPISODE_ID}.jsonl")
    echo "🧩 Chunks: $CHUNK_COUNT"
    echo ""
    echo "First chunk:"
    head -1 "data/chunks/${EPISODE_ID}.jsonl" | jq '{chunk_id, chunk_index, start_time, end_time, text: (.text | .[0:100] + "...")}'
    echo ""
fi

if [ -f "data/embeddings/${EPISODE_ID}.jsonl" ]; then
    EMBEDDING_COUNT=$(wc -l < "data/embeddings/${EPISODE_ID}.jsonl")
    echo "🔢 Embeddings: $EMBEDDING_COUNT"
    echo ""
    echo "First embedding:"
    head -1 "data/embeddings/${EPISODE_ID}.jsonl" | jq '{chunk_id, model, dimension, vector_preview: (.vector | .[0:5])}'
    echo ""
fi

if [ -f "data/index/${EPISODE_ID}.done" ]; then
    echo "✅ Indexed: YES"
else
    echo "⚠️  Indexed: NO"
fi
