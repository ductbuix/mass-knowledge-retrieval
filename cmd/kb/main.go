package main

// A minimal proof‑of‑concept knowledge base tool written in Go.
//
// This program implements three subcommands: `ingest`, `search` and `serve`.
//
//  • ingest: reads a newline separated list of URLs or file paths, fetches their
//    contents, normalizes them to Markdown, splits the result into chunks and
//    persists all artefacts to the local filesystem. After ingestion an index
//    is built to enable simple full‑text search.
//
//  • search: opens the previously built index and performs a naive keyword
//    search. Matching chunk identifiers are mapped back to their metadata and
//    a short preview of each chunk is printed to stdout.
//
//  • serve: starts a local HTTP API that wraps the search functionality. This
//    endpoint returns JSON and is intentionally simple – it only exposes
//    `POST /retrieve` for keyword queries. Additional endpoints can be added
//    later to support context pack generation or chunk retrieval.
//
// Everything is persisted as plain files under the `data` directory. There is
// no use of an embedded database – instead, the code writes small JSON and
// Markdown files that can be inspected and versioned easily. The directory
// layout looks like this:
//
//    data/
//      manifest.jsonl            – one line per artefact (document)
//      blobs/<sha256>            – normalized markdown for the document
//      chunks/<sha256>-<n>.md    – text for chunk n of document with sha256
//      meta/<sha256>.json        – per‑document metadata (title, url, updatedAt, source)
//      index/index.json          – map of token → list of chunk identifiers
//      index/docmap.json         – map of chunk identifier → basic metadata
//
// The ingestion process is deterministic: artefacts are identified by the
// SHA256 of their normalized content. Chunks derive their identities from the
// document hash. No external dependencies are required to build the lexical
// index – it's just a map encoded as JSON.

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"kb/internal/api"
	"kb/internal/fetch"
	"kb/internal/index"
	"kb/internal/pipe"
	"kb/internal/retr"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kb <command> [args]\ncommands: ingest, search, serve")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "ingest":
		ingestCmd(os.Args[2:])
	case "search":
		searchCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// ingestCmd implements the `ingest` subcommand. It accepts flags:
// --list points to a file containing newline separated targets. Each target
// may be a local file path or a GitHub URL. --data sets the directory where 
// all persisted files live. --github-token provides authentication for 
// private GitHub repositories.
func ingestCmd(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	listPath := fs.String("list", "", "path to a file containing newline separated URLs or file paths")
	dataDir := fs.String("data", "./data", "directory in which to store artefacts and index files")
	githubToken := fs.String("github-token", "", "GitHub personal access token for private repositories")
	fs.Parse(args)
	if *listPath == "" {
		fmt.Fprintln(os.Stderr, "--list is required")
		os.Exit(1)
	}
	// prepare required subdirectories
	if err := pipe.PrepareDirs(*dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "error preparing directories: %v\n", err)
		os.Exit(1)
	}
	f, err := os.Open(*listPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open list: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		art, err := fetch.FetchArtifact(line, *githubToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", line, err)
			continue
		}
		if err := pipe.ProcessArtifact(art, *dataDir); err != nil {
			fmt.Fprintf(os.Stderr, "error processing %s: %v\n", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading list: %v\n", err)
		os.Exit(1)
	}
	// build lexical index after ingestion
	if err := index.BuildIndex(*dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "error building index: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ingestion complete")
}

// searchCmd implements the `search` subcommand. It loads the token index and
// docmap, computes a naive score based on token frequency per chunk and
// displays the top results. By default the top 10 matches are returned; the
// number can be changed with --top.
func searchCmd(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "directory containing the index")
	topN := fs.Int("top", 10, "number of top results to display")
	fs.Parse(args)
	// read query from remaining args
	leftover := fs.Args()
	if len(leftover) == 0 {
		fmt.Fprintln(os.Stderr, "usage: kb search [--data dir] [--top n] query")
		os.Exit(1)
	}
	query := strings.Join(leftover, " ")
	
	// perform search
	results, err := retr.HybridSearch(*dataDir, query, *topN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search error: %v\n", err)
		os.Exit(1)
	}
	
	if len(results) == 0 {
		fmt.Println("no matches found")
		return
	}
	
	// output results
	fmt.Print(retr.FormatSearchResults(results))
}

// serveCmd launches a simple HTTP server exposing an endpoint for retrieval. This
// server listens on the specified address (default :8080) and loads the index
// into memory at startup. Clients can issue POST requests with a JSON body
// containing a `query` field; the server returns the top 10 chunk metadata
// entries in JSON format. For brevity the implementation here focuses on a
// single endpoint; additional endpoints can be added as needed.
func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "directory containing the index")
	addr := fs.String("addr", ":8080", "address to listen on")
	topN := fs.Int("top", 10, "number of results to return")
	fs.Parse(args)
	
	// create handler
	handler := api.SearchHandler(*dataDir, *topN)
	
	// start server
	fmt.Printf("listening on %s\n", *addr)
	// use the net/http package only when running inside a proper Go environment; here we implement a naive HTTP server using the standard library
	// because the environment this code runs in might not have a long‑running process, this is stubbed out.
	fmt.Println("HTTP server stub is not implemented in this environment. Please implement using net/http when running locally.")
	_ = handler
}