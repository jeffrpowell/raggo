# Raggo: Containerized Multi-Mode RAG Pipeline

Raggo is a containerized RAG ingestion pipeline that orchestrates multi-corpus indexing with support for one-shot, scheduled, and watch modes. It transforms podcast RSS feeds and document directories into searchable vector embeddings through a configurable, artifact-based pipeline.

RAG-Go, get it?

**Key Characteristics:**
- **Artifact-first**: Every stage consumes and produces files on disk
- **Inspectable**: All intermediate outputs are human-readable JSON/JSONL
- **Multi-corpus**: Process multiple data sources simultaneously
- **Flexible modes**: One-shot, scheduled (cron), or watch (change detection)
- **Containerized**: Single Docker image with all pipeline binaries
- **Stateful**: Track runs and watch state in Qdrant metadata

**Documentation:**
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design, philosophy, and structure
- [docs/orchestrator.md](docs/orchestrator.md) - Orchestrator configuration and modes
- [docs/migration.md](docs/migration.md) - Migration from Make-based workflow
- [EXAMPLES.md](EXAMPLES.md) - Comprehensive usage examples
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development setup and guidelines

## Pipeline Overview

Raggo supports two pipeline types:

**Podcast Pipeline** (8 stages):
1. **RSS Ingestion** - Parse feed, extract episode metadata
2. **Audio Download** - Download audio files with parallel workers
3. **Speech-to-Text** - Transcribe audio via external STT service
4. **Normalization** - Normalize transcripts (punctuation, whitespace, speakers)
5. **Chunking** - Create time-windowed chunks (default: 60s windows, 5s overlap)
6. **Contextualize** - LLM writes a short context situating each chunk in its episode (optional)
7. **Embedding** - Generate embeddings of context + chunk via external service
8. **Indexing** - Upsert dense + BM25 vectors into Qdrant with metadata

**Document Pipeline** (6 stages):
1. **Scan** - Discover documents in directory
2. **Extract** - Extract text using Tika and optional vision OCR
3. **Chunk** - Split into character-based chunks
4. **Contextualize** - LLM writes a short context situating each chunk in its document (optional)
5. **Embed** - Generate embeddings of context + chunk via external service
6. **Index** - Upsert dense + BM25 vectors into Qdrant with metadata

Contextualization follows Anthropic's [Contextual Retrieval](https://www.anthropic.com/engineering/contextual-retrieval), adapted for small local models whose context window can't hold a whole document. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#contextual-retrieval). For the query side (hybrid search, RRF fusion, reranking), see [examples/open-webui](examples/open-webui/) for Open WebUI chat and [examples/open-webui-computer](examples/open-webui-computer/) for Open WebUI Computer agents.

The orchestrator manages pipeline execution with automatic retries, state tracking, and multiple operating modes. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design.

## Quick Start

### Orchestrator (Recommended for Production)

**Prerequisites:** Docker and Docker Compose

```bash
# 1. Clone and create configuration
git clone https://github.com/jeffrpowell/raggo.git
cd raggo
cp config/raggo.example.yml config/raggo.yml

# 2. Edit config/raggo.yml with your corpus
# - Set your RSS feed URL or document directory
# - Configure STT and embedding endpoints
# - Choose mode: one-shot, scheduled, or watch

# 3. Start orchestrator and Qdrant
docker compose -f docker-compose.orchestrator.yml up -d

# 4. Monitor progress
docker compose -f docker-compose.orchestrator.yml logs -f raggo

# 5. Query indexed data
curl http://localhost:6333/collections/my-podcast
```

**Operating Modes:**
- **One-shot** - Run once and exit
- **Scheduled** - Run on cron schedule (e.g., `"0 2 * * *"` for daily at 2 AM)
- **Watch** - Monitor RSS feeds or directories for changes

**Configuration Reference:**

The orchestrator uses a single `raggo.yml` file with:
- **Storage paths** - Configure where artifacts are stored
- **Qdrant connection** - Host, port, metadata collection
- **External services** - STT and embedding endpoints  
- **Orchestrator settings** - Retry limits, log directory
- **Corpora definitions** - Multi-corpus with different modes

See [config/raggo.example.yml](config/raggo.example.yml) for a complete configuration template and [docs/EXAMPLES.md](docs/EXAMPLES.md) for detailed examples.

### Local Development

For local development and testing of individual binaries:

```bash
# Prerequisites: Go 1.21+, Docker
git clone https://github.com/jeffrpowell/raggo.git
cd raggo

# Install dependencies
go mod download

# Build a specific binary for testing
go build -o bin/raggo-rss-podcast ./cmd/raggo-rss-podcast

# Or build the orchestrator
go build -o bin/raggo-orchestrator ./cmd/raggo-orchestrator

# Start Qdrant for testing
docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Testing Individual Binaries

For development and debugging, you can run binaries directly:

```bash
# Test with config file
bin/raggo-rss-podcast \
  -config config/raggo.yml \
  -corpus-id tech-podcast \
  -feed-url "https://feeds.example.com/podcast.rss"

# Or with flags only
bin/raggo-chunk-podcast \
  -episode-id "episode-123" \
  -transcript-dir "data/transcripts" \
  -chunk-dir "data/chunks" \
  -window 120.0 \
  -overlap 10.0
```

See [docs/EXAMPLES.md](docs/EXAMPLES.md) for comprehensive usage patterns.

## Inspecting Artifacts

All pipeline stages produce human-readable JSON/JSONL files:

```bash
# View episode manifests
cat data/podcast/manifests/*.jsonl | jq .title

# View chunks
cat data/podcast/chunks/<episode_id>.jsonl | jq -c

# View generated chunk contexts (mode is "full" or "windowed")
cat data/podcast/contexts/<episode_id>.jsonl | jq -r '[.mode, .context] | @tsv'

# Check embedding dimensions
cat data/podcast/embeddings/<episode_id>.jsonl | jq .dimension | head -1

# Query Qdrant collections
curl http://localhost:6333/collections/tech-podcast | jq
```

## Configuration

Common configuration variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `FEED_URL` | (required) | RSS feed URL |
| `STT_ENDPOINT` | (none) | Speech-to-text HTTP endpoint |
| `EMBED_ENDPOINT` | (none) | Embedding service HTTP endpoint |
| `QDRANT_HOST` | `localhost` | Qdrant server host |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `COLLECTION` | `raggo` | Qdrant collection name |
| `CORPUS` | `default` | Corpus identifier |
| `WORKERS` | `4` | Concurrent workers |

For external service API specifications, see [EXAMPLES.md](EXAMPLES.md).

## Common Operations

**Re-run a stage:**
```bash
rm data/podcast/chunks/<episode_id>.jsonl
bin/raggo-chunk-podcast \
  -episode-id "episode-123" \
  -transcript-dir "data/transcripts" \
  -chunk-dir "data/chunks" \
  -window 120.0 \
  -overlap 10.0
```

**Parallel processing:**
```bash
bin/raggo-transcribe-podcast \
  -episode-id "episode-123" \
  -transcript-dir "data/transcripts" \
  -stt-endpoint "http://localhost:8000/stt" \
  -workers 4
```

**Clean up:**
```bash
# Remove data artifacts
rm -rf data/podcast

# Remove everything including binaries
rm -rf data bin
```

For comprehensive examples and workflows, see [EXAMPLES.md](EXAMPLES.md).

## Extending Raggo

The system is designed for extension:

- **New source types**: Add binaries in `cmd/`, define pipeline in `pkg/pipeline/`, configure in `raggo.yml`
- **Replace services**: Update endpoints in `raggo.yml` or fork existing binaries
- **Modify chunking**: Configure in `raggo.yml` or override per-corpus

See [ARCHITECTURE.md](ARCHITECTURE.md) for extension patterns and [CONTRIBUTING.md](CONTRIBUTING.md) for development practices.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Empty transcripts | Configure `stt_endpoint` in `raggo.yml` |
| Qdrant connection refused | Ensure Qdrant is running: `docker compose up -d` |
| Missing dependencies | Run `go mod download && go mod tidy` |
| Re-run corpus | Update config and restart orchestrator |

For detailed debugging workflows, see [EXAMPLES.md](EXAMPLES.md).

## License

MIT

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and contribution guidelines.
