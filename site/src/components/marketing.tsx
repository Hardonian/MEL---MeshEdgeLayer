import Link from 'next/link';
import type { ReactNode } from 'react';

export const NAV_LINKS = [
  { href: '/', label: 'Home' },
  { href: '/quickstart', label: 'Quick Start' },
  { href: '/help', label: 'Help' },
  { href: '/contribute', label: 'Contribute' },
  { href: '/acknowledgements', label: 'Dependencies' },
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
                  <Link href={link.href}>{link.label}</Link>
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
            environments.
          </p>
          <div className="footerLinks">
            <Link href="/quickstart">Quick Start</Link>
            <Link href="/help">Help</Link>
            <Link href="/contribute">Contribute</Link>
            <Link href="/acknowledgements">Dependencies</Link>
            <a href="https://github.com/mel-project/mel">GitHub</a>
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
