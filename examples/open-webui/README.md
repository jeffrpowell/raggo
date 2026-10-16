# raggo in Open WebUI chat: hybrid search + reranking

[`raggo_search_tool.py`](raggo_search_tool.py) is an Open WebUI Workspace Tool. It implements the query side of [Contextual Retrieval](https://www.anthropic.com/engineering/contextual-retrieval) against collections raggo has indexed:

1. **Embed** the query with the same model raggo used for chunks, plus a query instruction prefix (Qwen3-Embedding expects one).
2. **Hybrid retrieve** with a single Qdrant Query API call. It prefetches the top `prefetch_k` from the `dense` vector and the top `prefetch_k` from the `bm25` sparse vector, then fuses them with RRF. BM25 is computed on the server by Qdrant (`qdrant/bm25`), so query and index tokenization always match.
3. **Rerank** the fused candidates with a cross-encoder through llama-server's `/v1/rerank`, scoring `context + chunk`. It keeps the top `top_k`.
4. **Cite**: each result is emitted as an Open WebUI citation.

The model sees three tools:

- `search_knowledge(query, collection?)` searches one collection, or every configured collection when `collection` is omitted.
- `list_collections()` returns each configured collection's name, description and chunk count.
- `get_neighbors(collection, source_id, chunk_index, radius)` reads the chunks around a hit.

## Picking a collection

Without `collection`, the tool runs one hybrid query per collection and reranks the pooled candidates together. The reranker is what makes this work. RRF scores only reflect rank within one collection, so every collection's best hit gets the same score. Reranker scores measure relevance to the query and compare across collections. With `reranker_endpoint` empty, results from different collections are just interleaved.

Searching everything works well when the collections cover different topics. It's weaker when they share vocabulary (two rulebooks for games built on the same system, say) and the question doesn't say which one it means. That's when the model should pass `collection`, and it needs to know what each collection holds to do so. Give each corpus a `description` in `raggo.yml`:

```yaml
corpora:
  - id: avatar-legends
    type: documents
    config:
      root_path: /mnt/documents/avatar-legends
      collection: avatar-legends
      corpus: avatar-legends-v1
      description: Rules for the Avatar Legends TTRPG (playbooks, moves, techniques, GM advice)
```

raggo's `describe` stage stores it in the collection's Qdrant metadata at the end of each pipeline run, and `list_collections` reads it back. Editing the description doesn't require re-indexing. Small local models sometimes skip a discovery call, so paste the same list into the system prompt too (step 4 below).

## Why a Tool instead of External Knowledge Sources?

Open WebUI's *External Knowledge Sources* (Admin → Integrations, experimental) can point at a raggo collection. Set **Content Field** to `text`, **Source Field** to `source_path` and **Document ID Field** to `document_id`. It only does a mapped vector lookup, though. It doesn't know about raggo's named `dense` + `bm25` vectors, RRF fusion or your reranker. Use the Tool when you want the full hybrid + rerank pipeline.

## Serving the models (llama-swap)

Run the embedder and reranker together, and add the small context model if it fits, so neither ingestion nor queries force a swap:

```yaml
models:
  embed-qwen3-0.6b:
    cmd: llama-server --port ${PORT} -m Qwen3-Embedding-0.6B-Q8_0.gguf --embedding --pooling last -ub 8192
  qwen3-reranker-0.6b:
    cmd: llama-server --port ${PORT} -m Qwen3-Reranker-0.6B-Q8_0.gguf --reranking
  qwen3.5-4b:
    cmd: llama-server --port ${PORT} -m Qwen3.5-4B-Q4_K_M.gguf -c 8192 --parallel 1
groups:
  rag:
    swap: false
    exclusive: true
    members: [embed-qwen3-0.6b, qwen3-reranker-0.6b, qwen3.5-4b]
```

`--parallel 1` keeps raggo's contextualize calls on one slot, so consecutive chunks of a document reuse the KV cache for the shared document prefix.

## Install

1. **Workspace → Tools → +**, paste `raggo_search_tool.py` and save. Open WebUI already bundles `httpx` and `pydantic`.
2. Click the gear icon on the tool and set the **Valves**:

   | Valve | Example |
   | --- | --- |
   | `qdrant_url` | `http://qdrant:6333` |
   | `collections` | `company-docs,tech-podcast` |
   | `embed_endpoint` / `embed_api` / `embed_model` | `http://llama-swap:8080/v1/embeddings` / `openai` / `embed-qwen3-0.6b` |
   | `reranker_endpoint` / `reranker_model` | `http://llama-swap:8080/v1/rerank` / `qwen3-reranker-0.6b` |
   | `qdrant_api_key` | Sent as Qdrant's `api-key` header. Leave empty if Qdrant has no key |
   | `embed_api_key` / `reranker_api_key` | Sent as `Authorization: Bearer <key>`. Set them if llama-swap has `apiKeys`; leave empty otherwise |
   | `prefetch_k` / `top_k` | `20` / `5` |

   If you use raggo's own embed service instead of llama-server, set `embed_api` to `raggo` and point `embed_endpoint` at it.
3. **Workspace → Models → (your model) → Tools**: enable *raggo Search*. Keep **Function Calling** set to `Native` (the default since v0.10).
4. Add this to the model's system prompt, with your own collections in the list (`raggo_search.py collections` prints them):

   > You can search the user's own documents and podcasts with `search_knowledge`. Search before answering any question their material might cover, even if they don't mention it. The collections are:
   > - `avatar-legends`: rules for the Avatar Legends TTRPG
   > - `company-docs`: internal policies, runbooks and architecture notes
   >
   > Pass `collection` when a question clearly belongs to one of them, and omit it otherwise. Call `list_collections` if you're unsure what a collection holds. Each result has a `context` line that says where the passage sits in its source. If the results are weak, search again with terms from those context lines. Use `get_neighbors` to read around a promising hit. Cite sources by their `source` label and don't answer from memory when the knowledge base has the answer.

## Try it

Ask something only your corpus can answer. The chat shows a status line ("Searching raggo for: …"), a `search_knowledge` tool call, and citation chips under the answer. Expand the tool call to compare `fusion_score` (RRF) with `score` (reranker). The reranker often reorders the fused list, which is the point of step 3.

## Tests

```bash
pip install httpx pydantic pytest
cd examples/open-webui && pytest
```
