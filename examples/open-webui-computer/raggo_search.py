#!/usr/bin/env python3
"""CLI over the raggo Open WebUI tool, for shell-driven agents (Open WebUI Computer).

Every Valve can be set with an environment variable named RAGGO_<VALVE>, e.g.
RAGGO_QDRANT_URL, RAGGO_COLLECTIONS, RAGGO_EMBED_ENDPOINT, RAGGO_RERANKER_ENDPOINT.

  raggo_search.py search "how do we rotate the signing keys?"
  raggo_search.py search "key rotation" --collection company-docs --json
  raggo_search.py neighbors company-docs <source_id> <chunk_index> --radius 2
"""

import argparse
import asyncio
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "open-webui"))

from raggo_search_tool import Tools  # noqa: E402


def load_tools() -> Tools:
    tools = Tools()
    overrides = {}
    for name in Tools.Valves.model_fields:
        value = os.environ.get(f"RAGGO_{name.upper()}")
        if value is not None:
            overrides[name] = value
    tools.valves = Tools.Valves(**{**tools.valves.model_dump(), **overrides})
    return tools


def print_results(results: list, as_json: bool) -> None:
    if as_json:
        print(json.dumps(results, indent=2, ensure_ascii=False))
        return
    for i, r in enumerate(results, 1):
        print(f"[{i}] {r['source']}  (score={r['score']:.3f}, collection={r['collection']}, "
              f"source_id={r['source_id']}, chunk_index={r['chunk_index']})")
        if r["context"]:
            print(f"    context: {r['context']}")
        print(f"    {r['text']}\n")


async def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    s = sub.add_parser("search", help="Hybrid search + rerank")
    s.add_argument("query")
    s.add_argument("--collection", help="Search only this collection")
    s.add_argument("--top-k", type=int, help="Override the number of results")
    s.add_argument("--json", action="store_true")

    n = sub.add_parser("neighbors", help="Fetch chunks around a hit")
    n.add_argument("collection")
    n.add_argument("source_id")
    n.add_argument("chunk_index", type=int)
    n.add_argument("--radius", type=int, default=1)
    n.add_argument("--json", action="store_true")

    args = parser.parse_args()
    tools = load_tools()

    try:
        if args.command == "search":
            if args.top_k:
                tools.valves.top_k = args.top_k
            results = await tools._search(args.query, args.collection)
        else:
            results = await tools._neighbors(args.collection, args.source_id, args.chunk_index, args.radius)
    except Exception as e:  # surface HTTP/config errors to the agent in one line
        print(f"error: {e}", file=sys.stderr)
        return 1

    print_results(results, args.json)
    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
