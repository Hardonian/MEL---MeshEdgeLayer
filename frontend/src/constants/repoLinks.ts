/**
 * Canonical public repository URLs for in-app help and adoption surfaces.
 * Keep aligned with the primary GitHub remote for this fork (see README).
 */
export const MEL_GITHUB_REPO = 'https://github.com/Hardonian/MEL-MeshEdgeLayer' as const

export const melGithubFile = (path: string) => `${MEL_GITHUB_REPO}/blob/main/${path.replace(/^\//, '')}`
