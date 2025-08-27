package api

import (
	"encoding/json"
	"io"

	"kb/internal"
	"kb/internal/retr"
)

// SearchHandler handles search requests and returns JSON results.
// This is a simple implementation that can be extended for HTTP serving.
func SearchHandler(dataDir string, topN int) func(io.Writer, string) {
	return func(w io.Writer, query string) {
		results, err := retr.HybridSearch(dataDir, query, topN)
		if err != nil {
			// In a real HTTP handler, this would be an error response
			results = []retr.SearchResult{}
		}
		
		// Convert to DocEntry format for compatibility
		docEntries := make([]internal.DocEntry, 0, len(results))
		for _, result := range results {
			docEntries = append(docEntries, result.Meta)
		}
		
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(docEntries)
	}
}