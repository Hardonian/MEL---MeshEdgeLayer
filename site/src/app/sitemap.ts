import type { MetadataRoute } from 'next';
import { getSiteOriginString } from '@/lib/site-url';

import { getSiteCanonicalOrigin } from '@/lib/siteOrigin';

export default function sitemap(): MetadataRoute.Sitemap {
  const base = getSiteCanonicalOrigin();
  const now = new Date();
  const routes = [
    '',
    '/quickstart',
    '/architecture',
    '/trust',
    '/help',
    '/faq',
    '/contribute',
    '/acknowledgements',
  ];

  return routes.map((route) => ({
    url: route === '' ? `${base}/` : `${base}${route}`,
    lastModified: now,
    changeFrequency: 'weekly',
    priority: route === '' ? 1 : 0.8,
  }));
}
