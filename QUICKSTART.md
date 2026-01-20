# Raggo Quick Start Guide

## Prerequisites Check

### Option 1: Using Devcontainer (Recommended)

```bash
# Open in VS Code
code /home/jeffpowell/dev/jeffrpowell/raggo

# Reopen in container when prompted
# Everything is automatically configured!
```

### Option 2: Local Development

```bash
# Verify Go installation
go version  # Should be 1.21+

# Verify Make installation
make --version

# Verify jq (for JSON inspection)
jq --version
```

## 1. Setup

```bash
# Navigate to raggo directory
cd /home/jeffpowell/dev/jeffrpowell/raggo

# Download dependencies
make install-deps

# Build all binaries
make build

# Verify binaries
ls -lh bin/
```

## 2. Start Qdrant (Required for indexing)

**If using devcontainer:** Qdrant is already running!

**If using local development:**
```bash
# Using Docker Compose
docker-compose up -d

# Verify Qdrant is running
curl http://localhost:6333/collections
```

## 3. Run Your First Pipeline

### Option A: Test with Placeholder Services (No External APIs)

This will work immediately but generate placeholder transcripts and embeddings:

```bash
make podcast FEED_URL="https://feeds.simplecast.com/54nAGcIl"
```

**What happens:**
- ✅ RSS feed is fetched and parsed
- ✅ Audio files are downloaded
- ⚠️  Placeholder transcripts are generated (STT not configured)
- ✅ Transcripts are normalized
- ✅ Chunks are created
- ⚠️  Placeholder embeddings are generated (embedding service not configured)
- ✅ Everything is indexed into Qdrant

### Option B: With Real STT and Embeddings

**Set up external services:**

```bash
# Example: Start a Whisper STT service
# (You'll need to implement or use an existing service)
python3 -m whisper_server --port 8000

# Example: Start an embedding service
# (You'll need to implement or use an existing service)
python3 -m embedding_server --port 8001
```

**Run pipeline with services:**

```bash
make podcast \
  FEED_URL="https://feeds.simplecast.com/54nAGcIl" \
  STT_ENDPOINT="http://localhost:8000/stt" \
  EMBED_ENDPOINT="http://localhost:8001/embed"
```

## 4. Inspect Results

### Check Pipeline Status

```bash
# Get feed hash (from RSS ingestion output)
FEED_HASH=$(ls data/podcast/manifests/ | head -1 | sed 's/.jsonl//')

# Verify pipeline completion
./scripts/verify-pipeline.sh $FEED_HASH

# Check if full pipeline completed
ls -lh data/podcast/index/qdrant.done
```

### List Episodes

```bash
./scripts/list-episodes.sh $FEED_HASH
```

### Inspect Specific Episode

```bash
# Get an episode ID from the manifest
EPISODE_ID=$(jq -r '.episode_id' data/podcast/manifests/$FEED_HASH.jsonl | head -1)

# Inspect all artifacts
./scripts/inspect-episode.sh $EPISODE_ID
```

### Query Qdrant

```bash
# Check collection info
curl http://localhost:6333/collections/raggo

# Count points
curl http://localhost:6333/collections/raggo | jq '.result.points_count'
```

## 5. Re-run Specific Stages

### Re-run a Single Episode

```bash
# Remove artifacts for one episode
./scripts/clean-episode.sh $EPISODE_ID

# Re-run pipeline
make podcast FEED_URL="..."
```

### Re-run a Stage for All Episodes

```bash
# Example: Re-chunk everything with different parameters
rm data/podcast/chunks/*.jsonl
rm data/podcast/index/qdrant.done

# Re-run from chunking onwards
cd pipelines/podcast
make chunks FEED_URL="..."
```

## 6. Advanced Usage

### Parallel Processing

```bash
# Download and transcribe 8 episodes concurrently
cd pipelines/podcast
make -j 8 transcripts FEED_URL="..." STT_ENDPOINT="..."
```

### Using Configuration File

```bash
# Copy example config
cp config.example.mk config.mk

# Edit config.mk with your settings
nano config.mk

# Run with config
make pipeline -f config.mk
```

### Custom Chunking Parameters

```bash
# Edit cmd/raggo-chunk-podcast/main.go to change defaults
# Or modify the binary call in Makefile:
#   -window=120.0    # 2-minute chunks
#   -overlap=10.0    # 10-second overlap
```

## 7. Troubleshooting

### "go: command not found"

Install Go from https://golang.org/dl/

### "Manifest not found"

You need to run RSS ingestion first:
```bash
make rss FEED_URL="..."
```

### "No episodes found"

The manifest might be empty. Check:
```bash
cat data/manifests/*.jsonl
```

### Transcripts are placeholders

You haven't configured `STT_ENDPOINT`. Either:
1. Set up an STT service and configure the endpoint
2. Continue with placeholders for testing

### Qdrant connection refused

Start Qdrant:
```bash
docker-compose up -d
```

### Permission denied on scripts

Make them executable:
```bash
chmod +x scripts/*.sh
```

## 8. Production Deployment

### Recommended Setup

1. **Run Qdrant in production mode:**
   ```yaml
   # docker-compose.prod.yml
   services:
     qdrant:
       image: qdrant/qdrant:latest
       volumes:
         - /var/lib/qdrant:/qdrant/storage
       restart: always
   ```

2. **Use real STT and embedding services:**
   - OpenAI Whisper API
   - OpenAI Embeddings API
   - Or self-hosted Whisper + SentenceTransformers

3. **Schedule periodic ingestion:**
   ```bash
   # crontab -e
   0 */6 * * * cd /path/to/raggo && make pipeline FEED_URL="..."
   ```

4. **Monitor disk usage:**
   ```bash
   du -sh data/audio
   du -sh data/embeddings
   ```

5. **Backup artifacts:**
   ```bash
   tar -czf backup-$(date +%Y%m%d).tar.gz data/
   ```

## Next Steps

- Read `README.md` for full documentation
- Read `ARCHITECTURE.md` for design principles
- Explore `cmd/` to understand each stage
- Modify stages to fit your needs
- Add new source types following the same patterns

## Example Workflows

### Workflow 1: Daily Podcast Ingestion

```bash
#!/bin/bash
# daily-ingest.sh

FEEDS=(
  "https://feeds.example.com/podcast1.rss"
  "https://feeds.example.com/podcast2.rss"
)

for feed in "${FEEDS[@]}"; do
  echo "Processing: $feed"
  make pipeline \
    FEED_URL="$feed" \
    STT_ENDPOINT="http://stt-service:8000/stt" \
    EMBED_ENDPOINT="http://embed-service:8001/embed"
done
```

### Workflow 2: Incremental Updates

```bash
# Only process new episodes by checking existing manifests
make rss FEED_URL="..."

# Make will automatically skip existing audio/transcripts/etc
make pipeline FEED_URL="..."
```

### Workflow 3: Batch Reprocessing

```bash
# Re-embed everything with a better model
rm data/podcast/embeddings/*.jsonl
rm data/podcast/index/*.done

# Re-run with new embedding endpoint
cd pipelines/podcast
make embeddings FEED_URL="..." EMBED_ENDPOINT="http://new-model:8001/embed"
make index FEED_URL="..."
```
