# Raggo Architecture

## Design Principles

### 1. Artifact-First Design

Every pipeline stage operates on the principle of **consuming files** and **producing files**. No stage maintains internal state or relies on in-memory communication.

**Benefits**:
- **Inspectability**: `cat`, `jq`, `grep` work on all intermediate outputs
- **Debuggability**: Examine exact inputs/outputs at each stage
- **Reproducibility**: Same input files → same output files
- **Disaster recovery**: No hidden database state to corrupt

**Example**:
```bash
# Inspect what episodes were discovered
cat data/manifests/<feed_hash>.jsonl | jq .title

# Check a specific chunk's content
cat data/chunks/<episode_id>.jsonl | jq -r 'select(.chunk_index == 5) | .text'

# Verify embedding dimensions
cat data/embeddings/<episode_id>.jsonl | jq .dimension | head -1
```

### 2. Stage Isolation

Each CLI binary is a **standalone program** with zero knowledge of other stages.

**Benefits**:
- **Testability**: Run and test each stage independently
- **Composability**: Reuse stages across different pipelines
- **Replaceability**: Swap out implementations without breaking others
- **Resource isolation**: Memory-intensive STT doesn't affect lightweight chunking

**Example**:
```bash
# Run normalization standalone
bin/raggo-normalize-podcast \
  -episode-id="abc123" \
  -transcript-dir="data/transcripts"

# Replace STT with different implementation
bin/my-custom-stt \
  -episode-id="abc123" \
  -audio-path="data/audio/abc123.mp3" \
  -transcript-dir="data/transcripts"
```

### 3. Makefile Orchestration

GNU Make provides the **control plane** for pipeline execution, with a **two-tier structure**:

**Top-level Makefile** (minimal delegator):
```makefile
podcast:
	$(MAKE) -C pipelines/podcast
```

**Pipeline-specific Makefiles** (complete DAG):
- Located in `pipelines/<type>/Makefile`
- Define all targets for that pipeline
- Use fully qualified artifact paths (`../../data/podcast/...`)
- End in a terminal `.done` target (`data/podcast/index/qdrant.done`)

**Benefits**:
- **Declarative DAG**: Dependencies are explicit, not hidden in code
- **Automatic caching**: Make uses timestamps to skip unchanged work
- **Parallel execution**: Built-in `-j` flag for concurrency
- **Pipeline isolation**: Each pipeline is self-contained
- **Universal**: No custom scheduler to debug
- **Version controlled**: Pipeline logic is code

**Example**:
```makefile
# In pipelines/podcast/Makefile
all: $(INDEX_DIR)/qdrant.done

$(CHUNK_DIR)/%.jsonl: $(TRANSCRIPT_DIR)/%.normalized.json
	$(BIN_DIR)/raggo-chunk-podcast -episode-id="$*" ...

$(INDEX_DIR)/qdrant.done: $(INDEX_FILES)
	@touch $@
```

### 4. Go as Control Language

Go provides the implementation language for orchestration, I/O, and concurrency.

**Benefits**:
- **Strong typing**: Schemas catch errors at compile time
- **Explicit errors**: No silent failures
- **Bounded concurrency**: Worker pools prevent resource exhaustion
- **Fast iteration**: Compiled binaries with no runtime dependencies

**Non-goals**:
- ❌ Native ML inference (use external APIs instead)
- ❌ Dynamic pipeline configuration (static stages preferred)
- ❌ Hidden abstractions (explicit is better)

## Data Flow

```
┌─────────────────┐
│   RSS Feed      │
│   (External)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      data/raw/*.xml
│  raggo-rss-     │      data/raw/*.json
│  podcast        │───►  data/manifests/*.jsonl
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/audio/*.mp3
│  raggo-download-│───►  data/audio/*.json
│  audio          │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      
│  External STT   │      
│  Service        │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/transcripts/*.json
│  raggo-stt-     │───►  
│  audio          │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/transcripts/*.normalized.json
│  raggo-         │───►  
│  normalize-     │      
│  podcast        │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/chunks/*.jsonl
│  raggo-chunk-   │───►  
│  podcast        │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      
│  External       │      
│  Embedding      │      
│  Service        │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/embeddings/*.jsonl
│  raggo-embed-   │───►  
│  podcast        │      
└────────┬────────┘      
         │
         ▼
┌─────────────────┐      data/index/*.done
│  raggo-index-   │───►  
│  podcast        │      Qdrant
└────────┬────────┘      Vector DB
         │
         ▼
    ┌────────┐
    │ Qdrant │
    │ Index  │
    └────────┘
```

## Naming Conventions

### Binary Names

Format: `raggo-<action>-<source-type>`

- `raggo-rss-podcast` - RSS-specific ingestion
- `raggo-download-audio` - Generic audio downloader
- `raggo-stt-audio` - Generic audio transcription
- `raggo-normalize-podcast` - Podcast-specific normalization
- `raggo-chunk-podcast` - Podcast-specific chunking
- `raggo-embed-podcast` - Podcast text embedding (reusable for other text sources)
- `raggo-index-podcast` - Podcast indexing

### Artifact Paths

Format: `data/<pipeline>/<stage>/<identifier>.<extension>`

- `data/podcast/raw/<feed_hash>.xml` - Immutable feed snapshot
- `data/podcast/manifests/<feed_hash>.jsonl` - Line-delimited episodes
- `data/podcast/audio/<episode_id>.mp3` - Audio file
- `data/podcast/transcripts/<episode_id>.json` - Raw transcript
- `data/podcast/transcripts/<episode_id>.normalized.json` - Normalized transcript
- `data/podcast/chunks/<episode_id>.jsonl` - Chunk records
- `data/podcast/embeddings/<episode_id>.jsonl` - Embedding records
- `data/podcast/index/<episode_id>.done` - Per-episode completion marker
- `data/podcast/index/qdrant.done` - **Terminal pipeline marker**

## Concurrency Model

### Episode-Level Parallelism

Make provides **horizontal parallelism** across episodes:

```bash
# Process 4 episodes concurrently
make -j 4 transcripts FEED_URL="..."
```

### Within-Stage Parallelism

Go binaries use **bounded worker pools** for internal concurrency:

```go
pool := concurrency.NewWorkerPool(workers)
for _, episode := range episodes {
    pool.Submit(func() error {
        return processEpisode(episode)
    })
}
pool.Wait()
```

**Benefits**:
- No unbounded goroutines
- Predictable resource usage
- Graceful error handling

## Error Handling

### Fail-Fast Philosophy

Stages must **fail loudly** on errors:

```go
if err != nil {
    log.Fatal("Failed to download: %v", err)
}
```

**Benefits**:
- No silent corruption
- Clear error messages
- Make stops immediately

### Idempotency

Re-running a stage should be safe:

```go
if storage.MarkerExists(outputPath) {
    log.Info("Output already exists, skipping")
    return nil
}
```

**Benefits**:
- Re-running `make` is always safe
- Partial failures don't corrupt state
- Incremental processing

### Atomic Writes

Use temporary files for writes:

```go
tmpPath := outputPath + ".tmp"
// Write to tmpPath
os.Rename(tmpPath, outputPath)  // Atomic on POSIX
```

## Extensibility

### Adding New Source Types

To add YouTube video support:

1. **Create source-specific binaries**:
   - `cmd/raggo-youtube-download/` - Download videos
   - `cmd/raggo-extract-audio/` - Extract audio from video
   
2. **Reuse generic binaries**:
   - `raggo-stt-audio` (already generic)
   - `raggo-embed-podcast` → rename to `raggo-embed-text`
   - `raggo-index-podcast` → rename to `raggo-index`

3. **Create pipeline-specific Makefile**:
   ```makefile
   # pipelines/youtube/Makefile
   all: ../../data/youtube/index/qdrant.done
   
   $(AUDIO_DIR)/%.m4a: $(VIDEO_DIR)/%.mp4
   	$(BIN_DIR)/raggo-extract-audio ...
   ```

4. **Add top-level delegator**:
   ```makefile
   # Root Makefile
   youtube:
   	$(MAKE) -C pipelines/youtube
   ```

5. **Define new schemas**:
   ```go
   // pkg/schema/youtube.go
   type VideoMetadata struct { ... }
   ```

### Replacing External Services

To use a different STT engine:

**Option 1: HTTP Wrapper**
```bash
# Wrap your STT in HTTP service matching expected API
python my_stt_wrapper.py --port 8000
make transcripts STT_ENDPOINT="http://localhost:8000/stt"
```

**Option 2: Fork Binary**
```bash
# Copy and modify raggo-stt-audio
cp -r cmd/raggo-stt-audio cmd/raggo-stt-whisper-local
# Modify to call local Whisper binary
# Update Makefile to use new binary
```

## Testing Strategy

### Unit Tests

Test shared libraries in isolation:

```go
func TestHashString(t *testing.T) {
    hash := hashing.HashString("test")
    assert.Equal(t, expected, hash)
}
```

### Integration Tests

Test individual CLI binaries:

```bash
# Test RSS ingestion
bin/raggo-rss-podcast -feed-url="file://test/fixtures/feed.xml" -manifest-dir="test/output"
jq -e '.episode_id' test/output/*.jsonl
```

### End-to-End Tests

Test full pipeline with fixtures:

```bash
# Use test feed with 2 episodes
make pipeline FEED_URL="file://test/fixtures/small-feed.xml"
test -f data/index/*.done
```

## Operational Best Practices

### Monitoring

Monitor artifact counts:

```bash
# Count episodes per stage
find data/audio -name "*.mp3" | wc -l
find data/transcripts -name "*.json" | wc -l
find data/index -name "*.done" | wc -l
```

### Debugging

Inspect intermediate artifacts:

```bash
# Check what failed
make transcripts 2>&1 | grep ERROR

# Examine specific artifact
cat data/transcripts/<episode_id>.json | jq .segments[0]
```

### Maintenance

Clean up old artifacts:

```bash
# Remove transcripts older than 30 days
find data/transcripts -name "*.json" -mtime +30 -delete

# Re-run affected stages
make chunks FEED_URL="..."
```

## Performance Characteristics

### Bottlenecks

1. **Audio download**: Network I/O bound → increase `WORKERS`
2. **STT**: External API rate limits → batch requests
3. **Embedding**: External API rate limits → batch requests
4. **Indexing**: Qdrant network latency → batch upserts

### Optimization Strategies

1. **Parallel downloads**: `make -j 8 audio`
2. **Batch API calls**: Modify binaries to batch 10 items per request
3. **Local inference**: Run Whisper/embeddings locally to remove API limits
4. **Incremental processing**: Only process new episodes

## Security Considerations

### API Keys

Store secrets in environment or config files:

```bash
export OPENAI_API_KEY="sk-..."
export QDRANT_API_KEY="..."

# Reference in binaries via flags
bin/raggo-embed-podcast -api-key="$OPENAI_API_KEY" ...
```

### Data Privacy

All artifacts are stored locally:

- Audio files: `data/audio/` (gitignored)
- Transcripts: `data/transcripts/` (gitignored)
- Embeddings: `data/embeddings/` (gitignored)

**Recommendation**: Encrypt `data/` directory at rest if processing sensitive content.

## Future Enhancements

### Potential Additions

1. **Deduplication**: Hash-based skip of duplicate chunks
2. **Incremental updates**: Only process new episodes from feed
3. **Metadata extraction**: Speaker identification, topic modeling
4. **Quality metrics**: Transcript confidence scores, embedding quality
5. **Visualization**: Pipeline progress dashboard

### Non-Goals

1. ❌ Real-time processing (batch-oriented by design)
2. ❌ Auto-scaling (use external orchestration if needed)
3. ❌ Built-in web UI (CLI-first philosophy)
4. ❌ Plugin system (fork and modify preferred)
