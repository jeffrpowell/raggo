# Raggo Architecture

## Overview

Raggo is a deterministic, artifact-first RAG ingestion pipeline built from composable CLI tools orchestrated by GNU Make. This document is designed to help LLM agents and developers quickly understand the project's philosophy, structure, and design decisions.

## Core Philosophy

### Why This Exists

Most RAG ingestion pipelines are black boxes - hidden state, implicit dependencies, difficult to debug. Raggo takes the opposite approach: everything is explicit, inspectable, and reproducible.

### Key Principles

1. **Artifact-first**: Every pipeline stage consumes files and produces files. No internal state, no hidden databases.
2. **Stage isolation**: Each CLI binary is self-contained and stateless. Swap implementations without breaking the pipeline.
3. **Makefile orchestration**: GNU Make defines the DAG explicitly. Timestamps handle caching automatically.
4. **Inspectability**: All intermediate outputs are human-readable JSON/JSONL. Use `cat`, `jq`, `grep`.
5. **Re-runnability**: Delete an artifact and re-run `make` to regenerate it deterministically.

### Design Philosophy: Why These Choices?

**Why Make?**
- Declarative DAG: Dependencies are explicit in the Makefile
- Timestamp-based: Automatically detects stale artifacts
- Parallel execution: Built-in `-j` flag
- Universal: Available on all Unix systems
- Inspectable: Pipeline logic is readable and version-controlled

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
│   ├── schema/                   # Data schemas (podcast.go, document.go)
│   ├── storage/                  # File I/O utilities
│   ├── hashing/                  # Content hashing
│   ├── concurrency/              # Worker pool
│   ├── logging/                  # Structured logging
│   ├── tika/                     # Apache Tika client for text extraction
│   └── vision/                   # Vision API client for image processing
│
├── pipelines/                    # Pipeline-specific orchestration
│   ├── podcast/Makefile          # Podcast pipeline DAG
│   └── documents/Makefile        # Documents pipeline DAG
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
├── scripts/                      # Helper utilities
│   ├── verify-pipeline.sh        # Check pipeline status
│   ├── list-episodes.sh          # List all episodes
│   └── inspect-episode.sh        # Debug specific episode
│
├── .devcontainer/                # VS Code devcontainer config
└── Makefile                      # Top-level delegator
```

### How to Navigate the Codebase

**Looking for business logic?** Check `cmd/raggo-*-*/main.go` files. Each binary is self-contained.

**Looking for data schemas?** Check `pkg/schema/`. All JSON structures are defined here.

**Looking for pipeline DAG?** Check `pipelines/*/Makefile`. Dependencies are explicit.

**Looking for example usage?** Check `EXAMPLES.md` for comprehensive examples.

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

Each stage only knows about its input and output file locations. Dependencies are managed by Make.

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

The system uses two levels of parallelism:

1. **Episode-level**: Make's `-j` flag processes multiple episodes concurrently
2. **Within-stage**: Go worker pools handle parallel operations (downloads, API calls) with bounded concurrency

This prevents resource exhaustion while maximizing throughput.

### Error Handling Philosophy

**Fail-fast**: Stages fail loudly on errors. No silent corruption. Make stops immediately on failure.

**Idempotency**: Re-running a stage is always safe. Existing outputs are detected and skipped.

**Atomic writes**: Temporary files are used, then atomically renamed to prevent partial writes

## Extensibility: Adding New Source Types

The system is designed to be extended. To add support for a new content type (e.g., YouTube videos):

1. Create source-specific binaries in `cmd/` (e.g., `raggo-youtube-download/`, `raggo-extract-audio/`)
2. Reuse generic binaries where possible (`raggo-stt-audio`, `raggo-embed-podcast`)
3. Create a new pipeline Makefile in `pipelines/youtube/Makefile`
4. Define new schemas in `pkg/schema/youtube.go`
5. Add a top-level delegator target in root `Makefile`

See `EXAMPLES.md` for detailed extension examples.

## Extensibility: Replacing External Services

Want to use a different STT or embedding service?

**Option 1**: Wrap your service in an HTTP API matching the expected request/response format (see `EXAMPLES.md` for API specs)

**Option 2**: Fork and modify the relevant binary (`cmd/raggo-stt-audio/`, `cmd/raggo-embed-podcast/`)

The system is designed for easy swapping of external dependencies.

## Testing Strategy

The codebase supports three levels of testing:

1. **Unit tests**: Test shared libraries (`pkg/`) in isolation
2. **Integration tests**: Test individual CLI binaries with fixtures
3. **End-to-end tests**: Test full pipeline with small test feeds

See `CONTRIBUTING.md` for testing commands and practices.

## Operational Considerations

### Monitoring

Monitor pipeline progress by counting artifacts at each stage. All intermediate outputs are files, so standard Unix tools work.

### Debugging

Inspect artifacts with `cat`, `jq`, and `grep`. Check specific stage outputs to identify failures. See `EXAMPLES.md` for debugging workflows.

### Performance

**Bottlenecks:**
- Audio download: Network I/O (increase workers)
- STT/Embedding: External API rate limits (batch requests or use local inference)
- Indexing: Qdrant network latency (batch upserts)

**Optimization:** Use Make's `-j` flag for parallel episode processing, or modify binaries for batched API calls.

### Security

**API Keys**: Store in environment variables or config files, never hardcode.

**Data Privacy**: All artifacts stored locally in gitignored `data/` directory. Encrypt at rest for sensitive content.

## Future Enhancement Opportunities

Potential additions that maintain design principles:
- Hash-based deduplication
- Incremental feed updates
- Speaker identification
- Quality metrics tracking
- Pipeline progress visualization

Remember: The system prioritizes inspectability and determinism over convenience and automation.
