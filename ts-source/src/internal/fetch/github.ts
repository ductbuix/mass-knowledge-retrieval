import { Artifact } from '../types.js';

export interface GitHubContent {
  name: string;
  path: string;
  type: string;
  size: number;
  content?: string;
  encoding?: string;
  download_url?: string;
}

export class GitHubClient {
  private token: string;
  private httpClient: { timeout: number };

  constructor(token: string = '') {
    this.token = token;
    this.httpClient = { timeout: 30000 };
  }

  private async makeAPIRequest(url: string): Promise<Response> {
    const headers: Record<string, string> = {
      'Accept': 'application/vnd.github.v3+json',
      'User-Agent': 'kb-knowledge-retrieval/1.0'
    };

    if (this.token) {
      headers['Authorization'] = `token ${this.token}`;
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.httpClient.timeout);

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers,
        signal: controller.signal
      });
      return response;
    } finally {
      clearTimeout(timeout);
    }
  }

  async getFileContent(owner: string, repo: string, path: string, ref: string = ''): Promise<GitHubContent> {
    let apiURL = `https://api.github.com/repos/${owner}/${repo}/contents/${path}`;
    if (ref) {
      apiURL += `?ref=${ref}`;
    }

    const response = await this.makeAPIRequest(apiURL);

    if (!response.ok) {
      throw new Error(`GitHub API returned status ${response.status}`);
    }

    const content = await response.json() as GitHubContent;
    return content;
  }

  async getDirectoryContents(owner: string, repo: string, path: string, ref: string = ''): Promise<GitHubContent[]> {
    let apiURL = `https://api.github.com/repos/${owner}/${repo}/contents/${path}`;
    if (ref) {
      apiURL += `?ref=${ref}`;
    }

    const response = await this.makeAPIRequest(apiURL);

    if (!response.ok) {
      throw new Error(`GitHub API returned status ${response.status}`);
    }

    const contents = await response.json() as GitHubContent[];
    return contents;
  }
}

export interface GitHubURLInfo {
  owner: string;
  repo: string;
  path: string;
  ref: string;
}

export function parseGitHubURL(gitURL: string): GitHubURLInfo {
  const url = new URL(gitURL);

  if (url.hostname !== 'github.com') {
    throw new Error('Not a GitHub URL');
  }

  const parts = url.pathname.split('/').filter(p => p.length > 0);
  if (parts.length < 2) {
    throw new Error('Invalid GitHub URL format');
  }

  const info: GitHubURLInfo = {
    owner: parts[0],
    repo: parts[1],
    path: '',
    ref: ''
  };

  if (parts.length >= 4 && parts[2] === 'blob') {
    info.ref = parts[3];
    if (parts.length > 4) {
      info.path = parts.slice(4).join('/');
    }
  } else if (parts.length >= 4 && parts[2] === 'tree') {
    info.ref = parts[3];
    if (parts.length > 4) {
      info.path = parts.slice(4).join('/');
    }
  } else if (parts.length > 2) {
    info.ref = 'main';
    info.path = parts.slice(2).join('/');
  }

  return info;
}

export function isGitHubURL(target: string): boolean {
  return target.includes('github.com');
}

export async function fetchGitHubArtifact(target: string, token: string = ''): Promise<Artifact> {
  const info = parseGitHubURL(target);
  const client = new GitHubClient(token);

  if (!info.path) {
    return fetchGitHubDirectory(client, info, target);
  }

  try {
    const content = await client.getFileContent(info.owner, info.repo, info.path, info.ref);

    if (content.type !== 'file') {
      return fetchGitHubDirectory(client, info, target);
    }

    let text = '';
    if (content.encoding === 'base64' && content.content) {
      text = Buffer.from(content.content, 'base64').toString('utf-8');
    } else if (content.content) {
      text = content.content;
    }

    const md = formatFileContent(text, getExtension(content.name));

    return {
      source: 'github',
      canonicalUrl: target,
      title: content.name,
      updatedAt: new Date(),
      markdown: md,
      breadcrumbs: [info.owner, info.repo, content.path]
    };
  } catch (error) {
    return fetchGitHubDirectory(client, info, target);
  }
}

async function fetchGitHubDirectory(client: GitHubClient, info: GitHubURLInfo, originalURL: string): Promise<Artifact> {
  const contents = await client.getDirectoryContents(info.owner, info.repo, info.path, info.ref);

  let allContent = '';
  let fileCount = 0;

  for (const item of contents) {
    if (item.type === 'file') {
      try {
        const fileContent = await client.getFileContent(info.owner, info.repo, item.path, info.ref);

        let text = '';
        if (fileContent.encoding === 'base64' && fileContent.content) {
          text = Buffer.from(fileContent.content, 'base64').toString('utf-8');
        } else if (fileContent.content) {
          text = fileContent.content;
        }

        if (fileCount > 0) {
          allContent += '\n\n---\n\n';
        }

        allContent += `# ${item.path}\n\n`;
        allContent += formatFileContent(text, getExtension(item.name));
        fileCount++;
      } catch (error) {
        console.warn(`Warning: failed to fetch ${item.path}: ${error}`);
      }
    }
  }

  if (fileCount === 0) {
    throw new Error('No files found in directory');
  }

  let title = `${info.owner}/${info.repo}`;
  if (info.path) {
    title += `/${info.path}`;
  }

  return {
    source: 'github',
    canonicalUrl: originalURL,
    title,
    updatedAt: new Date(),
    markdown: allContent,
    breadcrumbs: [info.owner, info.repo, info.path]
  };
}

function getExtension(filename: string): string {
  const lastDot = filename.lastIndexOf('.');
  return lastDot >= 0 ? filename.substring(lastDot) : '';
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