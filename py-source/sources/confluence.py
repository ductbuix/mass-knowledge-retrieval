import base64
import datetime as dt
import json
import os
import re
import time
from typing import Dict, Iterable, List, Optional, Tuple
from urllib import parse, request, error

from utils.html_to_markdown import html_to_markdown
from storage import write_markdown_with_frontmatter
from indexer import Index


def _slugify(value: str, max_len: int = 80) -> str:
    value = value.lower().strip()
    value = re.sub(r"[^a-z0-9\-\s_]", "", value)
    value = re.sub(r"[\s_]+", "-", value)
    if len(value) > max_len:
        value = value[:max_len]
    return value or "doc"


def _auth_header(auth_type: str, email: Optional[str], token: Optional[str]) -> str:
    if auth_type == "basic":
        if not (email and token):
            raise ValueError("Basic auth requires email and token (API token/password)")
        raw = f"{email}:{token}".encode("utf-8")
        return "Basic " + base64.b64encode(raw).decode("ascii")
    elif auth_type == "bearer":
        if not token:
            raise ValueError("Bearer auth requires token")
        return "Bearer " + token
    else:
        raise ValueError(f"Unsupported auth_type: {auth_type}")


def _http_get_json(url: str, headers: Dict[str, str]) -> Dict:
    req = request.Request(url, headers=headers)
    try:
        with request.urlopen(req) as resp:
            data = resp.read()
            return json.loads(data.decode("utf-8"))
    except error.HTTPError as e:
        raise RuntimeError(
            f"HTTP {e.code} for {url}: {e.read().decode('utf-8', 'ignore')}"
        )


def _build_api_base(host: str) -> str:
    # Allow callers to pass full API base directly
    if host.rstrip("/").endswith("/rest/api"):
        return host.rstrip("/")
    # Common patterns: https://<site>/wiki (Cloud) or https://<site> (Server/DC)
    host = host.rstrip("/")
    if host.endswith("/wiki"):
        return host + "/rest/api"
    return host + "/rest/api"


def api_base_from_page_url(url: str) -> str:
    parsed = parse.urlparse(url)
    origin = f"{parsed.scheme}://{parsed.netloc}"
    # If '/wiki' occurs near the root, assume Cloud-style web path
    if parsed.path.startswith("/wiki/") or parsed.path == "/wiki":
        return origin + "/wiki/rest/api"
    return origin + "/rest/api"


def parse_confluence_page_id(url: str) -> Optional[str]:
    # 1) Query param pageId
    parsed = parse.urlparse(url)
    qs = parse.parse_qs(parsed.query)
    if "pageId" in qs and qs["pageId"]:
        return qs["pageId"][0]

    # 2) Path patterns: /wiki/pages/<id>/..., /wiki/spaces/<space>/pages/<id>/...
    m = re.search(r"/(?:wiki/)?pages/(\d+)(?:/|$)", parsed.path)
    if m:
        return m.group(1)

    m = re.search(r"/(?:wiki/)?spaces/[^/]+/pages/(\d+)(?:/|$)", parsed.path)
    if m:
        return m.group(1)

    return None


def _fetch_page(api_base: str, auth_header: str, page_id: str) -> Dict:
    headers = {
        "Accept": "application/json",
        "Authorization": auth_header,
    }
    url = f"{api_base}/content/{page_id}?expand=body.storage,version,space,_links"
    return _http_get_json(url, headers)


def ingest_confluence_url(
    url: str,
    email: Optional[str],
    token: Optional[str],
    auth_type: str,
    out_dir: str,
    index: bool,
    data_dir: str,
) -> Optional[str]:
    page_id = parse_confluence_page_id(url)
    if not page_id:
        return None

    auth_header = _auth_header(auth_type, email, token)
    api_base = api_base_from_page_url(url)
    item = _fetch_page(api_base, auth_header, page_id)

    title = item.get("title") or "Untitled"
    space_key = (item.get("space") or {}).get("key")
    storage = ((item.get("body") or {}).get("storage") or {}).get("value") or ""
    version = (item.get("version") or {}).get("number")

    md = html_to_markdown(storage)
    slug = _slugify(title)
    fname = f"{slug}-{page_id}.md"
    path = os.path.join(out_dir, fname)

    frontmatter = {
        "source": "confluence",
        "confluence_id": page_id,
        "title": title,
        "url": url,
        "space": space_key,
        "version": version,
    }
    write_markdown_with_frontmatter(path, frontmatter, md)

    if index:
        idx = Index(data_dir=data_dir)
        idx.add_or_update_document(
            doc_id=f"confluence:{page_id}",
            title=title,
            path=path,
            source="confluence",
            url=url,
            text=md,
        )
        idx.save()

    return path


def _page_url(base: str, item: Dict) -> Optional[str]:
    links = item.get("_links") or {}
    if not links:
        return None
    base_url = links.get("base") or base.rsplit("/rest/api", 1)[0]
    webui = links.get("webui")
    if webui:
        return base_url.rstrip("/") + webui
    return None


def _iter_confluence_pages(
    api_base: str,
    auth_header: str,
    spaces: Iterable[str],
    limit: Optional[int] = None,
) -> Iterable[Dict]:
    headers = {
        "Accept": "application/json",
        "Authorization": auth_header,
    }

    total_yielded = 0
    space_list = list(spaces) if spaces else [None]

    for space in space_list:
        start = 0
        page_size = 100
        while True:
            params = {
                "expand": "body.storage,version,space,_links",
                "limit": str(page_size),
                "start": str(start),
            }
            if space:
                params["spaceKey"] = space

            url = f"{api_base}/content?{parse.urlencode(params)}"
            data = _http_get_json(url, headers)

            results = data.get("results", [])
            if not results:
                break

            for item in results:
                yield item
                total_yielded += 1
                if limit and total_yielded >= limit:
                    return

            if data.get("_links", {}).get("next"):
                start += page_size
                # Be polite if used against rate-limited APIs
                time.sleep(0.05)
            else:
                break


def ingest_confluence(
    host: str,
    email: Optional[str],
    token: Optional[str],
    auth_type: str,
    spaces: List[str],
    out_dir: str,
    limit: Optional[int],
    index: bool,
    data_dir: str,
) -> List[str]:
    os.makedirs(out_dir, exist_ok=True)
    api_base = _build_api_base(host)
    auth_header = _auth_header(auth_type, email, token)

    idx = Index(data_dir=data_dir)
    written: List[str] = []

    for item in _iter_confluence_pages(api_base, auth_header, spaces, limit):
        title = item.get("title") or "Untitled"
        page_id = item.get("id") or "unknown"
        space_key = (item.get("space") or {}).get("key")
        url = _page_url(api_base, item)
        version = (item.get("version") or {}).get("number")
        storage = ((item.get("body") or {}).get("storage") or {}).get("value") or ""

        md = html_to_markdown(storage)
        fetched_at = (
            dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat() + "Z"
        )

        slug = _slugify(title)
        fname = f"{slug}-{page_id}.md"
        path = os.path.join(out_dir, fname)

        frontmatter = {
            "source": "confluence",
            "confluence_id": page_id,
            "title": title,
            "url": url,
            "space": space_key,
            "version": version,
            "fetched_at": fetched_at,
        }
        write_markdown_with_frontmatter(path, frontmatter, md)

        # Index
        if index:
            doc_id = os.path.relpath(path, os.path.join(data_dir, "ingested"))
            idx.add_or_update_document(
                doc_id=doc_id,
                title=title,
                path=path,
                source="confluence",
                url=url,
                text=md,
            )

        written.append(path)

    if index:
        idx.save()

    return written
