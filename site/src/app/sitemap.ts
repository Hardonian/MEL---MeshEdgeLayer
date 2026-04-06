import type { MetadataRoute } from 'next';
import { getSiteOriginString } from '@/lib/site-url';

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();
  const origin = getSiteOriginString();
  const routes = ['', '/quickstart', '/help', '/contribute', '/acknowledgements'];

  return routes.map((route) => ({
    url: `${origin}${route || '/'}`,
    lastModified: now,
    changeFrequency: 'weekly',
    priority: route === '' ? 1 : 0.8,
  }));
}
