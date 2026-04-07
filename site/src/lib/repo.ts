/**
 * Canonical GitHub URLs for the public Next.js site.
 * Single source of truth for wording: docs/repo-os/canonical-github.md
 */
export const MEL_GITHUB_REPO = 'https://github.com/Hardonian/MEL-MeshEdgeLayer' as const;
export const REPO_URL = MEL_GITHUB_REPO;
export const REPO_ISSUES_URL = `${MEL_GITHUB_REPO}/issues` as const;

export function melGithubFile(path: string): string {
  return `${MEL_GITHUB_REPO}/blob/main/${path.replace(/^\//, '')}`;
}

export const repoBlob = melGithubFile;
