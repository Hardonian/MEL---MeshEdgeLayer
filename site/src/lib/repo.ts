/**
 * Canonical GitHub URLs for the public Next.js site.
 * Keep aligned with frontend/src/constants/repoLinks.ts and the primary remote.
 */
export const MEL_GITHUB_REPO = 'https://github.com/Hardonian/MEL-MeshEdgeLayer' as const;

export function melGithubFile(path: string): string {
  return `${MEL_GITHUB_REPO}/blob/main/${path.replace(/^\//, '')}`;
}
