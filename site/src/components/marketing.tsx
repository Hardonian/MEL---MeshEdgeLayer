import Link from 'next/link';
import type { ReactNode } from 'react';
import { MEL_GITHUB_REPO, melGithubFile, repoBlob } from '@/lib/repo';

const DOCS_HUB = melGithubFile('docs/README.md');

type NavLink = { readonly href: string; readonly label: string; readonly external?: boolean };

export const NAV_LINKS: readonly NavLink[] = [
  { href: '/', label: 'Home' },
  { href: '/quickstart', label: 'Quick Start' },
  { href: '/guide', label: 'Guide' },
  { href: '/architecture', label: 'Architecture' },
  { href: '/trust', label: 'Trust' },
  { href: '/help', label: 'Help' },
  { href: '/faq', label: 'FAQ' },
  { href: '/contribute', label: 'Contribute' },
  { href: '/acknowledgements', label: 'Credits' },
  { href: DOCS_HUB, label: 'Docs', external: true },
] as const;

export function SiteShell({ children }: { children: ReactNode }) {
  return (
    <div className="shell">
      <a href="#main-content" className="skipLink">
        Skip to main content
      </a>
      <header className="header">
        <div className="container headerInner">
          <Link href="/" className="wordmark" aria-label="MEL home">
            MEL
          </Link>
          <nav aria-label="Primary">
            <ul className="navList">
              {NAV_LINKS.map((link) => (
                <li key={link.href}>
                  {link.external ? (
                    <a href={link.href} rel="noreferrer" target="_blank">
                      {link.label}
                      <span className="srOnly"> (opens in new tab)</span>
                    </a>
                  ) : (
                    <Link href={link.href}>{link.label}</Link>
                  )}
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </header>
      <main id="main-content" className="container main" tabIndex={-1}>
        {children}
      </main>
      <footer className="footer">
        <div className="container footerGrid">
          <p>
            MEL is a local-first operator OS for incident intelligence and trusted control in mesh and edge
            environments. Licensed under GPLv3 — see{' '}
            <a href={repoBlob('LICENSE')} rel="noreferrer" target="_blank">
              LICENSE
            </a>{' '}
            in the repo.
          </p>
          <div className="footerLinks">
            <Link href="/quickstart">Quick Start</Link>
            <Link href="/guide">Guide</Link>
            <Link href="/architecture">Architecture</Link>
            <Link href="/trust">Trust</Link>
            <Link href="/help">Help</Link>
            <Link href="/faq">FAQ</Link>
            <Link href="/contribute">Contribute</Link>
            <Link href="/acknowledgements">Credits</Link>
            <a href={DOCS_HUB} rel="noreferrer" target="_blank">
              Docs
            </a>
            <a href={MEL_GITHUB_REPO} rel="noreferrer" target="_blank">
              GitHub
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

export function PageHeader({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <header className="pageHeader">
      <h1>{title}</h1>
      <p>{subtitle}</p>
    </header>
  );
}

export function Section({
  title,
  children,
  id,
  description,
}: {
  title: string;
  children: ReactNode;
  id?: string;
  description?: ReactNode;
}) {
  const headingId = id ? `${id}-heading` : undefined;
  return (
    <section className="section" id={id} aria-labelledby={headingId}>
      <h2 id={headingId}>{title}</h2>
      {description ? <p className="sectionLead">{description}</p> : null}
      <div className="sectionBody">{children}</div>
    </section>
  );
}

export function TerminalBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="terminal" role="group" aria-label={title}>
      <p className="terminalTitle">{title}</p>
      <pre>{children}</pre>
    </div>
  );
}

export function PrincipleList({ items }: { items: { name: string; detail: string }[] }) {
  return (
    <ul className="principleList">
      {items.map((item) => (
        <li key={item.name}>
          <h3>{item.name}</h3>
          <p>{item.detail}</p>
        </li>
      ))}
    </ul>
  );
}
