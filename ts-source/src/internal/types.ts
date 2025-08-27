export interface DocMeta {
  title: string;
  url: string;
  updatedAt: string;
  source: string;
  breadcrumbs?: string[];
}

export interface ManifestRow {
  artifactId: string;
  canonicalUrl: string;
  contentSha: string;
  updatedAt: string;
  source: string;
  title: string;
  tags: string[];
}

export interface DocEntry {
  docId: string;
  title: string;
  url: string;
  source: string;
  updatedAt: string;
  breadcrumbs?: string[];
}

export interface Chunk {
  id: string;
  docId: string;
  text: string;
  breadcrumbs: string[];
}

export interface Artifact {
  source: string;
  canonicalUrl: string;
  title: string;
  updatedAt: Date;
  markdown: string;
  breadcrumbs: string[];
}

export interface SearchResult {
  chunkId: string;
  score: number;
  preview: string;
  metadata: DocEntry;
}