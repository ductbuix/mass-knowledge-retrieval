import os
import re
from typing import Dict, List, Optional

from indexer import Index, tokenize


def _make_snippet(path: str, terms: List[str], max_len: int = 300) -> Optional[str]:
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            text = f.read()
    except Exception:
        return None

    # Remove frontmatter for snippet
    if text.startswith("---\n"):
        end = text.find("\n---\n")
        if end != -1:
            text = text[end + 5 :]

    # Find first occurrence of any term (case-insensitive)
    pattern = re.compile("|".join(re.escape(t) for t in set(terms)), re.IGNORECASE)
    m = pattern.search(text) if terms else None
    if not m:
        snippet = text.strip().splitlines()[:5]
        return "\n".join(snippet)

    start = max(0, m.start() - max_len // 2)
    end = min(len(text), start + max_len)
    snippet = text[start:end]
    return snippet.strip()


def retrieve(
    query: str, data_dir: str = "data", top_k: int = 5, show_snippets: bool = False
) -> List[Dict]:
    idx = Index(data_dir=data_dir)
    ranked = idx.search_bm25(query=query, top_k=top_k)

    results: List[Dict] = []
    terms = tokenize(query)
    for doc_id, score in ranked:
        meta = idx.data.get("doc_store", {}).get(doc_id) or {}
        path = meta.get("path")
        item = {
            "doc_id": doc_id,
            "title": meta.get("title", os.path.basename(path or doc_id)),
            "path": path,
            "source": meta.get("source", "unknown"),
            "url": meta.get("url"),
            "score": float(score),
        }
        if show_snippets and path:
            item["snippet"] = _make_snippet(path, terms)
        results.append(item)
    return results

