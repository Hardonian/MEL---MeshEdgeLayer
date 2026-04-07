import Link from 'next/link';
import type { ReactNode } from 'react';
import { REPO_ISSUES_URL, REPO_URL, repoBlob } from '@/lib/repo';

import { MEL_GITHUB_REPO, melGithubFile } from '@/lib/repo';

const DOCS_HUB = melGithubFile('docs/README.md');

type NavLink = { readonly href: string; readonly label: string; readonly external?: boolean };

export const NAV_LINKS: readonly NavLink[] = [
  { href: '/', label: 'Home' },
  { href: '/quickstart', label: 'Quick Start' },
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
      <main className="container main">{children}</main>
      <footer className="footer">
        <div className="container footerGrid">
          <p>
            MEL is a local-first operator OS for incident intelligence and trusted control in mesh and edge
            environments. Licensed under GPLv3 — see{' '}
            <a href={repoBlob('LICENSE')}>LICENSE</a> in the repo.
          </p>
          <div className="footerLinks">
            <Link href="/quickstart">Quick Start</Link>
            <Link href="/architecture">Architecture</Link>
            <Link href="/trust">Trust</Link>
            <Link href="/help">Help</Link>
            <Link href="/faq">FAQ</Link>
            <Link href="/contribute">Contribute</Link>
            <Link href="/acknowledgements">Credits</Link>
            <a href={DOCS_HUB} rel="noreferrer" target="_blank">
              Docs
            </a>
            <a href={MEL_GITHUB_REPO}>GitHub</a>
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

export function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="section">
      <h2>{title}</h2>
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
