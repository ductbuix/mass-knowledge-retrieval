# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a minimal proof-of-concept knowledge base tool written in Go that implements document ingestion, indexing, and search functionality. The system ingests local files and GitHub repositories, normalizes them to Markdown, chunks the content, and builds a lexical index for keyword-based retrieval.

## Commands

### Building the Application
```bash
go build -o bin/kb cmd/kb/main.go
```

### Running Commands
The application provides three main subcommands:

#### Ingest Documents
```bash
./bin/kb ingest --list urls.txt --data ./data
```

With GitHub token for private repositories:
```bash
./bin/kb ingest --list urls.txt --data ./data --github-token YOUR_TOKEN
```

- Reads URLs/file paths from a newline-separated list file
- Supports both local files and GitHub URLs (public and private with token)
- Processes files into normalized markdown, chunks them, and builds metadata
- GitHub URLs can be individual files, directories, or entire repositories

#### Search Index
```bash
./bin/kb search --data ./data --top 10 "search query"
```
- Performs keyword search against the built index
- Returns top N results with previews

#### Start HTTP Server
```bash
./bin/kb serve --data ./data --addr :8080 --top 10
```
- Note: HTTP server implementation is stubbed out in this POC

## Architecture

### Package Structure
```
cmd/kb/               # CLI: ingest, search, serve commands
internal/
├── types.go          # Shared data structures
├── fetch/            # Content fetchers (GitHub, local files)
├── pipe/             # Processing pipeline (normalize, chunk)
├── index/            # Indexing (lexical search, file readers)  
├── retr/             # Retrieval and search logic
└── api/              # HTTP handlers
```

### Data Storage Structure
All data is persisted as plain files under the `data/` directory:
- `manifest.jsonl` - One line per ingested document
- `blobs/<sha256>` - Normalized markdown content
- `chunks/<sha256>-<n>.md` - Individual text chunks
- `meta/<sha256>.json` - Document metadata (title, URL, timestamps)
- `index/index.json` - Token → chunk ID mappings
- `index/docmap.json` - Chunk ID → metadata mappings

### Key Components

#### internal/fetch/
- **GitHub integration**: Supports various GitHub URL formats, API authentication, base64 content decoding
- **Local files**: Direct file system access with automatic code formatting
- **Content formatting**: Wraps code files in fenced blocks based on extension

#### internal/pipe/
- **Normalization**: Processes artifacts into standardized format, manages file persistence
- **Chunking**: Splits documents into ~1000 word chunks, respects headings and code blocks

#### internal/index/
- **Lexical indexing**: Builds token-based search index using simple tokenization
- **File readers**: Utilities for reading chunk previews and JSON data

#### internal/retr/
- **Search**: Naive keyword frequency scoring, result ranking and formatting
- **Hybrid approach**: Designed to support future vector search integration

#### internal/api/
- **HTTP handlers**: JSON API endpoints for search functionality

### GitHub Integration
- Supports various GitHub URL formats:
  - Single files: `https://github.com/owner/repo/blob/branch/path/to/file.go`
  - Directories: `https://github.com/owner/repo/tree/branch/path/to/dir`
  - Repository root: `https://github.com/owner/repo`
- Uses GitHub Contents API for reliable content fetching
- Automatically handles base64-encoded content
- Preserves repository structure in breadcrumbs
- Works with both public repositories and private repositories (with personal access token)

### Authentication
For private GitHub repositories, set the `--github-token` flag with a personal access token that has appropriate repository access permissions.

## Development Notes

- Clean separation of concerns with focused packages
- Uses Go standard library plus HTTP client for GitHub API
- Document identity is based on SHA256 hash of normalized content
- Search uses basic word frequency scoring without advanced NLP
- HTTP server functionality is not implemented (stubbed out)
- All packages are internal to prevent external usage
- Extensible architecture supports future enhancements (vector search, MMR reranking)