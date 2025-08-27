# Knowledge Base Tool (TypeScript)

A minimal proof-of-concept knowledge base tool written in TypeScript that implements document ingestion, indexing, and search functionality. This is a port of the original Go implementation.

## Features

- **Document Ingestion**: Supports both local files and GitHub repositories (public and private with token)
- **Content Processing**: Normalizes content to Markdown, chunks documents intelligently
- **Lexical Search**: Builds and queries a token-based search index
- **CLI Interface**: Simple command-line interface with three main commands
- **File-based Storage**: No database required - everything stored as plain files

## Installation

```bash
npm install
npm run build
```

## Commands

### Build the Application

```bash
npm run build
```

### Ingest Documents

```bash
npm run dev ingest --list urls.txt --data ./data
```

With GitHub token for private repositories:

```bash
npm run dev ingest --list urls.txt --data ./data --github-token YOUR_TOKEN
```

### Search Index

```bash
npm run dev search --data ./data --top 10 "search query"
```

### Start HTTP Server (Stubbed)

```bash
npm run dev serve --data ./data --addr :8080 --top 10
```

*Note: HTTP server implementation is stubbed out in this POC*

## Architecture

### Package Structure

```
src/
├── cli/                  # CLI application entry point
├── internal/
│   ├── types.ts          # Shared TypeScript interfaces
│   ├── fetch/            # Content fetchers (GitHub, local files)
│   ├── pipe/             # Processing pipeline (normalize, chunk)
│   ├── index/            # Indexing (lexical search, file readers)  
│   ├── retr/             # Retrieval and search logic
│   └── api/              # HTTP handlers
```

### Data Storage Structure

All data is persisted as plain files under the `data/` directory:
- `manifest.jsonl` - One line per ingested document
- `blobs/<sha256>` - Normalized markdown content
- `chunks/<sha256>-<n>.md` - Individual text chunks
- `meta/<sha256>.json` - Document metadata (title, URL, timestamps)
- `index/index.json` - Token → chunk ID mappings
- `index/docmap.json` - Chunk ID → metadata mappings

### Key Features

#### GitHub Integration
- Supports various GitHub URL formats:
  - Single files: `https://github.com/owner/repo/blob/branch/path/to/file.ts`
  - Directories: `https://github.com/owner/repo/tree/branch/path/to/dir`
  - Repository root: `https://github.com/owner/repo`
- Uses GitHub Contents API for reliable content fetching
- Automatically handles base64-encoded content
- Works with both public and private repositories (with personal access token)

#### Content Processing
- Normalizes content to Markdown format
- Intelligently chunks documents (~1000 words per chunk)
- Respects headings and code blocks when chunking
- Preserves document structure in breadcrumbs

#### Search
- Token-based lexical search
- Simple frequency scoring
- No external dependencies for search functionality
- Fast file-based index lookup

## Development

### Development Mode

```bash
npm run dev <command> [args]
```

### Type Checking

```bash
npm run typecheck
```

### Clean Build

```bash
npm run clean
npm run build
```

## Migration from Go

This TypeScript implementation maintains API compatibility with the original Go version while providing:

- Modern async/await patterns
- Strong typing with TypeScript interfaces
- ES modules for better tree-shaking
- Node.js ecosystem integration
- Familiar npm/yarn tooling

The file formats and data structures remain identical, so you can use the same data directory between Go and TypeScript implementations.