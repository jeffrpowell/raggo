---
name: raggo-search
description: Search indexed documents and podcasts with reranking.
version: 0.1.0
---

# raggo Search

Answers questions from the user's raggo knowledge base: documents and podcast transcripts that raggo has indexed into Qdrant. Each query runs hybrid retrieval (dense embeddings plus BM25 over LLM-generated chunk context), fuses the results with RRF, and reranks them. The skill only reads the index. It does not ingest or modify anything. It assumes `raggo_search.py` is installed on this machine and can reach Qdrant, the embedding model and the reranker.

## When to Use

- "What do my docs say about …", "find where we decided …", "search my notes/podcasts for …"
- Any question whose answer is likely in the user's own documents or podcast episodes rather than on the public web
- Follow-ups that need more of a document the user has already been shown

## Prerequisites

- `python3` with `httpx` and `pydantic` (`pip install httpx pydantic`)
- The raggo repo checked out; commands below assume `$RAGGO_EXAMPLES` points at its `examples/` directory
- Environment variables (override only what differs from the defaults):
  - `RAGGO_QDRANT_URL` (default `http://qdrant:6333`)
  - `RAGGO_COLLECTIONS`, comma-separated (default `company-docs`)
  - `RAGGO_EMBED_ENDPOINT`, `RAGGO_EMBED_MODEL`, `RAGGO_EMBED_API` (`openai` or `raggo`)
  - `RAGGO_RERANKER_ENDPOINT`, `RAGGO_RERANKER_MODEL` (set the endpoint to an empty string to skip reranking)

## How to Run

Run every command through `run_command`. Searching is iterative. Treat the first query as a probe, then refine it.

1. Break the user's question into one to three focused sub-queries.
2. Search each one. Read both the `context` line (what the chunk is about within its source) and the chunk text.
3. If a hit looks relevant but cuts off mid-thought, fetch its neighbors.
4. If the results are weak, rewrite the query using terms from the `context` lines of the best hits (names, product terms, dates) and search again. Stop after three rounds with nothing new.
5. Answer only from the retrieved text and cite each claim with its `source` label.

## Quick Reference

- `python3 $RAGGO_EXAMPLES/open-webui-computer/raggo_search.py search "<query>"`: top reranked passages
- `... search "<query>" --collection <name>`: search a single collection
- `... search "<query>" --top-k 10 --json`: more results, machine-readable
- `... neighbors <collection> <source_id> <chunk_index> --radius 2`: surrounding chunks, in order

Each result prints `source`, `score`, `collection`, `source_id`, `chunk_index`, `context` and the chunk text. Podcast sources look like `podcast <episode_id> @ h:mm:ss-h:mm:ss`.

## Procedure

1. `python3 $RAGGO_EXAMPLES/open-webui-computer/raggo_search.py search "how are signing keys rotated"`
2. Pick the best hit and note its `collection`, `source_id` and `chunk_index`.
3. `python3 $RAGGO_EXAMPLES/open-webui-computer/raggo_search.py neighbors company-docs <source_id> <chunk_index> --radius 2`
4. If needed, search again with refined terms, e.g. `search "KMS key rotation schedule quarterly"`.
5. Write the answer and cite sources as `[source]`.

## Pitfalls

- `error: All connection attempts failed` means a service is down or the URL is wrong. Check `RAGGO_QDRANT_URL`, `RAGGO_EMBED_ENDPOINT` and `RAGGO_RERANKER_ENDPOINT`.
- `Not found: Vector with name dense` means the collection was indexed before raggo added hybrid vectors. It has to be re-indexed.
- The embedding model has to match the one raggo indexed with, or the dense results will be noise.
- Scores are reranker relevance scores, not probabilities. Compare them within one result list only.
- `neighbors` needs `chunk_index` in the payload. Collections indexed by older raggo versions don't have it.

## Verification

`python3 $RAGGO_EXAMPLES/open-webui-computer/raggo_search.py search "test" --top-k 1 --json` exits 0 and prints a JSON array.
