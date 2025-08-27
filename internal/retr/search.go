package retr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kb/internal"
	"kb/internal/index"
)

// SearchResult contains a search result with metadata and snippet
type SearchResult struct {
	ChunkID  string
	Meta     internal.DocEntry
	Snippet  string
	Score    int
}

// HybridSearch performs a naive keyword search using the lexical index.
// It loads the index and docmap, computes scores based on token frequency
// per chunk and returns the top results.
func HybridSearch(dataDir, query string, topN int) ([]SearchResult, error) {
	// load index and docmap
	idxPath := filepath.Join(dataDir, "index", "index.json")
	idxFile, err := os.Open(idxPath)
	if err != nil {
		return nil, fmt.Errorf("could not open index at %s: %w", idxPath, err)
	}
	defer idxFile.Close()
	var idx map[string][]string
	if err := json.NewDecoder(idxFile).Decode(&idx); err != nil {
		return nil, fmt.Errorf("invalid index file: %w", err)
	}
	
	docmapPath := filepath.Join(dataDir, "index", "docmap.json")
	docFile, err := os.Open(docmapPath)
	if err != nil {
		return nil, fmt.Errorf("could not open docmap at %s: %w", docmapPath, err)
	}
	defer docFile.Close()
	var docmap map[string]internal.DocEntry
	if err := json.NewDecoder(docFile).Decode(&docmap); err != nil {
		return nil, fmt.Errorf("invalid docmap file: %w", err)
	}
	
	// tokenize query
	tokens := index.Tokenize(query)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no valid search terms provided")
	}
	
	// accumulate counts per chunk
	counts := make(map[string]int)
	for _, tok := range tokens {
		if list, ok := idx[tok]; ok {
			for _, cid := range list {
				counts[cid]++
			}
		}
	}
	
	if len(counts) == 0 {
		return []SearchResult{}, nil
	}
	
	// sort by count descending
	type kv struct {
		Key   string
		Count int
	}
	var arr []kv
	for k, c := range counts {
		arr = append(arr, kv{Key: k, Count: c})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].Count == arr[j].Count {
			return arr[i].Key < arr[j].Key
		}
		return arr[i].Count > arr[j].Count
	})
	
	if topN > len(arr) {
		topN = len(arr)
	}
	
	// build results with snippets
	results := make([]SearchResult, 0, topN)
	for i := 0; i < topN; i++ {
		cid := arr[i].Key
		meta, ok := docmap[cid]
		if !ok {
			// skip unknown entries
			continue
		}
		
		// read snippet from chunk file
		chunkPath := filepath.Join(dataDir, "chunks", cid+".md")
		snippet := index.ReadPreview(chunkPath, 300)
		
		results = append(results, SearchResult{
			ChunkID: cid,
			Meta:    meta,
			Snippet: snippet,
			Score:   arr[i].Count,
		})
	}
	
	return results, nil
}

// FormatSearchResults formats search results for console output
func FormatSearchResults(results []SearchResult) string {
	var output strings.Builder
	
	for _, result := range results {
		output.WriteString(fmt.Sprintf("[%s] %s — %s\n", 
			result.Meta.Source, result.Meta.Title, result.Meta.URL))
		
		if len(result.Meta.Breadcrumbs) > 0 {
			output.WriteString(fmt.Sprintf("%s\n", 
				strings.Join(result.Meta.Breadcrumbs, " > ")))
		}
		
		output.WriteString(fmt.Sprintf("%s\n\n", result.Snippet))
	}
	
	return output.String()
}