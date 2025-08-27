import { Artifact } from '../types.js';
import { fetchGitHubArtifact, isGitHubURL } from './github.js';
import { fetchLocalArtifact } from './local.js';

export async function fetchArtifact(target: string, githubToken: string = ''): Promise<Artifact> {
  if (isGitHubURL(target)) {
    return fetchGitHubArtifact(target, githubToken);
  }

  if (target.startsWith('http://') || target.startsWith('https://')) {
    throw new Error('Only GitHub URLs are supported for remote fetching');
  }

  return fetchLocalArtifact(target);
}

export * from './github.js';
export * from './local.js';