package pipe

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kb/internal"
)

// ProcessArtifact persists a single artefact to the data directory. It writes
// the normalized markdown, splits it into chunks, writes each chunk as a
// separate file and records metadata in manifest.jsonl and meta/<sha>.json.
// The docID is constructed from the artefact's source and the content hash.
func ProcessArtifact(art internal.Artifact, dataDir string) error {
	// compute hash of normalized markdown
	sum := sha256.Sum256([]byte(art.Markdown))
	sha := fmt.Sprintf("%x", sum)
	docID := art.Source + ":" + sha
	// write blob file if it doesn't already exist
	blobPath := filepath.Join(dataDir, "blobs", sha)
	if _, err := os.Stat(blobPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(blobPath, []byte(art.Markdown), 0644); err != nil {
			return fmt.Errorf("write blob: %w", err)
		}
	}
	// write meta file
	meta := internal.DocMeta{
		Title:       art.Title,
		URL:         art.CanonicalURL,
		UpdatedAt:   art.UpdatedAt.UTC().Format(time.RFC3339),
		Source:      art.Source,
		Breadcrumbs: art.Breadcrumbs,
	}
	metaPath := filepath.Join(dataDir, "meta", sha+".json")
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	// write manifest entry
	manif := internal.ManifestRow{
		ArtifactID:   docID,
		CanonicalURL: art.CanonicalURL,
		ContentSha:   sha,
		UpdatedAt:    art.UpdatedAt.UTC().Format(time.RFC3339),
		Source:       art.Source,
		Title:        art.Title,
		Tags:         []string{},
	}
	manifestPath := filepath.Join(dataDir, "manifest.jsonl")
	mf, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open manifest: %w", err)
	}
	defer mf.Close()
	b, _ := json.Marshal(manif)
	if _, err := mf.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	// chunk the markdown
	chunks := ChunkMarkdown(art.Markdown, art.Title, art.Breadcrumbs)
	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s-%d", sha, i)
		c.ID = chunkID
		c.DocID = docID
		// write chunk text
		chunkPath := filepath.Join(dataDir, "chunks", chunkID+".md")
		if err := os.WriteFile(chunkPath, []byte(c.Text), 0644); err != nil {
			return fmt.Errorf("write chunk %s: %w", chunkID, err)
		}
	}
	return nil
}

// PrepareDirs ensures that all required subdirectories exist under the given
// data directory. It creates blobs, chunks, meta and index directories if
// necessary.
func PrepareDirs(root string) error {
	dirs := []string{
		filepath.Join(root, "blobs"),
		filepath.Join(root, "chunks"),
		filepath.Join(root, "meta"),
		filepath.Join(root, "index"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}