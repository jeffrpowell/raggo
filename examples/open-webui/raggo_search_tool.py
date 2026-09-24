"""
title: raggo Search
author: raggo
description: Hybrid (dense + BM25) search with RRF fusion and reranking over raggo's Qdrant collections, following Anthropic's Contextual Retrieval recipe.
requirements: httpx, pydantic
version: 0.1.0
licence: MIT
"""

# This file works both as an Open WebUI Workspace Tool (paste it into
# Workspace -> Tools) and as a plain Python module imported by
# examples/open-webui-computer/raggo_search.py. Keep it free of open_webui
# imports so it runs anywhere.

import json
from typing import Any, Awaitable, Callable, Optional

import httpx
from pydantic import BaseModel, Field

DENSE_VECTOR = "dense"
SPARSE_VECTOR = "bm25"
BM25_MODEL = "qdrant/bm25"


class Tools:
    class Valves(BaseModel):
        qdrant_url: str = Field("http://qdrant:6333", description="Qdrant REST endpoint")
        qdrant_api_key: str = Field("", description="Qdrant API key (optional)")
        collections: str = Field(
            "company-docs", description="Comma-separated raggo collections to search"
        )
        embed_endpoint: str = Field(
            "http://llama-swap:8080/v1/embeddings",
            description="Query embedding endpoint (must use the same model raggo indexed with)",
        )
        embed_api: str = Field(
            "openai",
            description="'openai' for /v1/embeddings, or 'raggo' for raggo's {text, model} -> {embedding} contract",
        )
        embed_model: str = Field("embed-qwen3-0.6b", description="Embedding model name")
        query_instruction: str = Field(
            "Instruct: Given a question, retrieve passages that answer it\nQuery: ",
            description="Prefix for queries only (Qwen3-Embedding expects one; documents are embedded without it)",
        )
        reranker_endpoint: str = Field(
            "http://llama-swap:8080/v1/rerank",
            description="llama-server /v1/rerank endpoint (empty disables reranking)",
        )
        reranker_model: str = Field("qwen3-reranker-0.6b", description="Reranker model name")
        prefetch_k: int = Field(20, description="Candidates fetched per retriever before fusion")
        top_k: int = Field(5, description="Results returned after reranking")
        timeout_seconds: float = Field(60.0, description="HTTP timeout")

    def __init__(self):
        self.valves = self.Valves()
        self._transport: Optional[httpx.AsyncBaseTransport] = None

    async def search_knowledge(
        self,
        query: str,
        collection: Optional[str] = None,
        __event_emitter__: Optional[Callable[[dict], Awaitable[None]]] = None,
    ) -> str:
        """
        Search the raggo knowledge base (documents and podcast transcripts) using hybrid
        keyword + semantic search with reranking. Call this before answering questions
        about the user's documents or podcasts. If results are weak, rephrase the query
        using vocabulary from the returned context passages and search again.
        :param query: A natural-language question or keyword query.
        :param collection: Optional single collection name to search instead of all configured collections.
        :return: JSON list of results with source, score, context, text, collection, source_id and chunk_index.
        """
        await self._status(__event_emitter__, f"Searching raggo for: {query}", False)
        try:
            results = await self._search(query, collection)
        except httpx.HTTPError as e:
            await self._status(__event_emitter__, "raggo search failed", True)
            return json.dumps({"error": f"raggo search failed: {e}"})

        if __event_emitter__:
            for r in results:
                await __event_emitter__(
                    {
                        "type": "citation",
                        "data": {
                            "document": [r["text"]],
                            "metadata": [{"source": r["source"]}],
                            "source": {"name": r["source"]},
                        },
                    }
                )
        await self._status(__event_emitter__, f"Found {len(results)} passages", True)
        return json.dumps(results, ensure_ascii=False)

    async def get_neighbors(
        self,
        collection: str,
        source_id: str,
        chunk_index: int,
        radius: int = 1,
    ) -> str:
        """
        Fetch the chunks immediately before and after a search hit to read more of the
        surrounding document or transcript. Use the collection, source_id and chunk_index
        values from a search_knowledge result.
        :param collection: Collection name from the search result.
        :param source_id: source_id from the search result (document or episode ID).
        :param chunk_index: chunk_index from the search result.
        :param radius: How many chunks to fetch on each side (default 1).
        :return: JSON list of chunks ordered by chunk_index.
        """
        try:
            return json.dumps(
                await self._neighbors(collection, source_id, chunk_index, radius),
                ensure_ascii=False,
            )
        except httpx.HTTPError as e:
            return json.dumps({"error": f"raggo neighbor lookup failed: {e}"})

    # --- helpers (underscore prefix keeps them hidden from the model) ---

    def _client(self) -> httpx.AsyncClient:
        return httpx.AsyncClient(timeout=self.valves.timeout_seconds, transport=self._transport)

    def _qdrant_headers(self) -> dict:
        if self.valves.qdrant_api_key:
            return {"api-key": self.valves.qdrant_api_key}
        return {}

    async def _search(self, query: str, collection: Optional[str] = None) -> list:
        v = self.valves
        names = [collection] if collection else [c.strip() for c in v.collections.split(",") if c.strip()]

        async with self._client() as client:
            vector = await self._embed(client, query)
            candidates = []
            for name in names:
                candidates.extend(await self._hybrid_query(client, name, query, vector))
            if not candidates:
                return []
            ranked = await self._rerank(client, query, candidates)
        return ranked[: v.top_k]

    async def _embed(self, client: httpx.AsyncClient, query: str) -> list:
        v = self.valves
        text = v.query_instruction + query
        if v.embed_api == "raggo":
            resp = await client.post(v.embed_endpoint, json={"text": text, "model": v.embed_model})
            resp.raise_for_status()
            return resp.json()["embedding"]
        resp = await client.post(v.embed_endpoint, json={"input": text, "model": v.embed_model})
        resp.raise_for_status()
        return resp.json()["data"][0]["embedding"]

    async def _hybrid_query(self, client: httpx.AsyncClient, collection: str, query: str, vector: list) -> list:
        k = self.valves.prefetch_k
        body = {
            "prefetch": [
                {"query": vector, "using": DENSE_VECTOR, "limit": k},
                {"query": {"text": query, "model": BM25_MODEL}, "using": SPARSE_VECTOR, "limit": k},
            ],
            "query": {"fusion": "rrf"},
            "limit": k,
            "with_payload": True,
        }
        resp = await client.post(
            f"{self.valves.qdrant_url}/collections/{collection}/points/query",
            json=body,
            headers=self._qdrant_headers(),
        )
        resp.raise_for_status()
        return [_to_result(collection, p) for p in resp.json()["result"]["points"]]

    async def _rerank(self, client: httpx.AsyncClient, query: str, candidates: list) -> list:
        v = self.valves
        if not v.reranker_endpoint:
            return sorted(candidates, key=lambda r: r["score"], reverse=True)

        documents = [_rerank_text(r) for r in candidates]
        resp = await client.post(
            v.reranker_endpoint,
            json={"model": v.reranker_model, "query": query, "documents": documents, "top_n": len(documents)},
        )
        resp.raise_for_status()
        ranked = []
        for item in sorted(resp.json()["results"], key=lambda x: x["relevance_score"], reverse=True):
            r = dict(candidates[item["index"]])
            r["fusion_score"] = r["score"]
            r["score"] = item["relevance_score"]
            ranked.append(r)
        return ranked

    async def _neighbors(self, collection: str, source_id: str, chunk_index: int, radius: int) -> list:
        body = {
            "filter": {
                "must": [{"key": "chunk_index", "range": {"gte": chunk_index - radius, "lte": chunk_index + radius}}],
                "should": [
                    {"key": "document_id", "match": {"value": source_id}},
                    {"key": "episode_id", "match": {"value": source_id}},
                ],
            },
            "limit": 2 * radius + 1,
            "with_payload": True,
        }
        async with self._client() as client:
            resp = await client.post(
                f"{self.valves.qdrant_url}/collections/{collection}/points/scroll",
                json=body,
                headers=self._qdrant_headers(),
            )
            resp.raise_for_status()
            points = resp.json()["result"]["points"]
        results = [_to_result(collection, p) for p in points]
        return sorted(results, key=lambda r: r["chunk_index"] if r["chunk_index"] is not None else 0)

    @staticmethod
    async def _status(emitter, description: str, done: bool) -> None:
        if emitter:
            await emitter({"type": "status", "data": {"description": description, "done": done, "hidden": False}})


def _to_result(collection: str, point: dict) -> dict:
    p = point.get("payload") or {}
    return {
        "collection": collection,
        "source": _source_label(p),
        "source_id": p.get("document_id") or p.get("episode_id") or "",
        "chunk_index": p.get("chunk_index"),
        "score": point.get("score", 0.0),
        "context": p.get("context", ""),
        "text": p.get("text", ""),
    }


def _source_label(p: dict) -> str:
    if p.get("episode_id"):
        return f"podcast {p['episode_id']} @ {_clock(p.get('start_time'))}-{_clock(p.get('end_time'))}"
    return p.get("source_path") or p.get("file_name") or p.get("document_id", "unknown")


def _clock(seconds: Any) -> str:
    try:
        s = int(float(seconds))
    except (TypeError, ValueError):
        return "?"
    return f"{s // 3600:d}:{s % 3600 // 60:02d}:{s % 60:02d}"


def _rerank_text(r: dict) -> str:
    if r["context"]:
        return f"{r['context']}\n\n{r['text']}"
    return r["text"]
