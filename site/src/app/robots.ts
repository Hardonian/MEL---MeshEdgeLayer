import type { MetadataRoute } from 'next';
import { getSiteOriginString } from '@/lib/site-url';

import { getSiteCanonicalOrigin } from '@/lib/siteOrigin';

export default function robots(): MetadataRoute.Robots {
  const base = getSiteCanonicalOrigin();
  return {
    rules: {
      userAgent: '*',
      allow: '/',
    },
    sitemap: `${base}/sitemap.xml`,
  };
}
