# Canonical public GitHub URLs

Use these URLs in documentation, issue templates, and orientation surfaces so links stay consistent with the primary public remote.

| Use | URL |
| --- | --- |
| Repository home | `https://github.com/Hardonian/MEL-MeshEdgeLayer` |
| Releases | `https://github.com/Hardonian/MEL-MeshEdgeLayer/releases` |
| File on `main` | `https://github.com/Hardonian/MEL-MeshEdgeLayer/blob/main/<path>` |

**In code:** import `MEL_GITHUB_REPO` and `melGithubFile()` from `frontend/src/constants/repoLinks.ts` (embedded UI) or mirror the same strings in `site/src/lib/repo.ts` (public Next.js site).

**Go module path:** the module in `go.mod` remains `github.com/mel-project/mel` for import compatibility; that is not the same as the browser-facing Git remote above.
