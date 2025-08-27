import { createHash } from 'crypto';
import { writeFile, mkdir, appendFile, stat } from 'fs/promises';
import { join } from 'path';
import { Artifact, DocMeta, ManifestRow } from '../types.js';
import { chunkMarkdown } from './chunk.js';

export async function processArtifact(art: Artifact, dataDir: string): Promise<void> {
  const hash = createHash('sha256');
  hash.update(art.markdown);
  const sha = hash.digest('hex');
  const docID = `${art.source}:${sha}`;

  const blobPath = join(dataDir, 'blobs', sha);
  
  try {
    await stat(blobPath);
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      await writeFile(blobPath, art.markdown, { encoding: 'utf-8' });
    } else {
      throw error;
    }
  }

  const meta: DocMeta = {
    title: art.title,
    url: art.canonicalUrl,
    updatedAt: art.updatedAt.toISOString(),
    source: art.source,
    breadcrumbs: art.breadcrumbs
  };

  const metaPath = join(dataDir, 'meta', `${sha}.json`);
  await writeFile(metaPath, JSON.stringify(meta, null, 2), { encoding: 'utf-8' });

  const manifest: ManifestRow = {
    artifactId: docID,
    canonicalUrl: art.canonicalUrl,
    contentSha: sha,
    updatedAt: art.updatedAt.toISOString(),
    source: art.source,
    title: art.title,
    tags: []
  };

  const manifestPath = join(dataDir, 'manifest.jsonl');
  const manifestLine = JSON.stringify(manifest) + '\n';
  
  try {
    await appendFile(manifestPath, manifestLine, { encoding: 'utf-8' });
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      await writeFile(manifestPath, manifestLine, { encoding: 'utf-8' });
    } else {
      throw error;
    }
  }

  const chunks = chunkMarkdown(art.markdown, art.title, art.breadcrumbs);
  
  for (let i = 0; i < chunks.length; i++) {
    const chunk = chunks[i];
    const chunkID = `${sha}-${i}`;
    chunk.id = chunkID;
    chunk.docId = docID;

    const chunkPath = join(dataDir, 'chunks', `${chunkID}.md`);
    await writeFile(chunkPath, chunk.text, { encoding: 'utf-8' });
  }
}

export async function prepareDirs(root: string): Promise<void> {
  const dirs = [
    join(root, 'blobs'),
    join(root, 'chunks'),
    join(root, 'meta'),
    join(root, 'index')
  ];

  for (const dir of dirs) {
    await mkdir(dir, { recursive: true });
  }
}