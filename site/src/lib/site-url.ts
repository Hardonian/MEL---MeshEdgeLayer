/**
 * Canonical absolute URLs for metadata, sitemap, and robots.
 * Set NEXT_PUBLIC_SITE_URL at build time to the deployed origin (e.g. https://example.org).
 * When unset, defaults to local dev — never invent a production hostname.
 */
export function getSiteOrigin(): URL {
  const raw = process.env.NEXT_PUBLIC_SITE_URL?.trim();
  if (raw) {
    try {
      return new URL(raw.endsWith('/') ? raw.slice(0, -1) : raw);
    } catch {
      // fall through
    }
  }
  return new URL('http://localhost:3000');
}

export function getSiteOriginString(): string {
  return getSiteOrigin().origin;
}
