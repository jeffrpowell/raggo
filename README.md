# Raggo: Deterministic RAG Ingestion Pipeline

Raggo is a **deterministic, artifact-first podcast ingestion pipeline** for Retrieval-Augmented Generation (RAG) systems. It is built as a collection of standalone Go CLI tools orchestrated via GNU Make.

## Architecture Principles

1. **Artifact-first**: Every stage consumes and produces files on disk
2. **Stage isolation**: Each CLI binary is self-contained and stateless
3. **Makefile orchestration**: GNU Make defines the DAG and ensures idempotency
4. **Inspectability**: All intermediate outputs are human-readable JSON/JSONL
5. **Re-runnability**: Delete an artifact and re-run `make` to regenerate it

## Repository Structure

```
raggo/
├── cmd/                          # CLI binaries (one per pipeline stage)
│   ├── raggo-rss-podcast/        # RSS feed ingestion
│   ├── raggo-download-audio/     # Audio file download
│   ├── raggo-stt-audio/          # Speech-to-text transcription
│   ├── raggo-normalize-podcast/  # Transcript normalization
│   ├── raggo-chunk-podcast/      # Time-based chunking
│   ├── raggo-embed-podcast/      # Embedding generation
│   └── raggo-index-podcast/      # Qdrant indexing
│
├── pkg/                          # Shared libraries
│   ├── schema/                   # Data schemas
│   ├── storage/                  # File I/O utilities
│   ├── hashing/                  # Content hashing
│   ├── concurrency/              # Worker pool
│   └── logging/                  # Structured logging
│
├── pipelines/                    # Pipeline-specific orchestration
│   └── podcast/
│       └── Makefile              # Podcast pipeline DAG
│
├── data/                         # Pipeline artifacts (organized by type)
│   └── podcast/
│       ├── raw/                  # Raw RSS feed XML
│       ├── manifests/            # Episode manifests (JSONL)
│       ├── audio/                # Downloaded audio files
│       ├── transcripts/          # Raw and normalized transcripts
│       ├── chunks/               # Text chunks (JSONL)
│       ├── embeddings/           # Vector embeddings (JSONL)
│       └── index/                # Index completion markers
│
├── .devcontainer/                # VS Code devcontainer configuration
│   ├── devcontainer.json         # Container settings
│   ├── docker-compose.yml        # Multi-container setup
│   └── Dockerfile                # Custom container image
│
├── Makefile                      # Top-level delegator
├── go.mod                        # Go dependencies
└── README.md                     # This file
```

## Pipeline Stages

### 1. RSS Ingestion (`raggo-rss-podcast`)

**Input**: RSS feed URL  
**Output**: 
- `data/raw/<feed_hash>.xml` - Raw feed XML
- `data/raw/<feed_hash>.json` - Feed metadata
- `data/manifests/<feed_hash>.jsonl` - Episode manifest

Parses podcast RSS feed, extracts episode metadata, and generates stable episode IDs.

### 2. Audio Download (`raggo-download-audio`)

**Input**: Episode manifest  
**Output**: 
- `data/audio/<episode_id>.mp3` - Audio file
- `data/audio/<episode_id>.json` - Audio metadata

Downloads audio files with content hashing and parallel workers.

### 3. Speech-to-Text (`raggo-stt-audio`)

**Input**: Audio file  
**Output**: `data/transcripts/<episode_id>.json` - Raw transcript with timestamps

Transcribes audio via external STT service (HTTP API or CLI). Generates placeholder if no service configured.

### 4. Normalization (`raggo-normalize-podcast`)

**Input**: Raw transcript  
**Output**: `data/transcripts/<episode_id>.normalized.json` - Normalized transcript

Normalizes punctuation, whitespace, and speaker labels while preserving timestamps.

### 5. Chunking (`raggo-chunk-podcast`)

**Input**: Normalized transcript  
**Output**: `data/chunks/<episode_id>.jsonl` - Chunk records

Creates time-windowed chunks with configurable overlap (default: 60s windows, 5s overlap).

### 6. Embedding (`raggo-embed-podcast`)

**Input**: Chunks  
**Output**: `data/embeddings/<episode_id>.jsonl` - Embedding vectors

Generates embeddings via external embedding service. Falls back to placeholders if unconfigured.

### 7. Indexing (`raggo-index-podcast`)

**Input**: Chunks + Embeddings  
**Output**: `data/index/<episode_id>.done` - Completion marker

Upserts vectors into Qdrant with rich payload metadata (corpus, timestamps, source info).

## Quick Start

### Development Environment Options

#### Option 1: Devcontainer (Recommended)

Open in VS Code with the Dev Containers extension:

```bash
# Open in VS Code
code /home/jeffpowell/dev/jeffrpowell/raggo

# When prompted, click "Reopen in Container"
# Or use Command Palette: "Dev Containers: Reopen in Container"
```

The devcontainer automatically:
- Installs Go 1.21 and all tools
- Starts Qdrant (ports 6333/6334)
- Runs `make install-deps && make build`
- Configures Go LSP and linting

#### Option 2: Local Development

**Prerequisites:**
- Go 1.21+
- GNU Make
- jq (for JSON processing)
- Qdrant instance (for indexing)
- Optional: STT service (e.g., Whisper API)
- Optional: Embedding service (e.g., OpenAI embeddings)

**Installation:**

```bash
# Clone repository
git clone https://github.com/jeffrpowell/raggo.git
cd raggo

# Install dependencies
make install-deps

# Build all binaries
make build

# Start Qdrant
docker-compose up -d
```

### Run Full Pipeline

```bash
# Run complete podcast pipeline
make podcast FEED_URL="https://example.com/podcast.rss"
```

### Run Individual Stages

Each pipeline has its own Makefile under `pipelines/<type>/`:

```bash
# Enter pipeline directory
cd pipelines/podcast

# View pipeline-specific help
make help

# Run individual stages
make rss FEED_URL="https://example.com/podcast.rss"
make audio FEED_URL="https://example.com/podcast.rss"
make transcripts FEED_URL="https://example.com/podcast.rss" STT_ENDPOINT="http://localhost:8000/stt"
make normalize FEED_URL="https://example.com/podcast.rss"
make chunks FEED_URL="https://example.com/podcast.rss"
make embeddings FEED_URL="https://example.com/podcast.rss" EMBED_ENDPOINT="http://localhost:8001/embed"
make index FEED_URL="https://example.com/podcast.rss"

# Or run from root
make -C pipelines/podcast all FEED_URL="..."
```

## Configuration

### Environment Variables / Make Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FEED_URL` | (required) | RSS feed URL |
| `STT_ENDPOINT` | (none) | Speech-to-text HTTP endpoint |
| `EMBED_ENDPOINT` | (none) | Embedding service HTTP endpoint |
| `QDRANT_HOST` | `localhost` | Qdrant server host |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `COLLECTION` | `raggo` | Qdrant collection name |
| `CORPUS` | `default` | Corpus identifier for payloads |
| `WORKERS` | `4` | Concurrent workers for downloads |

### External Service APIs

#### STT Service (Expected HTTP API)

**Request**:
```json
POST /stt
{
  "audio_path": "/path/to/audio.mp3"
}
```

**Response**:
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

#### Embedding Service (Expected HTTP API)

**Request**:
```json
POST /embed
{
  "text": "Text to embed",
  "model": "text-embedding-3-small"
}
```

**Response**:
```json
{
  "embedding": [0.123, -0.456, ...],
  "model": "text-embedding-3-small",
  "version": "v1"
}
```

## Operating the Pipeline

### Re-running Stages

To re-run a specific stage for a single episode:

```bash
# Delete the artifact
rm data/chunks/<episode_id>.jsonl

# Re-run make
make chunks FEED_URL="..."
```

### Parallel Processing

Make supports parallel execution:

```bash
make -j 4 transcripts FEED_URL="..."
```

### Inspecting Artifacts

All artifacts are JSON or JSONL:

```bash
# View manifest
cat data/manifests/<feed_hash>.jsonl | jq

# View chunks
cat data/chunks/<episode_id>.jsonl | jq -c

# Count embeddings
wc -l data/embeddings/<episode_id>.jsonl
```

### Cleaning Up

```bash
# Remove all data artifacts (preserves binaries)
make clean

# Remove everything (including binaries)
make clean-all
```

## Data Schemas

### Episode Manifest

```json
{
  "episode_id": "abc123...",
  "feed_url": "https://...",
  "title": "Episode Title",
  "description": "Episode description",
  "audio_url": "https://.../audio.mp3",
  "published_at": "2024-01-01T12:00:00Z",
  "duration": 3600,
  "guid": "episode-guid",
  "metadata_hash": "def456..."
}
```

### Raw Transcript

```json
{
  "episode_id": "abc123...",
  "audio_hash": "xyz789...",
  "model": "whisper-1",
  "model_version": "v1",
  "segments": [
    {
      "start": 0.0,
      "end": 10.5,
      "text": "Transcript text",
      "speaker": "Speaker 1"
    }
  ],
  "generated_at": "2024-01-01T12:00:00Z",
  "transcript_hash": "ghi012..."
}
```

### Chunk

```json
{
  "chunk_id": "jkl345...",
  "episode_id": "abc123...",
  "chunk_index": 0,
  "start_time": 0.0,
  "end_time": 60.0,
  "text": "Combined chunk text...",
  "text_hash": "mno678...",
  "overlap_start": 0.0,
  "overlap_end": 5.0,
  "created_at": "2024-01-01T12:00:00Z"
}
```

### Embedding

```json
{
  "chunk_id": "jkl345...",
  "episode_id": "abc123...",
  "chunk_hash": "mno678...",
  "model": "text-embedding-3-small",
  "model_version": "v1",
  "vector": [0.123, -0.456, ...],
  "dimension": 1536,
  "generated_at": "2024-01-01T12:00:00Z"
}
```

## Design Philosophy

### Why Make?

1. **Declarative DAG**: Dependencies are explicit in the Makefile
2. **Timestamp-based**: Make automatically detects stale artifacts
3. **Parallel execution**: Built-in `-j` flag for parallelism
4. **Universal**: Available on all Unix systems
5. **Inspectable**: Pipeline logic is readable and version-controlled

### Why Separate Binaries?

1. **Composability**: Each stage can be tested independently
2. **Replaceability**: Swap out STT engine without touching other stages
3. **Debuggability**: Run a single stage with custom flags
4. **Resource isolation**: Different stages have different resource needs

### Why File-Based?

1. **Inspectability**: `cat`, `jq`, `grep` work on all artifacts
2. **Versioning**: Git can track changes to artifacts
3. **Disaster recovery**: No hidden database state to corrupt
4. **Reproducibility**: Identical inputs → identical outputs

## Extension Points

### Adding a New Source Type

To add support for a new content type (e.g., YouTube videos):

1. Create new CLI binaries in `cmd/`:
   - `raggo-youtube-download/`
   - `raggo-normalize-youtube/`
   - `raggo-chunk-youtube/`
   
2. Reuse existing shared binaries:
   - `raggo-stt-audio` (if video has audio)
   - `raggo-embed-podcast` → `raggo-embed-text` (generic)
   - `raggo-index-podcast` → `raggo-index` (generic)

3. Add new Makefile targets following the same pattern

4. Define schemas in `pkg/schema/youtube.go`

### Replacing External Services

To swap out the STT engine:

1. Implement a new HTTP wrapper for your STT service
2. Point `STT_ENDPOINT` to your new service
3. Ensure it matches the expected request/response format
4. Alternatively: modify `raggo-stt-audio` to call a different API

## Troubleshooting

### "Episode already indexed" but I want to re-index

```bash
rm data/index/<episode_id>.done
make index FEED_URL="..."
```

### Transcripts are empty

Check that `STT_ENDPOINT` is set and the service is running. Without it, placeholder transcripts are generated.

### Qdrant connection refused

Ensure Qdrant is running and accessible:

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Build fails with missing dependencies

```bash
make install-deps
go mod tidy
```

## License

MIT

## Contributing

This is an operational system optimized for clarity and long-term maintainability. Contributions should prioritize:

1. Inspectability over abstraction
2. Files over in-memory state
3. Explicitness over magic
4. Determinism over convenience
