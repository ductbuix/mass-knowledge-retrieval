import argparse
import os
from typing import List, Optional

from sources.confluence import ingest_confluence
from sources.local_files import ingest_local
from retrieval import retrieve
from config import load_config, default_config_path, cfg_get


DEFAULT_DATA_DIR = "data"
DEFAULT_OUT_DIR = os.path.join(DEFAULT_DATA_DIR, "ingested")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="lc-knowledge",
        description="CLI to ingest and retrieve knowledge from multiple sources",
    )

    subparsers = parser.add_subparsers(dest="command", required=True)

    # Ingest command
    ingest_p = subparsers.add_parser(
        "ingest", help="Ingest data from sources and build/update index"
    )
    ingest_p.add_argument(
        "--config",
        default=None,
        help="Path to config.json; defaults to ./config.json if present",
    )
    ingest_p.add_argument(
        "--out-dir",
        default=DEFAULT_OUT_DIR,
        help="Directory where ingested markdown files are stored",
    )
    ingest_p.add_argument(
        "--urls-file",
        default=None,
        help="Path to a text file with URLs (http(s) or file paths)",
    )
    ingest_p.add_argument(
        "--sync-prune",
        action="store_true",
        help="When using --urls-file, remove indexed docs not present in file",
    )
    ingest_p.add_argument(
        "--sources",
        nargs="+",
        choices=["confluence", "local"],
        default=["local"],
        help="Sources to ingest",
    )
    # Local files options
    ingest_p.add_argument(
        "--local-path",
        default="local-data",
        help="Root folder for local files ingestion (txt/md)",
    )
    ingest_p.add_argument(
        "--local-glob",
        default="**/*",
        help="Glob pattern relative to local-path to include",
    )
    # Confluence options
    ingest_p.add_argument("--conf-host", default=os.environ.get("CONFLUENCE_HOST"))
    ingest_p.add_argument("--conf-email", default=os.environ.get("CONFLUENCE_EMAIL"))
    ingest_p.add_argument(
        "--conf-token", default=os.environ.get("CONFLUENCE_API_TOKEN")
    )
    ingest_p.add_argument(
        "--conf-space",
        action="append",
        default=[],
        help="Confluence space key to ingest; can be passed multiple times",
    )
    ingest_p.add_argument(
        "--conf-auth-type",
        choices=["basic", "bearer"],
        default=os.environ.get("CONFLUENCE_AUTH_TYPE", "basic"),
    )
    ingest_p.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Optional limit of docs to fetch from each source",
    )
    ingest_p.add_argument(
        "--no-index",
        action="store_true",
        help="Skip indexing after ingestion (not recommended)",
    )

    # Retrieve command
    retr_p = subparsers.add_parser(
        "retrieve", help="Search ingested content by a query string"
    )
    retr_p.add_argument(
        "--config",
        default=None,
        help="Path to config.json; defaults to ./config.json if present",
    )
    retr_p.add_argument("query", help="Query string to search")
    retr_p.add_argument("--top-k", type=int, default=5, help="Number of results")
    retr_p.add_argument(
        "--data-dir",
        default=DEFAULT_DATA_DIR,
        help="Base data directory containing ingested/ and index.json",
    )
    retr_p.add_argument(
        "--show-snippets",
        action="store_true",
        help="Show matching text snippets from documents",
    )

    return parser


def cmd_ingest(args: argparse.Namespace) -> None:
    # Load config (if any) and merge
    cfg_path = args.config or default_config_path()
    cfg = load_config(cfg_path)

    # Resolve directories
    out_dir = args.out_dir
    cfg_out_dir = cfg_get(cfg, "ingest.out_dir")
    if cfg_out_dir and out_dir == DEFAULT_OUT_DIR:
        out_dir = cfg_out_dir

    data_dir = os.path.dirname(out_dir) if out_dir else DEFAULT_DATA_DIR
    cfg_data_dir = cfg_get(cfg, "data_dir")
    if cfg_data_dir:
        # No data_dir CLI on ingest; prefer config if provided
        data_dir = cfg_data_dir
    os.makedirs(out_dir, exist_ok=True)

    ingested_paths: List[str] = []

    # Prefer URLs-file mode if provided in CLI or config
    urls_file = args.urls_file or cfg_get(cfg, "ingest.urls_file")
    sync_prune = args.sync_prune or bool(cfg_get(cfg, "ingest.sync_prune", False))

    if urls_file:
        from sources.urls_ingest import ingest_urls, prune_missing_urls

        ingested_paths += ingest_urls(
            urls_file=urls_file,
            out_dir=out_dir,
            data_dir=data_dir,
            index=not args.no_index if cfg_get(cfg, "ingest.index") is None else bool(cfg_get(cfg, "ingest.index")),
            conf_email=args.conf_email or cfg_get(cfg, "ingest.confluence.email"),
            conf_token=args.conf_token or cfg_get(cfg, "ingest.confluence.token"),
            conf_auth_type=args.conf_auth_type or cfg_get(cfg, "ingest.confluence.auth_type", "basic"),
        )

        if sync_prune and (not args.no_index if cfg_get(cfg, "ingest.index") is None else bool(cfg_get(cfg, "ingest.index"))):
            prune_missing_urls(
                urls_file=urls_file,
                data_dir=data_dir,
                ingested_base=out_dir,
            )

        print(f"Ingested {len(ingested_paths)} documents from URLs file.")
        return

    # Merge source selections
    sources = args.sources
    cfg_sources = cfg_get(cfg, "ingest.sources")
    if cfg_sources and sources == ["local"]:
        sources = list(cfg_sources)

    if "local" in sources:
        ingested_paths += ingest_local(
            root=cfg_get(cfg, "ingest.local.path") or args.local_path,
            pattern=cfg_get(cfg, "ingest.local.glob") or args.local_glob,
            out_dir=os.path.join(out_dir, "local"),
            limit=args.limit if args.limit is not None else cfg_get(cfg, "ingest.limit"),
            index=not args.no_index,
            data_dir=data_dir,
        )

    if "confluence" in sources:
        conf_host = args.conf_host or cfg_get(cfg, "ingest.confluence.host")
        conf_email = args.conf_email or cfg_get(cfg, "ingest.confluence.email")
        conf_token = args.conf_token or cfg_get(cfg, "ingest.confluence.token")
        conf_auth_type = args.conf_auth_type or cfg_get(
            cfg, "ingest.confluence.auth_type", "basic"
        )
        conf_spaces = args.conf_space or cfg_get(cfg, "ingest.confluence.spaces", [])

        if not conf_host:
            raise SystemExit(
                "--conf-host or CONFLUENCE_HOST must be provided for Confluence"
            )
        ingested_paths += ingest_confluence(
            host=conf_host,
            email=conf_email,
            token=conf_token,
            auth_type=conf_auth_type,
            spaces=conf_spaces,
            out_dir=os.path.join(out_dir, "confluence"),
            limit=args.limit if args.limit is not None else cfg_get(cfg, "ingest.limit"),
            index=not args.no_index if cfg_get(cfg, "ingest.index") is None else bool(cfg_get(cfg, "ingest.index")),
            data_dir=data_dir,
        )

    print(f"Ingested {len(ingested_paths)} documents.")


def cmd_retrieve(args: argparse.Namespace) -> None:
    cfg_path = args.config or default_config_path()
    cfg = load_config(cfg_path)

    data_dir = args.data_dir
    cfg_data_dir = cfg_get(cfg, "retrieve.data_dir") or cfg_get(cfg, "data_dir")
    if cfg_data_dir and data_dir == DEFAULT_DATA_DIR:
        data_dir = cfg_data_dir

    top_k = args.top_k
    cfg_top_k = cfg_get(cfg, "retrieve.top_k")
    if cfg_top_k and top_k == 5:
        top_k = int(cfg_top_k)

    show_snippets = args.show_snippets or bool(
        cfg_get(cfg, "retrieve.show_snippets", False)
    )

    results = retrieve(
        query=args.query,
        data_dir=data_dir,
        top_k=top_k,
        show_snippets=show_snippets,
    )

    if not results:
        print("No results.")
        return

    for i, r in enumerate(results, 1):
        print(f"{i}. {r['title']} [{r['source']}] -> {r['path']}")
        if r.get("url"):
            print(f"   URL: {r['url']}")
        print(f"   Score: {r['score']:.4f}")
        if args.show_snippets and r.get("snippet"):
            print("   ---")
            print("   " + r["snippet"].replace("\n", "\n   "))
            print("   ---")


def main(argv: Optional[List[str]] = None) -> None:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.command == "ingest":
        cmd_ingest(args)
    elif args.command == "retrieve":
        cmd_retrieve(args)
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
