LC Knowledge Retrieval (Python CLI)

Simple CLI to ingest content from Confluence and local files, convert to Markdown, and retrieve results with a lightweight BM25 full‑text search index.

Quick start

- Python 3.9+ recommended. No external dependencies required.
- Commands run from repository root (module files are in `py-source/`).

Configuration

- Place a `config.json` in the repo root (or pass `--config path/to/config.json`).
- CLI flags override config values; config overrides built‑in defaults and env.

Example `config.json`:
```
{
  "data_dir": "data",
  "ingest": {
    "out_dir": "data/ingested",
    "urls_file": "./urls.txt",
    "sync_prune": true,
    "sources": ["local", "confluence"],
    "limit": 100,
    "index": true,
    "local": { "path": "./local-data", "glob": "**/*" },
    "confluence": {
      "host": "https://your-domain.atlassian.net/wiki",
      "email": "you@example.com",
      "token": "<token>",
      "auth_type": "basic",
      "spaces": ["ENG"]
    }
  },
  "retrieve": { "data_dir": "data" }
}
```
You may also set retrieval defaults, e.g.:
```
{
  "retrieve": { "data_dir": "data", "top_k": 8, "show_snippets": true }
}
```

CLI usage

- Minimal ingest using config.json (auto‑loaded if present):
  python py-source/main.py ingest

- Minimal search using config.json data_dir:
  python py-source/main.py retrieve "how to deploy" --top-k 5 --show-snippets

- Ingest local files (txt/md):
  python py-source/main.py ingest --sources local --local-path ./local-data

- Ingest Confluence pages (requires credentials):
  export CONFLUENCE_HOST="https://your-domain.atlassian.net/wiki"
  export CONFLUENCE_EMAIL="you@example.com"
  export CONFLUENCE_API_TOKEN="<token>"
  python py-source/main.py ingest --sources confluence --conf-space ENG --limit 50

- Combine sources:
  python py-source/main.py ingest --sources local confluence --local-path ./local-data --conf-space ENG

- Ingest from a URLs file (sync style):
  - Each non-empty line is either an http(s) Confluence page URL or a local file path / file:// URL
  - Example urls.txt already exists in repo root
  - Using config.json:
  >   { "ingest": { "urls_file": "./urls.txt", "sync_prune": true } }
  python py-source/main.py ingest --config ./config.json

  - Or via flags (overrides config):
  python py-source/main.py ingest --urls-file ./urls.txt --sync-prune

  Notes:
  - Confluence URLs require credentials via env or flags (see above).
  - --sync-prune removes indexed docs whose URL is not listed.

- Retrieve by query:
  python py-source/main.py retrieve "how to deploy" --top-k 5 --show-snippets

Data layout

- Ingested Markdown files: `data/ingested/<source>/...`
- Search index: `data/index.json`

Notes

- Confluence API base: pass `--conf-host` including `/wiki` for Atlassian Cloud (e.g. `https://<site>.atlassian.net/wiki`) or plain host for Server/DC (e.g. `https://confluence.example.com`).
- The HTML→Markdown converter covers common tags (headings, lists, links, bold/italic, images, code blocks). Some complex Confluence macros may render as plain text.
- BM25 retrieval is lightweight and dependency‑free; good for keyword search. You can later swap in embeddings or a vector DB if desired.
