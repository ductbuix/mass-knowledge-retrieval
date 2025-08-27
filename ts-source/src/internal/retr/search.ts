import { readFile } from 'fs/promises';
import { join } from 'path';
import { DocEntry, SearchResult } from '../types.js';
import { tokenize } from '../index/lex.js';
import { readPreview } from '../index/reader.js';

export async function hybridSearch(dataDir: string, query: string, topN: number): Promise<SearchResult[]> {
  const indexPath = join(dataDir, 'index', 'index.json');
  const docmapPath = join(dataDir, 'index', 'docmap.json');

  const indexBuffer = await readFile(indexPath, 'utf-8');
  const tokenIndex: Record<string, string[]> = JSON.parse(indexBuffer);

  const docmapBuffer = await readFile(docmapPath, 'utf-8');
  const docmap: Record<string, DocEntry> = JSON.parse(docmapBuffer);

  const tokens = tokenize(query);
  if (tokens.length === 0) {
    throw new Error('No valid search terms provided');
  }

  const counts: Record<string, number> = {};
  
  for (const token of tokens) {
    const chunkIds = tokenIndex[token];
    if (chunkIds) {
      for (const chunkId of chunkIds) {
        counts[chunkId] = (counts[chunkId] || 0) + 1;
      }
    }
  }

  if (Object.keys(counts).length === 0) {
    return [];
  }

  const entries = Object.entries(counts)
    .map(([key, count]) => ({ key, count }))
    .sort((a, b) => {
      if (a.count === b.count) {
        return a.key.localeCompare(b.key);
      }
      return b.count - a.count;
    });

  const limitedEntries = entries.slice(0, topN);

  const results: SearchResult[] = [];
  
  for (const { key: chunkId, count } of limitedEntries) {
    const meta = docmap[chunkId];
    if (!meta) {
      continue;
    }

    const chunkPath = join(dataDir, 'chunks', `${chunkId}.md`);
    const snippet = await readPreview(chunkPath, 300);

    results.push({
      chunkId,
      score: count,
      preview: snippet,
      metadata: meta
    });
  }

  return results;
}

export function formatSearchResults(results: SearchResult[]): string {
  let output = '';

  for (const result of results) {
    output += `[${result.metadata.source}] ${result.metadata.title} — ${result.metadata.url}\n`;
    
    if (result.metadata.breadcrumbs && result.metadata.breadcrumbs.length > 0) {
      output += `${result.metadata.breadcrumbs.join(' > ')}\n`;
    }
    
    output += `${result.preview}\n\n`;
  }

  return output;
}