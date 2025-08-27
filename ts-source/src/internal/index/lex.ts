import { readdir, readFile, mkdir, writeFile } from 'fs/promises';
import { join } from 'path';
import { DocMeta, DocEntry } from '../types.js';

export async function buildIndex(dataDir: string): Promise<void> {
  const chunksDir = join(dataDir, 'chunks');
  const metaDir = join(dataDir, 'meta');
  const indexDir = join(dataDir, 'index');
  
  await mkdir(indexDir, { recursive: true });

  const tokenIndex: Record<string, string[]> = {};
  const docmap: Record<string, DocEntry> = {};

  const entries = await readdir(chunksDir, { withFileTypes: true });

  for (const entry of entries) {
    if (entry.isDirectory() || !entry.name.endsWith('.md')) {
      continue;
    }

    const chunkID = entry.name.slice(0, -3); // Remove .md extension
    const parts = chunkID.split('-');
    const sha = parts[0];

    const metaPath = join(metaDir, `${sha}.json`);
    const metaBuffer = await readFile(metaPath);
    const meta: DocMeta = JSON.parse(metaBuffer.toString('utf-8'));

    const docID = `${meta.source}:${sha}`;
    docmap[chunkID] = {
      docId: docID,
      title: meta.title,
      url: meta.url,
      source: meta.source,
      updatedAt: meta.updatedAt,
      breadcrumbs: meta.breadcrumbs
    };

    const chunkPath = join(chunksDir, entry.name);
    const data = await readFile(chunkPath, 'utf-8');

    const tokens = tokenize(data);
    const unique = new Set<string>();

    for (const token of tokens) {
      if (!unique.has(token)) {
        if (!tokenIndex[token]) {
          tokenIndex[token] = [];
        }
        tokenIndex[token].push(chunkID);
        unique.add(token);
      }
    }
  }

  const indexPath = join(indexDir, 'index.json');
  await writeJSON(indexPath, tokenIndex);

  const docmapPath = join(indexDir, 'docmap.json');
  await writeJSON(docmapPath, docmap);
}

export function tokenize(s: string): string[] {
  // Replace punctuation and symbols with spaces
  const cleaned = s.replace(/[\p{P}\p{S}]/gu, ' ');
  const fields = cleaned.split(/\s+/).filter(f => f.length > 0);
  
  const tokens: string[] = [];
  for (const field of fields) {
    const lower = field.toLowerCase();
    if (lower.length >= 2) {
      tokens.push(lower);
    }
  }
  
  return tokens;
}

export async function writeJSON(path: string, value: any): Promise<void> {
  const json = JSON.stringify(value, null, 2);
  await writeFile(path, json, 'utf-8');
}