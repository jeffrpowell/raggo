# raggo in an Open WebUI Computer agentic session

[Open WebUI Computer](https://github.com/open-webui/computer) (`cptr`) runs an agent on your real machine that has a shell. Instead of a chat plugin, it gets:

- [`raggo_search.py`](raggo_search.py): a CLI that wraps the same retrieval code as the [Open WebUI Tool](../open-webui/). It runs dense + BM25 hybrid retrieval, RRF fusion and reranking, and can also look up neighboring chunks.
- [`SKILL.md`](SKILL.md): a skill that teaches the agent an **agentic retrieval loop**. The agent breaks the question into sub-queries, searches, reads neighbors around good hits, and re-queries with vocabulary from the chunk contexts. Then it answers with citations.

The chat Tool gets one retrieval pass per tool call. In an agentic session the model can iterate, and raggo's per-chunk `context` lines give it concrete terms to refine its queries with.

## Setup on the Computer host

```bash
git clone <your raggo remote> ~/raggo
pip install httpx pydantic
export RAGGO_EXAMPLES=~/raggo/examples
export RAGGO_QDRANT_URL=http://localhost:6333
export RAGGO_COLLECTIONS=company-docs,tech-podcast
export RAGGO_EMBED_ENDPOINT=http://localhost:8080/v1/embeddings
export RAGGO_EMBED_MODEL=embed-qwen3-0.6b
export RAGGO_RERANKER_ENDPOINT=http://localhost:8080/v1/rerank
export RAGGO_RERANKER_MODEL=qwen3-reranker-0.6b

python3 $RAGGO_EXAMPLES/open-webui-computer/raggo_search.py search "test" --top-k 1
```

Put the exports wherever Computer's shell picks up your environment (for example `~/.bashrc`). Every Tool Valve maps to a `RAGGO_<VALVE>` variable.

Then add `SKILL.md` as a skill in Computer (Skills settings). It follows Open WebUI's skill format, so you can also import it into Open WebUI under **Workspace → Skills → Import** and pair it with Open Terminal.

## Example session

> **You:** What did we decide about retention for audit logs, and did the podcast ever discuss it?
>
> **Agent** (runs through `run_command`):
> ```
> raggo_search.py search "audit log retention decision"
> raggo_search.py neighbors company-docs 3f9c… 14 --radius 2
> raggo_search.py search "audit log retention" --collection tech-podcast
> raggo_search.py search "SOC 2 log retention 400 days"   # refined using terms from the context lines
> ```
> **Agent:** Audit logs are kept for 400 days. That meets the SOC 2 evidence window and allows one extra quarter of headroom [policies/logging.md]. Episode 42 discusses the trade-off between cold storage and retention [podcast 9a1e… @ 0:31:05-0:32:05] …

## Going further

Computer also supports MCP tool servers (`pip install 'cptr[mcp]'`). If you want structured tool calls instead of shell commands, `Tools` in `raggo_search_tool.py` can be wrapped in a small MCP server with the same two functions. These examples don't include one.
