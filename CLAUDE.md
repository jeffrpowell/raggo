# raggo

A containerized RAG ingestion pipeline written in Go. Podcast RSS feeds and document directories go through a chain of single-purpose CLI binaries (`cmd/raggo-*`) that end in Qdrant. `raggo-orchestrator` runs those binaries as pipeline stages (`pkg/pipeline/`). Querying is not part of raggo. The query-side examples live in `examples/`. Design docs: `docs/ARCHITECTURE.md`.

## Build & test

- `go vet ./... && go test ./...`, but `pkg/vision` and `cmd/raggo-extract-text` need CGO and a C compiler (go-fitz). Without gcc, leave them out: `go test $(go list ./... | grep -v -e pkg/vision -e raggo-extract-text)`.
- The Dockerfile builds with `CGO_ENABLED=0`, which breaks `raggo-extract-text`. This is a known, unresolved problem.
- Python examples: `pip install httpx pydantic pytest && cd examples/open-webui && pytest`.
- The devcontainer image (`.devcontainer/`) has Go 1.25. `go.mod` requires 1.25.6.
- Several older files (`pkg/schema/*.go`, `pkg/config/types.go`, `cmd/raggo-embed-*`) aren't gofmt-clean. Don't reformat them as a side effect. Keep diffs to the lines you actually change.

## Stage conventions

- Artifact-first: each stage reads files and writes files under `storage.*` dirs (`<stage>/<id>.jsonl`). A stage skips work when its output exists (`storage.MarkerExists`), so deleting an artifact is how you force a rerun.
- Flags win over config. Every dir and endpoint goes through a `config.Resolve*` helper in `pkg/config/paths.go` (flag → config → hardcoded default). New settings follow the same pattern.
- The orchestrator runs stages in order for the **whole corpus**: stage N finishes for every item before stage N+1 starts. Binaries that require `-document-id` / `-episode-id` get it through `Stage.Items` + `ItemFlag`. The executor runs them once per `Item`, appending `ItemFlag <id>` plus `Item.Args`.
- Item sources live in `pkg/pipeline/items.go`. Each one lists the *upstream* stage's outputs and is evaluated when the stage starts:
  - extract: scan manifests `<text>/*.jsonl`, which also supply `-file-path`
  - chunk-text: `<text>/<id>.json`
  - stt: audio metadata `<audio>/<id>.json`, which also supplies `-audio-path`. Feed metadata in the same dir is skipped.
  - normalize: `<transcripts>/<id>.json`
  - chunk-podcast: `<transcripts>/<id>.normalized.json`
  - embed and index: `<chunks>/<id>.jsonl`
- When you add an ID-requiring binary, give its stage an item source. The binaries skip finished items themselves, so listing every item is safe.
- Contextualize stages don't need an ID. Without one they process every chunk file that has no context file.
- Known bug: the `audio` stage builds its `-manifest` path with `pipeline.hashString` (hex of the URL's first 16 bytes), while `raggo-rss-podcast` names the manifest `hashing.HashString(feedURL)`. The names don't match, so downloads can't find the manifest.
- Per-corpus options are read from `corpus.Config` (a `map[string]interface{}`) with type assertions (`chunk_size`, `contextualize`, `context_budget_chars`).

## Contextual retrieval (`pkg/contextual`, `cmd/raggo-contextualize-*`)

- This implements Anthropic's Contextual Retrieval for small local models: full-document mode when the text fits `context_budget_chars`, otherwise a cached synopsis plus a local excerpt ("windowed").
- **Prompt layout is load-bearing.** The chunk must stay *last*, and chunks in a group must share a byte-identical prefix so llama-server's `cache_prompt` reuses its KV cache. Tests assert this. If you change prompt text, bump `PromptVersion`, which also invalidates cached synopses.
- `contexts/<id>.jsonl` is written to `.tmp` and renamed when complete. A partial file would count as done because of the marker-skip rule, so keep the write atomic.
- Embed and index look contexts up by `chunk_id` and require a matching `text_hash`. A stale context is ignored silently, never applied to changed chunk text.
- LLM calls go to an OpenAI-compatible endpoint (llama-server / llama-swap). `chat_template_kwargs.enable_thinking=false` is on by default for Qwen3.x.

## Qdrant layout (`pkg/vectorstore`)

- Each collection has a named `dense` vector (cosine) plus a sparse `bm25` vector with `Modifier_Idf`. **BM25 is computed on the server** (`qdrant/bm25` inference, Qdrant ≥ 1.15.2): the index stage sends text, not token IDs. Don't add a client-side tokenizer, or query and index tokenization will drift apart.
- Collections that don't have this layout are rejected with an error, never migrated. Re-indexing means deleting the collection plus the `embeddings/` and `index/` artifacts.
- The payload contract (`text`, `context`, `chunk_index`, `document_id`/`episode_id`, `source_path`, `start_time`/`end_time`) is used by `examples/open-webui/raggo_search_tool.py`. Update both together.

## Examples (`examples/`)

- `raggo_search_tool.py` is an Open WebUI Workspace Tool and is also imported by the Computer CLI. Keep it a single file with no `open_webui` imports. Every public method is exposed to the model as a tool, so helpers must start with `_`. `__init__` must set `self.valves = self.Valves()`.

## Git

- Commit directly to `main`. This is a solo repo with no Jira keys, and it overrides the global feature-branch rule.
