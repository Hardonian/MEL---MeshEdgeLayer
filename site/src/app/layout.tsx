import type { Metadata } from 'next';
import './globals.css';
import { SiteShell } from '@/components/marketing';

export const metadata: Metadata = {
  metadataBase: new URL('https://mel.local'),
  title: {
    default: 'MEL — MeshEdgeLayer',
    template: '%s | MEL',
  },
  description:
    'MEL is a local-first incident-intelligence and trusted-control operating system for mesh and edge operators.',
  openGraph: {
    title: 'MEL — MeshEdgeLayer',
    description:
      'Truthful incident intelligence and trusted control for mesh operations. Evidence-first, local-first, privacy-first.',
    type: 'website',
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
