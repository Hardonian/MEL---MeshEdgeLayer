import type { Metadata } from 'next';
import { IBM_Plex_Mono, Inter } from 'next/font/google';
import './globals.css';
import { SiteShell } from '@/components/marketing';

const inter = Inter({ subsets: ['latin'], variable: '--font-sans', display: 'swap' });
const ibmMono = IBM_Plex_Mono({
  weight: ['400', '600', '700'],
  subsets: ['latin'],
  variable: '--font-mono',
  display: 'swap',
});

export const metadata: Metadata = {
  metadataBase: new URL('https://mel.local'),
  title: {
    default: 'MEL — MeshEdgeLayer',
    template: '%s | MEL',
  },
  description:
    'MEL is a local-first incident-intelligence and trusted-control operating system for mesh and edge operators.',
  icons: {
    icon: '/favicon.svg',
  },
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
    <html lang="en" className={`${inter.variable} ${ibmMono.variable}`}>
      <body>
        <SiteShell>{children}</SiteShell>
      </body>
    </html>
  );
}
