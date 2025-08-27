package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kb/internal"
)

// BuildIndex scans the chunks directory and constructs two JSON files: a
// lexical index mapping tokens to chunk identifiers and a docmap mapping
// chunk identifiers to metadata. The index is built from scratch each time
// ingest completes; incremental updates could be implemented by merging
// existing data.
func BuildIndex(dataDir string) error {
	chunksDir := filepath.Join(dataDir, "chunks")
	metaDir := filepath.Join(dataDir, "meta")
	indexDir := filepath.Join(dataDir, "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}
	tokenIndex := make(map[string][]string)
	docmap := make(map[string]internal.DocEntry)
	// iterate over chunk files
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		chunkID := strings.TrimSuffix(entry.Name(), ".md")
		// derive sha prefix to locate meta file
		parts := strings.SplitN(chunkID, "-", 2)
		sha := parts[0]
		metaPath := filepath.Join(metaDir, sha+".json")
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			return fmt.Errorf("missing meta for chunk %s: %w", chunkID, err)
		}
		var meta internal.DocMeta
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			return fmt.Errorf("invalid meta for %s: %w", chunkID, err)
		}
		// build doc entry
		docID := meta.Source + ":" + sha
		docmap[chunkID] = internal.DocEntry{
			DocID:       docID,
			Title:       meta.Title,
			URL:         meta.URL,
			Source:      meta.Source,
			UpdatedAt:   meta.UpdatedAt,
			Breadcrumbs: meta.Breadcrumbs,
		}
		// open chunk text
		chunkPath := filepath.Join(chunksDir, entry.Name())
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", chunkPath, err)
		}
		// tokenize and populate index
		text := string(data)
		tokens := Tokenize(text)
		unique := make(map[string]bool)
		for _, tok := range tokens {
			if !unique[tok] {
				tokenIndex[tok] = append(tokenIndex[tok], chunkID)
				unique[tok] = true
			}
		}
	}
	// persist index.json
	idxPath := filepath.Join(indexDir, "index.json")
	if err := WriteJSON(idxPath, tokenIndex); err != nil {
		return err
	}
	// persist docmap.json
	docmapPath := filepath.Join(indexDir, "docmap.json")
	if err := WriteJSON(docmapPath, docmap); err != nil {
		return err
	}
	return nil
}

// Tokenize splits a string into lower‑cased tokens. It removes punctuation and
// short tokens (less than 2 characters) to reduce noise in the index.
func Tokenize(s string) []string {
	// replace punctuation with spaces
	re := regexp.MustCompile(`[\p{P}\p{S}]`)
	cleaned := re.ReplaceAllString(s, " ")
	fields := strings.Fields(cleaned)
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ToLower(f)
		if len(f) < 2 {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}

// WriteJSON writes the given value as indented JSON to the specified file.
func WriteJSON(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}