import math
import os
import re
from collections import Counter, defaultdict
from typing import Dict, Iterable, List, Optional, Tuple

from storage import load_json, save_json


DEFAULT_DATA_DIR = "data"


STOPWORDS = {
    "the",
    "a",
    "an",
    "and",
    "or",
    "is",
    "are",
    "was",
    "were",
    "be",
    "to",
    "of",
    "in",
    "on",
    "for",
    "with",
    "as",
    "by",
    "at",
    "from",
    "that",
    "this",
    "it",
    "we",
    "you",
    "your",
    "our",
}


TOKEN_RE = re.compile(r"[A-Za-z0-9_]+")


def tokenize(text: str) -> List[str]:
    tokens = [t.lower() for t in TOKEN_RE.findall(text)]
    return [t for t in tokens if t not in STOPWORDS and not t.isdigit()]


class Index:
    def __init__(self, data_dir: str = DEFAULT_DATA_DIR):
        self.data_dir = data_dir
        self.index_path = os.path.join(data_dir, "index.json")
        self.data = load_json(
            self.index_path,
            default={
                "doc_count": 0,
                "doc_store": {},  # doc_id -> {title, path, source, url, doc_len}
                "term_df": {},  # term -> df
                "postings": {},  # term -> {doc_id -> tf}
            },
        )

    # -------------------- Persistence --------------------
    def save(self) -> None:
        save_json(self.index_path, self.data)

    # -------------------- Document ops --------------------
    def add_or_update_document(
        self,
        doc_id: str,
        title: str,
        path: str,
        source: str,
        url: Optional[str],
        text: str,
    ) -> None:
        # Remove previous postings if doc exists
        if doc_id in self.data["doc_store"]:
            self._remove_document_postings(doc_id)

        tokens = tokenize(text)
        tf = Counter(tokens)
        doc_len = len(tokens)

        # Update doc_store
        self.data["doc_store"][doc_id] = {
            "title": title,
            "path": path,
            "source": source,
            "url": url,
            "doc_len": doc_len,
        }

        # Update postings and DF
        postings = self.data["postings"]
        term_df = self.data["term_df"]

        for term, count in tf.items():
            if term not in postings:
                postings[term] = {}
            postings[term][doc_id] = int(count)
        for term in tf.keys():
            term_df[term] = int(len(postings[term]))

        # Update doc_count (unique documents)
        self.data["doc_count"] = len(self.data["doc_store"])

    def _remove_document_postings(self, doc_id: str) -> None:
        postings = self.data["postings"]
        term_df = self.data["term_df"]

        to_delete_terms: List[str] = []
        for term, plist in postings.items():
            if doc_id in plist:
                del plist[doc_id]
                if not plist:
                    to_delete_terms.append(term)
                else:
                    term_df[term] = int(len(plist))
        for t in to_delete_terms:
            postings.pop(t, None)
            term_df.pop(t, None)

        self.data["doc_store"].pop(doc_id, None)

    def delete_document(self, doc_id: str) -> None:
        if doc_id not in self.data.get("doc_store", {}):
            return
        self._remove_document_postings(doc_id)
        # Recompute doc_count
        self.data["doc_count"] = len(self.data.get("doc_store", {}))

    # -------------------- Scoring --------------------
    def _idf(self, term: str) -> float:
        # Smoothing to avoid div by zero
        df = self.data["term_df"].get(term, 0)
        n = max(1, int(self.data.get("doc_count", 1)))
        return math.log((n - df + 0.5) / (df + 0.5) + 1)

    def search_bm25(
        self, query: str, top_k: int = 5, k1: float = 1.5, b: float = 0.75
    ) -> List[Tuple[str, float]]:
        terms = tokenize(query)
        if not terms:
            return []

        postings = self.data.get("postings", {})
        store = self.data.get("doc_store", {})
        n_docs = max(1, int(self.data.get("doc_count", 1)))
        avg_len = (
            sum(d.get("doc_len", 0) for d in store.values()) / n_docs if n_docs else 0.0
        )

        scores: Dict[str, float] = defaultdict(float)
        terms_set = list(dict.fromkeys(terms))  # unique in order

        for term in terms_set:
            plist = postings.get(term)
            if not plist:
                continue
            idf = self._idf(term)
            for doc_id, tf in plist.items():
                doc_len = store.get(doc_id, {}).get("doc_len", 0)
                denom = tf + k1 * (1 - b + b * (doc_len / (avg_len or 1.0)))
                score = idf * (tf * (k1 + 1)) / (denom or 1.0)
                scores[doc_id] += score

        ranked = sorted(scores.items(), key=lambda x: x[1], reverse=True)[:top_k]
        return ranked
