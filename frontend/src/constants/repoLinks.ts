/**
 * Canonical public repository URLs for in-app help and adoption surfaces.
 * Keep aligned with docs/repo-os/canonical-github.md and site/src/lib/repo.ts.
 */
export const MEL_GITHUB_REPO = 'https://github.com/Hardonian/MEL-MeshEdgeLayer' as const

export const melGithubFile = (path: string) => `${MEL_GITHUB_REPO}/blob/main/${path.replace(/^\//, '')}`
