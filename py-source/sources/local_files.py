import fnmatch
import os
from typing import List, Optional

from storage import write_markdown_with_frontmatter, read_text_file
from indexer import Index


SUPPORTED_EXT = {".md", ".markdown", ".txt"}


def _should_include(path: str, pattern: str) -> bool:
    # We use fnmatch with Unix-style patterns; to emulate "**/*" recursing, we already walk
    name = os.path.relpath(path)
    return fnmatch.fnmatch(name, pattern)


def ingest_local(
    root: str,
    pattern: str,
    out_dir: str,
    limit: Optional[int],
    index: bool,
    data_dir: str,
) -> List[str]:
    os.makedirs(out_dir, exist_ok=True)

    idx = Index(data_dir=data_dir)
    written: List[str] = []
    count = 0

    if not os.path.isdir(root):
        return []

    for dirpath, _dirnames, filenames in os.walk(root):
        for fname in filenames:
            ext = os.path.splitext(fname)[1].lower()
            if ext not in SUPPORTED_EXT:
                continue
            src_path = os.path.join(dirpath, fname)
            rel = os.path.relpath(src_path, root)
            if not _should_include(rel, pattern):
                continue

            text = read_text_file(src_path)
            if ext in {".txt"}:
                md = text
            else:
                md = text  # Already Markdown

            target_path = os.path.join(out_dir, rel)
            os.makedirs(os.path.dirname(target_path), exist_ok=True)

            frontmatter = {
                "source": "local",
                "original_path": os.path.abspath(src_path),
                "title": os.path.splitext(os.path.basename(src_path))[0],
            }
            write_markdown_with_frontmatter(target_path, frontmatter, md)

            if index:
                doc_id = os.path.relpath(
                    target_path, os.path.join(data_dir, "ingested")
                )
                idx.add_or_update_document(
                    doc_id=doc_id,
                    title=frontmatter["title"],
                    path=target_path,
                    source="local",
                    url=None,
                    text=md,
                )

            written.append(target_path)
            count += 1
            if limit and count >= limit:
                if index:
                    idx.save()
                return written

    if index:
        idx.save()
    return written
