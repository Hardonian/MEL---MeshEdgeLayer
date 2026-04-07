import type { Metadata } from 'next';
import './globals.css';
import { SiteShell } from '@/components/marketing';
import { getSiteCanonicalOrigin } from '@/lib/siteOrigin';

const siteOrigin = getSiteCanonicalOrigin();

export const metadata: Metadata = {
  metadataBase: new URL(siteOrigin),
  title: {
    default: 'MEL — MeshEdgeLayer',
    template: '%s | MEL',
  },
  description:
    'Local-first incident intelligence and trusted control for mesh and edge operators — evidence-first, explicit degraded states, CLI and embedded console.',
  icons: {
    icon: '/favicon.svg',
  },
  openGraph: {
    title: 'MEL — MeshEdgeLayer',
    description:
      'Truthful incident intelligence and trusted control for mesh operations. Evidence-first, local-first, explicit degraded states.',
    type: 'website',
    url: siteOrigin,
    siteName: 'MEL',
    locale: 'en',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MEL — MeshEdgeLayer',
    description:
      'Truthful incident intelligence and trusted control for mesh operations. Evidence-first, local-first, privacy-first.',
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <SiteShell>{children}</SiteShell>
      </body>
    </html>
  );
}
