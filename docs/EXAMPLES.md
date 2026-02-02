# Raggo Usage Examples

This document provides comprehensive usage examples for the Raggo pipeline, ordered from most common to advanced use cases.

## Orchestrator Examples

### 1. One-Shot Mode (Run Once)

Process a corpus once and exit:

```yaml
# config/raggo.yml
corpora:
  - id: my-podcast
    type: podcast
    mode: one-shot
    config:
      feed_url: https://feeds.example.com/podcast.rss
      collection: my-podcast
      corpus: my-podcast-v1
      workers: 4
```

```bash
docker compose -f docker-compose.orchestrator.yml up
```

### 2. Scheduled Mode (Cron)

Run on a schedule:

```yaml
corpora:
  - id: daily-podcast
    type: podcast
    mode: scheduled
    schedule: "0 2 * * *"  # Daily at 2 AM
    config:
      feed_url: https://feeds.example.com/podcast.rss
      collection: daily-podcast
      corpus: daily-podcast-v1
```

### 3. Watch Mode (RSS Feed)

Monitor RSS feed for new episodes:

```yaml
corpora:
  - id: watched-podcast
    type: podcast
    mode: watch
    watch_interval_seconds: 300  # Check every 5 minutes
    config:
      feed_url: https://feeds.example.com/podcast.rss
      collection: watched-podcast
      corpus: watched-podcast-v1
```

### 4. Watch Mode (Documents)

Monitor directory for file changes:

```yaml
corpora:
  - id: company-docs
    type: documents
    mode: watch
    watch_interval_seconds: 300
    config:
      root_path: /mnt/documents
      collection: company-docs
      corpus: company-docs-v1
      chunk_size: 1000
      overlap_size: 100
```

### 5. Multi-Corpus Configuration

Process multiple corpora with different modes:

```yaml
corpora:
  - id: tech-podcast
    type: podcast
    mode: scheduled
    schedule: "0 */6 * * *"
    config:
      feed_url: https://feeds.example.com/tech.rss
      collection: tech-podcast
      corpus: tech-podcast-v1

  - id: news-podcast
    type: podcast
    mode: watch
    watch_interval_seconds: 600
    config:
      feed_url: https://feeds.example.com/news.rss
      collection: news-podcast
      corpus: news-podcast-v1

  - id: internal-docs
    type: documents
    mode: watch
    watch_interval_seconds: 300
    config:
      root_path: /mnt/documents
      collection: internal-docs
      corpus: internal-docs-v1
```

### 6. Orchestrator Configuration Reference

**Full configuration schema:**

```yaml
storage:
  root: /var/lib/raggo
  retain_intermediates: false       # Keep intermediate files after indexing
  podcast:
    audio: ${root}/podcast/audio
    transcripts: ${root}/podcast/transcripts
    chunks: ${root}/podcast/chunks
    embeddings: ${root}/podcast/embeddings
    index: ${root}/podcast/index
  documents:
    sources: /mnt/documents
    text: ${root}/documents/text
    chunks: ${root}/documents/chunks
    embeddings: ${root}/documents/embeddings
    index: ${root}/documents/index

qdrant:
  host: qdrant
  port: 6334
  metadata_collection: raggo_metadata

services:
  stt_endpoint: http://stt-service:8000/transcribe
  embed_endpoint: http://embed-service:8001/embed

orchestrator:
  retry_limit: 3                    # Maximum retry attempts per corpus
  retry_delay_seconds: 60           # Delay between retries
  log_dir: /var/log/raggo          # Log output directory

corpora:
  - id: tech-podcast
    type: podcast                   # podcast or documents
    mode: scheduled                 # one-shot, scheduled, or watch
    schedule: "0 */6 * * *"        # Required for scheduled mode
    config:
      feed_url: https://feeds.example.com/tech.rss
      collection: tech-podcast
      corpus: tech-podcast-v1
      workers: 4
```

### 7. Environment Variables

**Orchestrator environment variables:**

```bash
# In docker-compose.yml
environment:
  - RAGGO_CONFIG=/etc/raggo/raggo.yml  # Config file path
  - TZ=America/Denver                   # Timezone for scheduled jobs
```

### 8. Monitoring and Logging

**View orchestrator logs:**
```bash
docker compose logs -f raggo
```

**Log format:**
```
[2024-01-15 10:30:00] [INFO] [raggo-orchestrator] Starting one-shot mode with 2 corpora
[2024-01-15 10:30:01] [INFO] [raggo-orchestrator] Pipeline attempt 1/3 for corpus: tech-podcast
[2024-01-15 10:30:05] [INFO] [raggo-orchestrator] Executing stage: rss
[2024-01-15 10:31:20] [INFO] [raggo-orchestrator] Stage completed: rss
```

**Query run state in Qdrant:**
```bash
# Get metadata collection info
curl http://localhost:6333/collections/raggo_metadata

# Query recent runs for a corpus
curl -X POST http://localhost:6333/collections/raggo_metadata/points/scroll \
  -H 'Content-Type: application/json' \
  -d '{
    "filter": {
      "must": [{"key": "corpus_id", "match": {"value": "tech-podcast"}}]
    },
    "limit": 10
  }'
```

**Monitor container health:**
```bash
# Check container status
docker compose ps

# Check resource usage
docker stats raggo

# Follow all logs
docker compose logs -f
```

### Orchestrator Troubleshooting

**No logs appearing:**
```bash
docker compose ps
docker compose logs raggo
```

**Config validation errors:**
```bash
# Validate YAML syntax
docker compose config

# Check for validation errors
docker compose logs raggo | grep -i "error\|failed"
```

**Pipeline fails immediately:**
```bash
# Check configuration validation
docker compose logs raggo | grep ERROR

# Verify config file is mounted
docker compose exec raggo ls -la /etc/raggo/
```

**Watch mode not triggering:**
```bash
# Check watch interval is reasonable
# Verify RSS feed is accessible
curl https://feeds.example.com/podcast.rss

# Ensure filesystem permissions for document watch
docker compose exec raggo ls -la /mnt/documents
```

**Missing external services:**
```bash
# Test STT endpoint
curl -X POST http://stt-service:8000/transcribe -d '{"audio_path": "test.mp3"}'

# Test embed endpoint  
curl -X POST http://embed-service:8001/embed -d '{"text": "test", "model": "test"}'
```

**Query indexed data:**
```bash
curl http://localhost:6333/collections/my-podcast
curl -X POST http://localhost:6333/collections/my-podcast/points/search \
  -H 'Content-Type: application/json' \
  -d '{"vector": [0.1, 0.2, ...], "limit": 5, "with_payload": true}'
```

---

## Direct Binary Usage (Development/Debugging)

For development and debugging, you can run individual binaries directly with config files or flags:

### 9. Testing Individual Binaries

```bash
# Build binaries
go build -o bin/raggo-rss-podcast ./cmd/raggo-rss-podcast
go build -o bin/raggo-chunk-podcast ./cmd/raggo-chunk-podcast

# Test RSS ingestion
bin/raggo-rss-podcast \
  -config config/raggo.yml \
  -corpus-id tech-podcast \
  -feed-url "https://feeds.example.com/podcast.rss"

# Test chunking with custom parameters
bin/raggo-chunk-podcast \
  -config config/raggo.yml \
  -corpus-id tech-podcast \
  -episode-id "episode-123" \
  -window 120.0 \
  -overlap 10.0

# Test with local files
bin/raggo-rss-podcast \
  -feed-url "file://$(pwd)/test/fixtures/feed.xml" \
  -manifest-dir "test/output"
```

### 10. Inspecting Pipeline Artifacts

Examine intermediate outputs to understand or debug the pipeline.

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

### Query Qdrant Collections

```bash
# Check collection info
curl http://localhost:6333/collections/tech-podcast

# Count indexed points
curl http://localhost:6333/collections/tech-podcast | jq '.result.points_count'

# View collection configuration
curl http://localhost:6333/collections/tech-podcast | jq .result.config

# Query specific corpus
curl -X POST http://localhost:6333/collections/tech-podcast/points/scroll \
  -H 'Content-Type: application/json' \
  -d '{
    "filter": {
      "must": [
        {"key": "corpus", "match": {"value": "tech-podcast-v1"}}
      ]
    },
    "limit": 10
  }'
```

---

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

### Qdrant Connection Refused

```bash
# Ensure Qdrant is running
docker compose ps

# Check Qdrant health
curl http://localhost:6333/healthz
```

### Missing Dependencies

```bash
go mod download
go mod tidy
```

### Build Failures

```bash
# Clean and rebuild
go clean -cache
go build ./cmd/raggo-orchestrator
```
