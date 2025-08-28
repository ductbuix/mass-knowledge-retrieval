import os
from typing import List, Optional, Set

from indexer import Index
from storage import read_text_file
from .confluence import ingest_confluence_url


def _normalize_file_url_or_path(s: str) -> Optional[str]:
    s = s.strip()
    if not s:
        return None
    if s.startswith("file://"):
        # Convert file:// to local path
        from urllib import parse

        p = parse.urlparse(s)
        path = parse.unquote(p.path)
        if os.name == "nt" and path.startswith("/") and len(path) > 3 and path[2] == ":":
            path = path.lstrip("/")
        return os.path.abspath(path)
    # Plain path
    if "://" not in s and (s.startswith("/") or os.path.exists(s)):
        return os.path.abspath(s)
    return None


def _sanitize_abs_to_rel(path: str) -> str:
    # Make an absolute path safe as a relative path component
    rel = path.replace(":", "").lstrip("/")
    rel = rel.replace("\\", "/")
    # Avoid accidental traversal
    rel = rel.replace("../", "").replace("..\\", "")
    return rel


def _ingest_local_single(
    path: str, out_dir: str, index: bool, data_dir: str
) -> Optional[str]:
    if not os.path.isfile(path):
        return None
    ext = os.path.splitext(path)[1].lower()
    if ext not in {".md", ".markdown", ".txt"}:
        return None

    text = read_text_file(path)
    md = text
    rel = _sanitize_abs_to_rel(os.path.abspath(path))
    # maintain original extension
    target_path = os.path.join(out_dir, "local", rel)
    os.makedirs(os.path.dirname(target_path), exist_ok=True)

    from storage import write_markdown_with_frontmatter

    frontmatter = {
        "source": "local",
        "original_path": os.path.abspath(path),
        "title": os.path.splitext(os.path.basename(path))[0],
        "url": f"file://{os.path.abspath(path)}",
    }
    write_markdown_with_frontmatter(target_path, frontmatter, md)

    if index:
        idx = Index(data_dir=data_dir)
        idx.add_or_update_document(
            doc_id=f"local:{os.path.abspath(path)}",
            title=frontmatter["title"],
            path=target_path,
            source="local",
            url=frontmatter["url"],
            text=md,
        )
        idx.save()
    return target_path


def ingest_urls(
    urls_file: str,
    out_dir: str,
    data_dir: str,
    index: bool,
    conf_email: Optional[str],
    conf_token: Optional[str],
    conf_auth_type: str,
) -> List[str]:
    os.makedirs(out_dir, exist_ok=True)
    lines = [l.strip() for l in read_text_file(urls_file).splitlines()]
    written: List[str] = []

    for line in lines:
        if not line or line.startswith("#"):
            continue

        # Local file or path
        local_path = _normalize_file_url_or_path(line)
        if local_path:
            p = _ingest_local_single(local_path, out_dir, index=index, data_dir=data_dir)
            if p:
                written.append(p)
            continue

        # Confluence URL (page)
        if line.startswith("http://") or line.startswith("https://"):
            p = ingest_confluence_url(
                url=line,
                email=conf_email,
                token=conf_token,
                auth_type=conf_auth_type,
                out_dir=os.path.join(out_dir, "confluence"),
                index=index,
                data_dir=data_dir,
            )
            if p:
                written.append(p)
            continue

        # Unknown type — skip
        continue

    return written


def prune_missing_urls(urls_file: str, data_dir: str, ingested_base: str) -> None:
    # Build the keep set of URL identifiers
    lines = [l.strip() for l in read_text_file(urls_file).splitlines()]
    keep_urls: Set[str] = set(
        l if (l.startswith("http://") or l.startswith("https://")) else (
            f"file://{os.path.abspath(_normalize_file_url_or_path(l) or l)}"
        )
        for l in lines
        if l and not l.startswith("#")
    )

    idx = Index(data_dir=data_dir)
    store = idx.data.get("doc_store", {})

    # Find docs with url not in keep set
    to_delete = []
    for doc_id, meta in store.items():
        url = meta.get("url")
        if not url:
            continue
        if url not in keep_urls:
            to_delete.append(doc_id)

    # Delete from index and remove markdown files under ingested_base
    for doc_id in to_delete:
        meta = store.get(doc_id) or {}
        path = meta.get("path")
        # Remove file if inside the ingested base directory
        try:
            if path and os.path.commonpath([os.path.abspath(path), os.path.abspath(ingested_base)]) == os.path.abspath(ingested_base):
                if os.path.exists(path):
                    os.remove(path)
        except Exception:
            pass
        idx.delete_document(doc_id)

    if to_delete:
        idx.save()

