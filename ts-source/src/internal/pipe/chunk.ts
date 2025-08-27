import { Chunk } from '../types.js';

interface Block {
  text: string;
  crumb: string[];
}

export function chunkMarkdown(md: string, title: string, breadcrumbs: string[]): Chunk[] {
  const lines = md.split('\n');
  const blocks: Block[] = [];
  let current: string[] = [];
  let currentBreadcrumb = [...breadcrumbs];
  
  const headingRegex = /^(#{1,6})\s+(.*)$/;
  let fenceCount = 0;

  for (const line of lines) {
    const headingMatch = headingRegex.exec(line);
    
    if (headingMatch) {
      if (current.length > 0) {
        const crumbCopy = [...currentBreadcrumb];
        blocks.push({
          text: current.join('\n'),
          crumb: crumbCopy
        });
        current = [];
      }
      
      currentBreadcrumb = [...breadcrumbs, headingMatch[2].trim()];
      current.push(line);
      continue;
    }

    if (line.startsWith('```')) {
      fenceCount = fenceCount % 2 === 0 ? fenceCount + 1 : fenceCount - 1;
    }

    current.push(line);
  }

  if (current.length > 0) {
    const crumbCopy = [...currentBreadcrumb];
    blocks.push({
      text: current.join('\n'),
      crumb: crumbCopy
    });
  }

  const chunks: Chunk[] = [];
  let buf: string[] = [];
  let bufBreadcrumb = [...breadcrumbs];
  let words = 0;
  const target = 1000;

  for (const block of blocks) {
    const w = wordCount(block.text);
    
    if (words + w > target) {
      if (buf.length > 0) {
        chunks.push({
          id: '', // Will be set later in processArtifact
          docId: '', // Will be set later in processArtifact
          text: buf.join('\n'),
          breadcrumbs: bufBreadcrumb
        });
      }
      
      buf = [block.text];
      bufBreadcrumb = [...block.crumb];
      words = w;
    } else {
      if (buf.length === 0) {
        bufBreadcrumb = [...block.crumb];
      }
      buf.push(block.text);
      words += w;
    }
  }

  if (buf.length > 0) {
    chunks.push({
      id: '', // Will be set later in processArtifact
      docId: '', // Will be set later in processArtifact
      text: buf.join('\n'),
      breadcrumbs: bufBreadcrumb
    });
  }

  if (chunks.length === 0) {
    chunks.push({
      id: '', // Will be set later in processArtifact
      docId: '', // Will be set later in processArtifact
      text: md,
      breadcrumbs: [...breadcrumbs]
    });
  }

  return chunks;
}

function wordCount(s: string): number {
  return s.trim().split(/\s+/).filter(word => word.length > 0).length;
}