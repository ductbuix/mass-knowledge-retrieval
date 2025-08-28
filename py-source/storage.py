import io
import json
import os
from typing import Dict


def write_markdown_with_frontmatter(path: str, frontmatter: Dict, markdown: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with io.open(path, "w", encoding="utf-8") as f:
        f.write("---\n")
        for k, v in frontmatter.items():
            # Write simple YAML-like frontmatter (no nested structures)
            if v is None:
                continue
            s = str(v).replace("\n", " ")
            f.write(f"{k}: {s}\n")
        f.write("---\n\n")
        f.write(markdown)
        if not markdown.endswith("\n"):
            f.write("\n")


def read_text_file(path: str) -> str:
    with io.open(path, "r", encoding="utf-8", errors="replace") as f:
        return f.read()


def load_json(path: str, default):
    if not os.path.exists(path):
        return default
    with io.open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def save_json(path: str, data) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with io.open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

