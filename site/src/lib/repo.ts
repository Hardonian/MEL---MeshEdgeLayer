/** Canonical upstream repo for doc and issue links from the public site. */
export const REPO_URL = 'https://github.com/mel-project/mel';
export const REPO_ISSUES_URL = `${REPO_URL}/issues`;
export const repoBlob = (pathFromRoot: string) => `${REPO_URL}/blob/main/${pathFromRoot.replace(/^\//, '')}`;
