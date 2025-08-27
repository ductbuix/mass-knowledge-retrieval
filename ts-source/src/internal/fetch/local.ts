import { readFile } from 'fs/promises';
import { basename, extname, resolve } from 'path';
import { Artifact } from '../types.js';

export async function fetchLocalArtifact(target: string): Promise<Artifact> {
  const path = target.startsWith('file://') ? target.substring(7) : target;
  const resolvedPath = resolve(path);
  
  try {
    const buffer = await readFile(resolvedPath);
    const text = buffer.toString('utf-8');
    const name = basename(resolvedPath);
    const ext = extname(resolvedPath);
    const md = formatFileContent(text, ext);

    return {
      source: 'local',
      canonicalUrl: `file://${resolvedPath}`,
      title: name,
      updatedAt: new Date(),
      markdown: md,
      breadcrumbs: []
    };
  } catch (error) {
    throw new Error(`Failed to read file ${path}: ${error instanceof Error ? error.message : String(error)}`);
  }
}

function formatFileContent(text: string, ext: string): string {
  const codeExts = new Set([
    '.go', '.js', '.ts', '.py', '.java', '.rb', '.rs', '.c', '.cpp', '.cs',
    '.sh', '.sql', '.yaml', '.yml', '.json', '.tf', '.html', '.css', '.scss',
    '.php', '.r', '.scala', '.kt', '.swift'
  ]);

  ext = ext.toLowerCase();

  if (ext && codeExts.has(ext)) {
    const lang = ext.substring(1);
    return `\`\`\`${lang}\n${text.trimEnd()}\n\`\`\`\n`;
  }

  return text;
}