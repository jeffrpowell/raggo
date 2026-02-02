# Raggo Usage Examples

This document provides comprehensive usage examples for the Raggo pipeline, ordered from most common to advanced use cases.

## 1. Basic: Full Pipeline with Placeholders

Run the complete pipeline without external services. Uses placeholder transcripts and embeddings for testing.

```bash
make podcast FEED_URL="https://feeds.simplecast.com/54nAGcIl"
```

**What happens:**
- ✅ RSS feed is fetched and parsed
- ✅ Audio files are downloaded
- ⚠️  Placeholder transcripts generated (no STT configured)
- ✅ Transcripts normalized
- ✅ Chunks created
- ⚠️  Placeholder embeddings generated (no embedding service)
- ✅ Indexed into Qdrant

## 2. Full Pipeline with Real Services

Run with actual STT and embedding services for production-quality results.

```bash
make podcast \
  FEED_URL="https://feeds.simplecast.com/54nAGcIl" \
  STT_ENDPOINT="http://localhost:8000/stt" \
  EMBED_ENDPOINT="http://localhost:8001/embed"
```

## 3. Running Individual Pipeline Stages

Execute specific stages independently for debugging or partial reprocessing.

```bash
cd pipelines/podcast

# Stage 1: RSS ingestion only
make rss FEED_URL="https://example.com/podcast.rss"

# Stage 2: Download audio files
make audio FEED_URL="https://example.com/podcast.rss"

# Stage 3: Transcribe audio
make transcripts FEED_URL="https://example.com/podcast.rss" STT_ENDPOINT="http://localhost:8000/stt"

# Stage 4: Normalize transcripts
make normalize FEED_URL="https://example.com/podcast.rss"

# Stage 5: Create chunks
make chunks FEED_URL="https://example.com/podcast.rss"

# Stage 6: Generate embeddings
make embeddings FEED_URL="https://example.com/podcast.rss" EMBED_ENDPOINT="http://localhost:8001/embed"

# Stage 7: Index into Qdrant
make index FEED_URL="https://example.com/podcast.rss"
```

## 4. Parallel Processing

Speed up pipeline execution by processing multiple episodes concurrently.

```bash
cd pipelines/podcast

# Process 8 episodes at once
make -j 8 transcripts FEED_URL="..." STT_ENDPOINT="..."

# Download audio for all episodes in parallel
make -j 4 audio FEED_URL="..."
```

## 5. Re-running Specific Stages

Delete artifacts and regenerate them with different parameters or services.

### Re-run Single Episode

```bash
# Get episode ID
EPISODE_ID=$(jq -r '.episode_id' data/podcast/manifests/*.jsonl | head -1)

# Clean episode artifacts
./scripts/clean-episode.sh $EPISODE_ID

# Re-run pipeline for that episode
make podcast FEED_URL="..."
```

### Re-run Stage for All Episodes

```bash
# Re-chunk everything with different parameters
rm data/podcast/chunks/*.jsonl
rm data/podcast/embeddings/*.jsonl
rm data/podcast/index/*.done

# Re-run from chunking onwards
cd pipelines/podcast
make chunks FEED_URL="..."
make embeddings FEED_URL="..." EMBED_ENDPOINT="..."
make index FEED_URL="..."
```

### Re-embed with Better Model

```bash
# Remove old embeddings and index markers
rm data/podcast/embeddings/*.jsonl
rm data/podcast/index/*.done

# Re-run with new embedding endpoint
cd pipelines/podcast
make embeddings FEED_URL="..." EMBED_ENDPOINT="http://new-model:8001/embed"
make index FEED_URL="..."
```

## 6. Inspecting Pipeline Artifacts

Examine intermediate outputs to understand or debug the pipeline.

### Check Pipeline Status

```bash
# Get feed hash
FEED_HASH=$(ls data/podcast/manifests/ | head -1 | sed 's/.jsonl//')

# Verify pipeline completion
./scripts/verify-pipeline.sh $FEED_HASH

# Check if full pipeline completed
ls -lh data/podcast/index/qdrant.done
```

### List All Episodes

```bash
./scripts/list-episodes.sh $FEED_HASH
```

### Inspect Specific Episode

```bash
# Get episode ID
EPISODE_ID=$(jq -r '.episode_id' data/podcast/manifests/$FEED_HASH.jsonl | head -1)

# Inspect all artifacts for this episode
./scripts/inspect-episode.sh $EPISODE_ID
```

### Manual Artifact Inspection

```bash
# View episode manifest
cat data/podcast/manifests/*.jsonl | jq .title

# View specific chunk content
cat data/podcast/chunks/<episode_id>.jsonl | jq -r 'select(.chunk_index == 5) | .text'

# Check embedding dimensions
cat data/podcast/embeddings/<episode_id>.jsonl | jq .dimension | head -1

# Count chunks per episode
wc -l data/podcast/chunks/*.jsonl

# View raw transcript segments
cat data/podcast/transcripts/<episode_id>.json | jq .segments[0]
```

### Query Qdrant

```bash
# Check collection info
curl http://localhost:6333/collections/raggo

# Count indexed points
curl http://localhost:6333/collections/raggo | jq '.result.points_count'

# View collection configuration
curl http://localhost:6333/collections/raggo | jq .result.config
```

## 7. Configuration Examples

### Using Configuration File

```bash
# Copy example config
cp config.example.mk config.mk

# Edit with your settings
nano config.mk

# Example config.mk content:
# FEED_URL = https://feeds.example.com/podcast.rss
# STT_ENDPOINT = http://localhost:8000/stt
# EMBED_ENDPOINT = http://localhost:8001/embed
# QDRANT_HOST = localhost
# QDRANT_PORT = 6334
# COLLECTION = my-raggo-collection
# CORPUS = my-podcast-corpus
# WORKERS = 8

# Run with config
make -f config.mk podcast
```

### Environment Variables

```bash
# Set environment variables
export FEED_URL="https://example.com/podcast.rss"
export STT_ENDPOINT="http://localhost:8000/stt"
export EMBED_ENDPOINT="http://localhost:8001/embed"
export QDRANT_HOST="localhost"
export QDRANT_PORT="6334"
export COLLECTION="raggo"
export CORPUS="my-corpus"
export WORKERS="4"

# Run pipeline
make podcast
```

## 8. Daily Podcast Ingestion Workflow

Automated script for processing multiple feeds regularly.

```bash
#!/bin/bash
# daily-ingest.sh

FEEDS=(
  "https://feeds.example.com/podcast1.rss"
  "https://feeds.example.com/podcast2.rss"
  "https://feeds.example.com/podcast3.rss"
)

for feed in "${FEEDS[@]}"; do
  echo "Processing: $feed"
  make podcast \
    FEED_URL="$feed" \
    STT_ENDPOINT="http://stt-service:8000/stt" \
    EMBED_ENDPOINT="http://embed-service:8001/embed" \
    CORPUS="$(echo $feed | md5sum | cut -d' ' -f1)"
done
```

**Schedule with cron:**

```bash
# crontab -e
0 */6 * * * cd /path/to/raggo && ./daily-ingest.sh
```

## 9. Incremental Updates

Process only new episodes from a feed you've already ingested.

```bash
# Fetch latest RSS (Make will skip existing episodes automatically)
make rss FEED_URL="..."

# Run full pipeline - only new episodes will be processed
make podcast FEED_URL="..."
```

## 10. Testing with Local Files

Use local test data instead of external feeds.

```bash
# Create test fixtures
mkdir -p test/fixtures
# Place test RSS feed at test/fixtures/feed.xml

# Run pipeline with file:// URL
bin/raggo-rss-podcast -feed-url="file://$(pwd)/test/fixtures/feed.xml" -manifest-dir="test/output"

# Test individual binary
bin/raggo-chunk-podcast \
  -episode-id="test-episode-123" \
  -transcript-dir="test/transcripts" \
  -chunk-dir="test/chunks"
```

## 11. Custom Chunking Parameters

Modify chunking behavior for different use cases.

```bash
# Edit the binary source
# cmd/raggo-chunk-podcast/main.go
# Modify defaults:
#   -window=120.0    # 2-minute chunks instead of 60s
#   -overlap=10.0    # 10-second overlap instead of 5s

# Rebuild binary
make build

# Or modify Makefile to pass custom flags:
$(BIN_DIR)/raggo-chunk-podcast \
  -episode-id="$*" \
  -window=120.0 \
  -overlap=10.0 \
  ...
```

## 12. Production Deployment

### Recommended Setup

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    volumes:
      - /var/lib/qdrant:/qdrant/storage
    ports:
      - "6333:6333"
      - "6334:6334"
    restart: always
```

### Production Pipeline with Monitoring

```bash
#!/bin/bash
# production-ingest.sh

set -e

FEED_URL="https://production-feed.com/rss"
LOG_DIR="logs/$(date +%Y%m%d)"
mkdir -p "$LOG_DIR"

echo "Starting ingestion at $(date)"

# Run pipeline with logging
make podcast \
  FEED_URL="$FEED_URL" \
  STT_ENDPOINT="http://stt-service:8000/stt" \
  EMBED_ENDPOINT="http://embed-service:8001/embed" \
  2>&1 | tee "$LOG_DIR/pipeline.log"

# Monitor disk usage
du -sh data/podcast/audio >> "$LOG_DIR/disk-usage.log"
du -sh data/podcast/embeddings >> "$LOG_DIR/disk-usage.log"

# Check Qdrant health
curl -s http://localhost:6333/healthz | tee "$LOG_DIR/qdrant-health.log"

echo "Completed at $(date)"
```

### Backup Artifacts

```bash
# Daily backup
tar -czf backup-$(date +%Y%m%d).tar.gz \
  data/podcast/manifests \
  data/podcast/chunks \
  data/podcast/embeddings

# Upload to remote storage
aws s3 cp backup-$(date +%Y%m%d).tar.gz s3://my-bucket/raggo-backups/
```

## 13. Batch Reprocessing

Reprocess all episodes with updated pipeline stages.

```bash
# Scenario: New normalization rules
rm data/podcast/transcripts/*.normalized.json
rm data/podcast/chunks/*.jsonl
rm data/podcast/embeddings/*.jsonl
rm data/podcast/index/*.done

# Re-run from normalization onwards
cd pipelines/podcast
make normalize FEED_URL="..."
make chunks FEED_URL="..."
make embeddings FEED_URL="..." EMBED_ENDPOINT="..."
make index FEED_URL="..."
```

## 14. Debugging Failed Episodes

Identify and fix issues with specific episodes.

```bash
# Run with verbose logging
make transcripts FEED_URL="..." 2>&1 | grep ERROR

# Check for missing artifacts
for episode in data/podcast/audio/*.mp3; do
  episode_id=$(basename "$episode" .mp3)
  if [ ! -f "data/podcast/transcripts/${episode_id}.json" ]; then
    echo "Missing transcript: $episode_id"
  fi
done

# Re-run single episode manually
bin/raggo-stt-audio \
  -episode-id="problematic-episode-123" \
  -audio-dir="data/podcast/audio" \
  -transcript-dir="data/podcast/transcripts" \
  -stt-endpoint="http://localhost:8000/stt"
```

## 15. Cleaning Up

Remove artifacts selectively or completely.

```bash
# Remove all data artifacts (preserves binaries)
make clean

# Remove everything including binaries
make clean-all

# Remove specific stage artifacts
rm -rf data/podcast/audio/*.mp3
rm -rf data/podcast/transcripts/*.json
rm -rf data/podcast/chunks/*.jsonl

# Clean old artifacts (older than 30 days)
find data/podcast/transcripts -name "*.json" -mtime +30 -delete
find data/podcast/audio -name "*.mp3" -mtime +30 -delete

# Reset Qdrant collection
curl -X DELETE http://localhost:6333/collections/raggo
```

## 16. Multi-Feed Management

Process and organize multiple podcast feeds with separate corpora.

```bash
# Process feeds with different corpus identifiers
make podcast FEED_URL="https://feed1.com/rss" CORPUS="tech-podcast"
make podcast FEED_URL="https://feed2.com/rss" CORPUS="news-podcast"
make podcast FEED_URL="https://feed3.com/rss" CORPUS="education-podcast"

# Query specific corpus in Qdrant
curl -X POST http://localhost:6333/collections/raggo/points/scroll \
  -H 'Content-Type: application/json' \
  -d '{
    "filter": {
      "must": [
        {"key": "corpus", "match": {"value": "tech-podcast"}}
      ]
    },
    "limit": 10
  }'
```

## External Service API Examples

### STT Service Request/Response

**Expected Request:**
```json
POST /stt
{
  "audio_path": "/path/to/audio.mp3"
}
```

**Expected Response:**
```json
{
  "segments": [
    {
      "start": 0.0,
      "end": 10.5,
      "text": "Transcript text",
      "speaker": "Speaker 1"
    }
  ],
  "model": "whisper-1",
  "version": "v1"
}
```

### Embedding Service Request/Response

**Expected Request:**
```json
POST /embed
{
  "text": "Text to embed",
  "model": "text-embedding-3-small"
}
```

**Expected Response:**
```json
{
  "embedding": [0.123, -0.456, ...],
  "model": "text-embedding-3-small",
  "version": "v1"
}
```

## Troubleshooting Common Issues

### "Episode already indexed"

```bash
rm data/podcast/index/<episode_id>.done
make index FEED_URL="..."
```

### Empty or Placeholder Transcripts

```bash
# Ensure STT_ENDPOINT is set and accessible
curl -X POST $STT_ENDPOINT -d '{"audio_path": "test.mp3"}'

# Re-run transcription
rm data/podcast/transcripts/<episode_id>.json
make transcripts FEED_URL="..." STT_ENDPOINT="..."
```

### Qdrant Connection Refused

```bash
# Ensure Qdrant is running
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# Or with docker-compose
docker-compose up -d
```

### Missing Dependencies

```bash
make install-deps
go mod tidy
```

### Permission Denied on Scripts

```bash
chmod +x scripts/*.sh
```
