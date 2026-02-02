# Raggo: Deterministic RAG Ingestion Pipeline

Raggo is a deterministic, artifact-first RAG ingestion pipeline built from composable CLI tools orchestrated by GNU Make. It transforms podcast RSS feeds into searchable vector embeddings through a transparent, file-based pipeline.

RAG-Go, get it?

**Key Characteristics:**
- **Artifact-first**: Every stage consumes and produces files on disk
- **Inspectable**: All intermediate outputs are human-readable JSON/JSONL
- **Deterministic**: Same inputs → same outputs, always
- **Composable**: Swap out stages independently without breaking the pipeline
- **Re-runnable**: Delete an artifact and re-run `make` to regenerate it

**Documentation:**
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design, philosophy, and structure
- [EXAMPLES.md](EXAMPLES.md) - Comprehensive usage examples
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development setup and guidelines

## Pipeline Overview

The podcast pipeline transforms RSS feeds into searchable vectors through seven stages:

1. **RSS Ingestion** - Parse feed, extract episode metadata
2. **Audio Download** - Download audio files with parallel workers
3. **Speech-to-Text** - Transcribe audio via external STT service
4. **Normalization** - Normalize transcripts (punctuation, whitespace, speakers)
5. **Chunking** - Create time-windowed chunks (default: 60s windows, 5s overlap)
6. **Embedding** - Generate embeddings via external service
7. **Indexing** - Upsert vectors into Qdrant with metadata

Each stage reads from and writes to the `data/podcast/` directory. See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed data flow and schemas.

## Quick Start

### Option 1: Devcontainer (Recommended)

```bash
# Open in VS Code
code /path/to/raggo

# Reopen in container when prompted
# Everything is automatically configured!
```

The devcontainer includes Go 1.21, Qdrant, and all dependencies pre-configured. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Option 2: Local Development

```bash
# Prerequisites: Go 1.21+, GNU Make, jq, Docker
git clone https://github.com/jeffrpowell/raggo.git
cd raggo
make install-deps
make build
docker-compose up -d  # Start Qdrant
```

### Run Your First Pipeline

**Basic test (with placeholders):**
```bash
make podcast FEED_URL="https://feeds.simplecast.com/54nAGcIl"
```

**With real services:**
```bash
make podcast \
  FEED_URL="https://example.com/podcast.rss" \
  STT_ENDPOINT="http://localhost:8000/stt" \
  EMBED_ENDPOINT="http://localhost:8001/embed"
```

**Run individual stages:**
```bash
cd pipelines/podcast
make rss FEED_URL="..."
make audio FEED_URL="..."
make transcripts FEED_URL="..." STT_ENDPOINT="..."
# ... and so on
```

See [EXAMPLES.md](EXAMPLES.md) for comprehensive usage patterns.

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
make chunks FEED_URL="..."
```

**Parallel processing:**
```bash
make -j 4 transcripts FEED_URL="..."
```

**Inspect artifacts:**
```bash
cat data/podcast/manifests/<feed_hash>.jsonl | jq
cat data/podcast/chunks/<episode_id>.jsonl | jq -c
```

**Clean up:**
```bash
make clean        # Remove data artifacts
make clean-all    # Remove everything including binaries
```

For comprehensive examples and workflows, see [EXAMPLES.md](EXAMPLES.md).

## Extending Raggo

The system is designed for extension:

- **New source types**: Add binaries in `cmd/`, create pipeline Makefile, define schemas
- **Replace services**: Wrap your service in HTTP API or fork existing binaries
- **Modify chunking**: Edit binary source or Makefile parameters

See [ARCHITECTURE.md](ARCHITECTURE.md) for extension patterns and [CONTRIBUTING.md](CONTRIBUTING.md) for development practices.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Empty transcripts | Configure `STT_ENDPOINT` or use placeholders for testing |
| Qdrant connection refused | Ensure Qdrant is running: `docker-compose up -d` |
| Missing dependencies | Run `make install-deps && go mod tidy` |
| Re-index episode | Delete marker: `rm data/podcast/index/<episode_id>.done` |

For detailed debugging workflows, see [EXAMPLES.md](EXAMPLES.md).

## License

MIT

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and contribution guidelines.
