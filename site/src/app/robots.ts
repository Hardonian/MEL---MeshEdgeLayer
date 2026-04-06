import type { MetadataRoute } from 'next';
import { getSiteOriginString } from '@/lib/site-url';

export default function robots(): MetadataRoute.Robots {
  const origin = getSiteOriginString();
  return {
    rules: {
      userAgent: '*',
      allow: '/',
    },
    sitemap: `${origin}/sitemap.xml`,
  };
}
