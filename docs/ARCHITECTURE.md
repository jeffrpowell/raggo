# Raggo Architecture

## Overview

Raggo is a containerized, artifact-first RAG ingestion pipeline built from composable CLI binaries orchestrated by a multi-mode orchestrator. This document is designed to help LLM agents and developers quickly understand the project's philosophy, structure, and design decisions.

## Core Philosophy

### Why This Exists

Most RAG ingestion pipelines are black boxes - hidden state, implicit dependencies, difficult to debug. Raggo takes the opposite approach: everything is explicit, inspectable, and reproducible.

### Key Principles

1. **Artifact-first**: Every pipeline stage consumes files and produces files. No internal state, no hidden databases.
2. **Stage isolation**: Each CLI binary is self-contained and stateless. Swap implementations without breaking the pipeline.
3. **Orchestrator coordination**: Pipeline stages execute sequentially with automatic retry, state tracking, and multiple operating modes.
4. **Inspectability**: All intermediate outputs are human-readable JSON/JSONL. Use `cat`, `jq`, `grep`.
5. **Re-runnability**: Stages check for existing outputs and skip if present. Idempotent by design.

### Design Philosophy: Why These Choices?

**Why Orchestrator?**
- Multi-mode support: One-shot, scheduled, and watch modes in one system
- State management: Track runs and watch state in Qdrant
- Retry logic: Automatic retries with configurable limits
- Configuration-driven: Single YAML file for all settings
- Containerized: Easy deployment with Docker

**Why Separate Binaries?**
- Composability: Test each stage independently
- Replaceability: Swap out STT engine without touching other stages
- Debuggability: Run a single stage with custom flags
- Resource isolation: Different stages have different resource needs

**Why File-Based?**
- Inspectability: Standard Unix tools work on all artifacts
- Versioning: Git can track changes to artifacts
- Disaster recovery: No hidden database state to corrupt
- Reproducibility: Identical inputs → identical outputs

**Why Go?**
- Strong typing: Schemas catch errors at compile time
- Explicit errors: No silent failures
- Bounded concurrency: Worker pools prevent resource exhaustion
- Fast iteration: Compiled binaries with no runtime dependencies

**What This System Does NOT Do:**
- ❌ Real-time processing (batch-oriented by design)
- ❌ Auto-scaling (use external orchestration if needed)
- ❌ Native ML inference (use external APIs instead)
- ❌ Dynamic pipeline configuration (static stages preferred)
- ❌ Built-in web UI (CLI-first philosophy)

## Repository Structure

```
raggo/
├── cmd/                          # CLI binaries (one per pipeline stage)
│   ├── raggo-orchestrator/       # Multi-mode orchestrator
│   ├── raggo-rss-podcast/        # RSS feed ingestion
│   ├── raggo-download-audio/     # Audio download
│   ├── raggo-stt-audio/          # Speech-to-text transcription
│   ├── raggo-normalize-podcast/  # Podcast transcript normalization
│   ├── raggo-chunk-podcast/      # Podcast time-based chunking
│   ├── raggo-embed-podcast/      # Podcast embedding generation
│   ├── raggo-index-podcast/      # Podcast Qdrant indexing
│   ├── raggo-scan-documents/     # Document discovery/scanning
│   ├── raggo-extract-text/       # Text extraction from documents
│   ├── raggo-chunk-text/         # Generic text chunking
│   ├── raggo-embed-text/         # Generic text embedding
│   └── raggo-index-documents/    # Document Qdrant indexing
│
├── pkg/                          # Shared libraries
│   ├── config/                   # Configuration loading and validation
│   ├── state/                    # Qdrant-based state management
│   ├── pipeline/                 # Pipeline definitions and executor
│   ├── schema/                   # Data schemas (podcast.go, document.go)
│   ├── storage/                  # File I/O utilities
│   ├── hashing/                  # Content hashing
│   ├── concurrency/              # Worker pool
│   ├── logging/                  # Structured logging
│   ├── tika/                     # Apache Tika client for text extraction
│   └── vision/                   # Vision API client for image processing
│
├── config/                       # Configuration files
│   └── raggo.example.yml         # Example configuration
│
├── data/                         # Pipeline artifacts (gitignored)
│   ├── podcast/
│   │   ├── raw/                  # Raw RSS feed XML
│   │   ├── manifests/            # Episode manifests (JSONL)
│   │   ├── audio/                # Downloaded audio files
│   │   ├── transcripts/          # Raw and normalized transcripts
│   │   ├── chunks/               # Text chunks (JSONL)
│   │   ├── embeddings/           # Vector embeddings (JSONL)
│   │   └── index/                # Index completion markers
│   │
│   └── documents/
│       ├── sources/              # Source document files
│       ├── extracted/            # Extracted text
│       ├── chunks/               # Text chunks (JSONL)
│       ├── embeddings/           # Vector embeddings (JSONL)
│       └── index/                # Index completion markers
│
├── scripts/                      # Helper utilities (optional)
│   ├── clean-episode.sh          # Clean episode artifacts
│   ├── list-episodes.sh          # List all episodes
│   └── inspect-episode.sh        # Debug specific episode
│
├── .devcontainer/                # VS Code devcontainer config
└── Dockerfile                    # Multi-stage build for all binaries
```

### How to Navigate the Codebase

**Looking for business logic?** Check `cmd/raggo-*-*/main.go` files. Each binary is self-contained.

**Looking for orchestration logic?** Check `cmd/raggo-orchestrator/` and `pkg/pipeline/`.

**Looking for data schemas?** Check `pkg/schema/`. All JSON structures are defined here.

**Looking for configuration?** Check `config/raggo.example.yml` for all options.

**Looking for example usage?** Check `docs/EXAMPLES.md` for comprehensive examples.

**Looking for development setup?** Check `CONTRIBUTING.md` for devcontainer and local setup.

## Pipeline Data Flow

The podcast pipeline follows a linear sequence of transformations:

1. **RSS Ingestion** → Produces: `raw/*.xml`, `manifests/*.jsonl`
2. **Audio Download** → Produces: `audio/*.mp3`
3. **Speech-to-Text** (external service) → Produces: `transcripts/*.json`
4. **Normalization** → Produces: `transcripts/*.normalized.json`
5. **Chunking** → Produces: `chunks/*.jsonl`
6. **Embedding** (external service) → Produces: `embeddings/*.jsonl`
7. **Indexing** → Produces: `index/*.done` markers, upserts to Qdrant

Each stage only knows about its input and output file locations. Dependencies are managed by the orchestrator, which executes stages sequentially with automatic retry on failure.

## Naming Conventions

**Binary Names:** `raggo-<action>-<source-type>`

Examples: `raggo-rss-podcast`, `raggo-download-audio`, `raggo-stt-audio`, `raggo-normalize-podcast`

**Artifact Paths:** `data/<pipeline>/<stage>/<identifier>.<extension>`

Examples: 
- `data/podcast/manifests/<feed_hash>.jsonl` - Episode lists
- `data/podcast/audio/<episode_id>.mp3` - Downloaded audio
- `data/podcast/chunks/<episode_id>.jsonl` - Text chunks
- `data/podcast/index/qdrant.done` - Terminal pipeline completion marker

## Technical Implementation Details

### Concurrency Model

The system uses multiple levels of parallelism:

1. **Corpus-level**: Orchestrator runs multiple corpora concurrently (one-shot, scheduled, watch)
2. **Within-stage**: Go worker pools handle parallel operations (downloads, API calls) with bounded concurrency
3. **Sequential stages**: Within a corpus, stages execute sequentially with idempotency checks

This prevents resource exhaustion while maximizing throughput across multiple data sources.

### Error Handling Philosophy

**Fail-fast with retry**: Stages fail loudly on errors. No silent corruption. The orchestrator automatically retries failed pipelines with configurable limits.

**Idempotency**: Re-running a stage is always safe. Existing outputs are detected and skipped.

**Atomic writes**: Temporary files are used, then atomically renamed to prevent partial writes.

**State tracking**: All runs are tracked in Qdrant metadata collection for observability and debugging.

## Extensibility: Adding New Source Types

The system is designed to be extended. To add support for a new content type (e.g., YouTube videos):

1. Create source-specific binaries in `cmd/` (e.g., `raggo-youtube-download/`, `raggo-extract-audio/`)
2. Reuse generic binaries where possible (`raggo-stt-audio`, `raggo-embed-podcast`)
3. Add a new pipeline builder in `pkg/pipeline/youtube.go` with stage definitions
4. Define new schemas in `pkg/schema/youtube.go`
5. Update orchestrator to recognize the new corpus type
6. Add corpus configuration in `raggo.yml`

The orchestrator will automatically handle execution, retries, and state tracking.

## Extensibility: Replacing External Services

Want to use a different STT or embedding service?

**Option 1**: Update service endpoints in `raggo.yml`:
```yaml
services:
  stt_endpoint: http://your-stt-service:8000/transcribe
  embed_endpoint: http://your-embed-service:8001/embed
```

**Option 2**: Wrap your service in an HTTP API matching the expected request/response format (see `docs/EXAMPLES.md` for API specs)

**Option 3**: Fork and modify the relevant binary (`cmd/raggo-stt-audio/`, `cmd/raggo-embed-podcast/`)

The orchestrator passes configuration to all binaries automatically.

## Testing Strategy

The codebase supports three levels of testing:

1. **Unit tests**: Test shared libraries (`pkg/`) in isolation
2. **Integration tests**: Test individual CLI binaries with fixtures
3. **End-to-end tests**: Test full pipeline with small test feeds

See `CONTRIBUTING.md` for testing commands and practices.

## Operational Considerations

### Monitoring

Monitor pipeline progress through:
- **Container logs**: `docker compose logs -f raggo`
- **Qdrant metadata collection**: Query run state and history
- **Artifact counting**: All intermediate outputs are files, so standard Unix tools work

### Debugging

Inspect artifacts with `cat`, `jq`, and `grep`. Check specific stage outputs to identify failures. Query Qdrant metadata collection for run history and errors. See `docs/EXAMPLES.md` for debugging workflows.

### Performance

**Bottlenecks:**
- Audio download: Network I/O (increase workers in corpus config)
- STT/Embedding: External API rate limits (batch requests or use local inference)
- Indexing: Qdrant network latency (batch upserts)

**Optimization:** Configure worker counts per corpus, or modify binaries for batched API calls. Multiple corpora run concurrently automatically.

### Security

**API Keys**: Store in environment variables or config files, never hardcode.

**Data Privacy**: All artifacts stored locally in gitignored `data/` directory. Encrypt at rest for sensitive content.

## Orchestrator Architecture

The containerized orchestrator extends the Make-based approach with automated multi-corpus management:

### Multi-Binary Container Approach

- **Each pipeline stage** is a separate binary (`raggo-rss-podcast`, `raggo-download-audio`, etc.)
- **Orchestrator binary** (`raggo-orchestrator`) coordinates execution across stages
- **Single Docker image** packages all 13 binaries together
- **State tracking** in Qdrant metadata collection for run history and watch state

### State Management

The orchestrator uses Qdrant's metadata collection to track:

**Run Metadata:**
- Run ID, corpus ID, start/completion times
- Current status (running, success, failed)
- Current stage being executed
- Items found, processed, and indexed
- Error messages if failed

**Watch State:**
- Last check timestamp
- Content hash of RSS feed or directory tree
- Individual item hashes for change detection

This enables resumable pipelines and change-based triggering.

### Retry Logic

Failed pipelines automatically retry with:
- Configurable retry limit (default: 3 attempts)
- Configurable delay between retries (default: 60 seconds)
- Immediate retry strategy (no exponential backoff)
- Each retry creates a new run ID for tracking

### Configuration-Driven Execution

All configuration is managed through a single `raggo.yml` file:
- Storage paths with variable expansion (`${root}/podcast/audio`)
- Qdrant connection settings
- External service endpoints (STT, embeddings)
- Corpus definitions with modes (one-shot, scheduled, watch)
- Retry and logging configuration

### Operating Modes

1. **One-shot**: Execute pipeline once per corpus and exit
2. **Scheduled**: Run on cron schedule (e.g., `"0 2 * * *"` for daily at 2 AM)
3. **Watch**: Monitor RSS feeds or directories for changes, trigger automatically

Multiple corpora can run with different modes simultaneously.

## Future Enhancement Opportunities

Potential additions that maintain design principles:
- Hash-based deduplication
- Incremental feed updates
- Speaker identification
- Quality metrics tracking
- Pipeline progress visualization
- Webhook notifications on completion/failure

Remember: The system prioritizes inspectability and determinism over convenience and automation.
