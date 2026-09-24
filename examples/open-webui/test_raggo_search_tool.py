"""Tests for raggo_search_tool. Run: pip install httpx pydantic pytest && pytest examples/"""

import asyncio
import json

import httpx

from raggo_search_tool import Tools


def make_tools(handler, **valves):
    tools = Tools()
    tools.valves = Tools.Valves(**{**tools.valves.model_dump(), **valves})
    tools._transport = httpx.MockTransport(handler)
    return tools


def point(pid, score, **payload):
    return {"id": pid, "score": score, "payload": payload}


def test_search_sends_hybrid_query_and_reranks():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content)
        path = request.url.path
        seen[path] = body
        if path == "/v1/embeddings":
            return httpx.Response(200, json={"data": [{"embedding": [0.1, 0.2]}]})
        if path.endswith("/points/query"):
            return httpx.Response(200, json={"result": {"points": [
                point("a", 0.9, document_id="d1", chunk_index=0, text="alpha", context="ctx a", file_name="a.pdf"),
                point("b", 0.8, document_id="d1", chunk_index=1, text="beta", context="", file_name="a.pdf"),
            ]}})
        if path == "/v1/rerank":
            return httpx.Response(200, json={"results": [
                {"index": 0, "relevance_score": 0.1},
                {"index": 1, "relevance_score": 0.7},
            ]})
        return httpx.Response(404)

    tools = make_tools(handler, qdrant_url="http://q:6333", collections="docs",
                       embed_endpoint="http://m/v1/embeddings", reranker_endpoint="http://m/v1/rerank", top_k=5)

    results = asyncio.run(tools._search("what is beta?"))

    assert seen["/v1/embeddings"]["input"].endswith("Query: what is beta?")
    q = seen["/collections/docs/points/query"]
    assert q["query"] == {"fusion": "rrf"}
    assert q["prefetch"][0] == {"query": [0.1, 0.2], "using": "dense", "limit": 20}
    assert q["prefetch"][1] == {"query": {"text": "what is beta?", "model": "qdrant/bm25"}, "using": "bm25", "limit": 20}
    assert seen["/v1/rerank"]["documents"] == ["ctx a\n\nalpha", "beta"]

    assert [r["text"] for r in results] == ["beta", "alpha"]
    assert results[0]["score"] == 0.7 and results[0]["fusion_score"] == 0.8
    assert results[0]["source"] == "a.pdf" and results[0]["source_id"] == "d1"


def test_search_without_reranker_uses_fusion_order_and_raggo_embed_api():
    def handler(request: httpx.Request) -> httpx.Response:
        path = request.url.path
        if path == "/embed":
            assert json.loads(request.content)["text"].endswith("q")
            return httpx.Response(200, json={"embedding": [1.0]})
        return httpx.Response(200, json={"result": {"points": [
            point("a", 0.2, episode_id="e1", chunk_index=3, start_time=65, end_time=125, text="low"),
            point("b", 0.5, episode_id="e1", chunk_index=4, start_time=120, end_time=180, text="high"),
        ]}})

    tools = make_tools(handler, embed_api="raggo", embed_endpoint="http://e/embed", reranker_endpoint="", top_k=1)
    results = asyncio.run(tools._search("q"))

    assert [r["text"] for r in results] == ["high"]
    assert results[0]["source"] == "podcast e1 @ 0:02:00-0:03:00"


def test_neighbors_filters_by_source_and_range():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["body"] = json.loads(request.content)
        seen["path"] = request.url.path
        return httpx.Response(200, json={"result": {"points": [
            point("c", 0, document_id="d1", chunk_index=6, text="after"),
            point("a", 0, document_id="d1", chunk_index=4, text="before"),
        ]}})

    tools = make_tools(handler, qdrant_url="http://q:6333")
    results = asyncio.run(tools._neighbors("docs", "d1", 5, 1))

    assert seen["path"] == "/collections/docs/points/scroll"
    f = seen["body"]["filter"]
    assert f["must"] == [{"key": "chunk_index", "range": {"gte": 4, "lte": 6}}]
    assert {"key": "document_id", "match": {"value": "d1"}} in f["should"]
    assert [r["text"] for r in results] == ["before", "after"]


def test_search_knowledge_emits_citations_and_returns_json():
    events = []

    async def emitter(event):
        events.append(event)

    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/v1/embeddings":
            return httpx.Response(200, json={"data": [{"embedding": [0.0]}]})
        return httpx.Response(200, json={"result": {"points": [
            point("a", 0.9, document_id="d1", chunk_index=0, text="alpha", source_path="docs/a.md"),
        ]}})

    tools = make_tools(handler, embed_endpoint="http://m/v1/embeddings", reranker_endpoint="")
    out = json.loads(asyncio.run(tools.search_knowledge("alpha", __event_emitter__=emitter)))

    assert out[0]["source"] == "docs/a.md"
    citations = [e for e in events if e["type"] == "citation"]
    assert citations[0]["data"]["document"] == ["alpha"]
    assert events[-1] == {"type": "status", "data": {"description": "Found 1 passages", "done": True, "hidden": False}}


def test_search_knowledge_reports_http_errors():
    tools = make_tools(lambda request: httpx.Response(503, text="loading"))
    out = json.loads(asyncio.run(tools.search_knowledge("x")))
    assert "error" in out
