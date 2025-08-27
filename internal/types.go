package internal

import "time"

// DocMeta holds metadata about an artefact. This information is stored in
// meta/<sha>.json and is later copied into the docmap for each chunk.
type DocMeta struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	UpdatedAt   string   `json:"updatedAt"`
	Source      string   `json:"source"`
	Breadcrumbs []string `json:"breadcrumbs,omitempty"`
}

// ManifestRow describes a document for the manifest.jsonl file. It is
// deliberately minimal – only enough information to identify the artefact and
// record when it was last updated.
type ManifestRow struct {
	ArtifactID   string   `json:"artifactId"`
	CanonicalURL string   `json:"canonicalUrl"`
	ContentSha   string   `json:"contentSha"`
	UpdatedAt    string   `json:"updatedAt"`
	Source       string   `json:"source"`
	Title        string   `json:"title"`
	Tags         []string `json:"tags"`
}

// DocEntry is written to index/docmap.json for each chunk. It ties a chunk
// identifier back to its parent document and includes the basic fields we
// expect clients to display. Additional fields can be added later.
type DocEntry struct {
	DocID       string   `json:"docId"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Source      string   `json:"source"`
	UpdatedAt   string   `json:"updatedAt"`
	Breadcrumbs []string `json:"breadcrumbs,omitempty"`
}

// Chunk represents a portion of a document. It is not persisted directly as
// JSON; instead, the text itself is written as chunks/<sha>-<n>.md and the
// docmap contains the corresponding metadata. This struct is used only
// internally during ingestion.
type Chunk struct {
	ID          string
	DocID       string
	Text        string
	Breadcrumbs []string
}

// Artifact represents a fetched document ready for normalization and chunking.
// It is returned by fetchArtifact and consumed by processArtifact.
type Artifact struct {
	Source       string
	CanonicalURL string
	Title        string
	UpdatedAt    time.Time
	Markdown     string
	Breadcrumbs  []string
}