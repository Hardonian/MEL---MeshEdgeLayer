/**
 * Canonical origin for absolute URLs (metadataBase, OG, sitemap, robots).
 * Override at build time: SITE_CANONICAL_ORIGIN=https://example.com npm run build
 *
 * Default matches the usual GitHub Pages URL for this repository; change via env for a custom domain.
 */
export function getSiteCanonicalOrigin(): string {
  const raw = process.env.SITE_CANONICAL_ORIGIN?.trim();
  if (raw) {
    return raw.replace(/\/$/, '');
  }
  return 'https://hardonian.github.io/MEL-MeshEdgeLayer';
}
