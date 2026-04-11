import Link from 'next/link';
import type { ReactNode } from 'react';
import { repoBlob, MEL_GITHUB_REPO, melGithubFile } from '@/lib/repo';
import { SiteNav } from './SiteNav';

const DOCS_HUB = melGithubFile('docs/README.md');

type NavLink = { readonly href: string; readonly label: string; readonly external?: boolean };

export const NAV_LINKS: readonly NavLink[] = [
  { href: '/', label: 'Home' },
  { href: '/quickstart', label: 'Quick Start' },
  { href: '/docs', label: 'Docs' },
  { href: '/compare', label: 'Compare' },
  { href: '/guide', label: 'Guide' },
  { href: '/architecture', label: 'Architecture' },
  { href: '/trust', label: 'Trust' },
  { href: '/help', label: 'Help' },
  { href: '/contribute', label: 'Contribute' },
  { href: DOCS_HUB, label: 'Repo Docs', external: true },
] as const;

/* ─────────────────────────────────────────────────
   Site Shell
───────────────────────────────────────────────── */

export function SiteShell({ children }: { children: ReactNode }) {
  return (
    <div className="shell">
      <a href="#main-content" className="skipLink">
        Skip to main content
      </a>
      <header className="header">
        <div className="container headerInner">
          <Link href="/" className="wordmark" aria-label="MEL home">
            <span className="wordmarkDot" aria-hidden="true" />
            MEL
          </Link>
          <SiteNav links={NAV_LINKS} />
        </div>
      </header>
      <main id="main-content" className="container main" tabIndex={-1}>
        {children}
      </main>
      <footer className="footer">
        <div className="container footerInner">
          <div>
            <p className="footerWordmark">
              <span className="footerWordmarkDot" aria-hidden="true" />
              MEL
            </p>
            <p className="footerDesc">
              Local-first incident intelligence and trusted control for mesh and edge operators.
              Evidence-first, explicit degraded states, CLI and embedded console. GPLv3.
            </p>
            <p className="footerMeta" style={{ marginTop: '0.75rem' }}>
              <a href={repoBlob('LICENSE')} rel="noreferrer" target="_blank">
                License
              </a>{' '}
              ·{' '}
              <a href={MEL_GITHUB_REPO} rel="noreferrer" target="_blank">
                GitHub
              </a>
            </p>
          </div>
          <div className="footerLinks">
            <Link href="/quickstart">Quick Start</Link>
            <Link href="/docs">Docs</Link>
            <Link href="/compare">Compare</Link>
            <Link href="/guide">Guide</Link>
            <Link href="/architecture">Architecture</Link>
            <Link href="/trust">Trust</Link>
            <Link href="/help">Help</Link>
            <Link href="/faq">FAQ</Link>
            <Link href="/contribute">Contribute</Link>
            <Link href="/acknowledgements">Credits</Link>
            <a href={DOCS_HUB} rel="noreferrer" target="_blank">
              Repo Docs
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

/* ─────────────────────────────────────────────────
   Page Header
───────────────────────────────────────────────── */

export function PageHeader({
  title,
  subtitle,
  kicker,
}: {
  title: string;
  subtitle: string;
  kicker?: string;
}) {
  return (
    <header className="pageHeader">
      {kicker ? <p className="pageHeaderKicker">{kicker}</p> : null}
      <h1>{title}</h1>
      <p>{subtitle}</p>
    </header>
  );
}

/* ─────────────────────────────────────────────────
   Section
───────────────────────────────────────────────── */

export function Section({
  title,
  children,
  id,
  description,
  kicker,
  accent,
}: {
  title: string;
  children: ReactNode;
  id?: string;
  description?: ReactNode;
  kicker?: string;
  accent?: 'green' | 'blue';
}) {
  const headingId = id ? `${id}-heading` : undefined;
  const accentClass = accent === 'blue' ? ' accent2Top' : accent === 'green' ? ' accentTop' : '';
  return (
    <section className={`section${accentClass}`} id={id} aria-labelledby={headingId}>
      <div className="sectionHeader">
        {kicker ? (
          <p className={`sectionKicker${accent === 'blue' ? ' blue' : ''}`} aria-hidden="true">
            {kicker}
          </p>
        ) : null}
        <h2 id={headingId}>{title}</h2>
        {description ? <p className="sectionLead">{description}</p> : null}
      </div>
      <div className="sectionBody">{children}</div>
    </section>
  );
}

/* ─────────────────────────────────────────────────
   Terminal Block
───────────────────────────────────────────────── */

export function TerminalBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="terminal" role="group" aria-label={title}>
      <div className="panelChrome">
        <div className="panelChromeDots" aria-hidden="true">
          <span className="panelChromeDot panelChromeDotR" />
          <span className="panelChromeDot panelChromeDotY" />
          <span className="panelChromeDot panelChromeDotG" />
        </div>
        <span className="panelChromeTitle">{title}</span>
      </div>
      <pre>{children}</pre>
    </div>
  );
}

/* ─────────────────────────────────────────────────
   Principle List
───────────────────────────────────────────────── */

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

/* ─────────────────────────────────────────────────
   Operator Card
───────────────────────────────────────────────── */

export function OpCard({
  kicker,
  title,
  body,
  icon,
}: {
  kicker?: string;
  title: string;
  body: string;
  icon?: ReactNode;
}) {
  return (
    <article className="opCard">
      {icon ? <div className="opCardIcon" aria-hidden="true">{icon}</div> : null}
      {kicker ? <p className="opCardKicker">{kicker}</p> : null}
      <h3>{title}</h3>
      <p>{body}</p>
    </article>
  );
}

/* ─────────────────────────────────────────────────
   Signal State Bar
───────────────────────────────────────────────── */

type SignalStateEntry = {
  key: string;
  label: string;
  desc: string;
  colorClass: string;
  dotClass: string;
};

const SIGNAL_STATES: SignalStateEntry[] = [
  {
    key: 'observed',
    label: 'OBSERVED',
    desc: 'Deterministic evidence from connected ingest',
    colorClass: 'postureLive',
    dotClass: 'sigLive',
  },
  {
    key: 'inferred',
    label: 'INFERRED',
    desc: 'Bounded heuristic — labeled non-canonical',
    colorClass: 'postureInferred',
    dotClass: 'sigInferred',
  },
  {
    key: 'stale',
    label: 'STALE',
    desc: 'Evidence present but freshness expired',
    colorClass: 'postureStale',
    dotClass: 'sigStale',
  },
  {
    key: 'degraded',
    label: 'DEGRADED',
    desc: 'Partial, unknown, or conflicting posture',
    colorClass: 'postureDegraded',
    dotClass: 'sigDegraded',
  },
];

export function SignalStateBar() {
  return (
    <div className="signalBar" role="list" aria-label="MEL signal state semantics">
      {SIGNAL_STATES.map((s) => (
        <div key={s.key} className="signalBarState" role="listitem">
          <p className="signalBarLabel">
            <span className={`sigDot ${s.dotClass}`} aria-hidden="true" />
            {s.label}
          </p>
          <p className="signalBarDesc">{s.desc}</p>
        </div>
      ))}
    </div>
  );
}

/* ─────────────────────────────────────────────────
   Truth Tier List
───────────────────────────────────────────────── */

type TruthTier = { num: string; label: string; badge: string; detail: string };

export function TruthTierList({ tiers }: { tiers: TruthTier[] }) {
  return (
    <ol className="truthTierList">
      {tiers.map((t) => (
        <li key={t.num} className="truthTierItem">
          <span className="truthTierNum">{t.num}</span>
          <div className="truthTierContent">
            <strong>
              {t.label}
              <span className="truthTierBadge">{t.badge}</span>
            </strong>
            <p>{t.detail}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}

/* ─────────────────────────────────────────────────
   Architecture Grid
───────────────────────────────────────────────── */

type ArchBlock = { tag: string; name: string; detail: string };

export function ArchGrid({ blocks }: { blocks: ArchBlock[] }) {
  return (
    <div className="archGrid" role="list" aria-label="System component map">
      {blocks.map((b) => (
        <div key={b.name} className="archBlock" role="listitem">
          <p className="archBlockTag">{b.tag}</p>
          <p className="archBlockName">{b.name}</p>
          <p className="archBlockDetail">{b.detail}</p>
        </div>
      ))}
    </div>
  );
}

/* ─────────────────────────────────────────────────
   Flow Steps
───────────────────────────────────────────────── */

type FlowStepItem = { title: string; body: ReactNode };

export function FlowSteps({ steps }: { steps: FlowStepItem[] }) {
  return (
    <ol className="flowSteps">
      {steps.map((step, i) => (
        <li key={i} className="flowStep">
          <span className="flowStepNum" aria-hidden="true">{i + 1}</span>
          <div className="flowStepContent">
            <h3>{step.title}</h3>
            <p>{step.body}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}

/* ─────────────────────────────────────────────────
   Hero Mesh Background SVG (decorative)
───────────────────────────────────────────────── */

export function HeroMeshBg() {
  // Static node positions in a loose mesh layout
  const nodes: [number, number][] = [
    [35, 25],  [110, 10], [195, 38], [275, 15], [350, 48], [415, 22], [465, 40],
    [18, 90],  [75, 75],  [155, 98], [235, 78], [310, 65], [385, 92], [445, 75],
    [48, 155], [130, 138],[210, 162],[285, 148],[360, 135],[430, 152],
    [22, 218], [98, 202], [178, 222],[255, 208],[332, 218],[405, 202],[462, 222],
  ];
  const edges: [number, number][] = [
    [0,1],[1,2],[2,3],[3,4],[4,5],[5,6],
    [7,8],[8,9],[9,10],[10,11],[11,12],[12,13],
    [14,15],[15,16],[16,17],[17,18],[18,19],
    [20,21],[21,22],[22,23],[23,24],[24,25],[25,26],
    [0,7],[1,8],[2,9],[3,10],[4,11],[5,12],
    [7,14],[8,15],[9,16],[10,17],[11,18],
    [14,20],[15,21],[16,22],[17,23],[18,24],
    [1,9],[3,11],[9,17],[11,16],[16,23],
  ];
  return (
    <svg
      className="heroMeshBg"
      viewBox="0 0 480 245"
      aria-hidden="true"
      role="presentation"
      preserveAspectRatio="xMaxYMid slice"
    >
      {edges.map(([a, b], i) => (
        <line
          key={i}
          x1={nodes[a][0]} y1={nodes[a][1]}
          x2={nodes[b][0]} y2={nodes[b][1]}
          stroke="currentColor"
          strokeWidth="0.7"
        />
      ))}
      {nodes.map(([x, y], i) => (
        <circle key={i} cx={x} cy={y} r={i % 5 === 0 ? 2.5 : 1.8} fill="currentColor" />
      ))}
    </svg>
  );
}

/* ─────────────────────────────────────────────────
   Operator Signal Panel (Hero aside)
───────────────────────────────────────────────── */

export function OperatorSignalPanel() {
  return (
    <aside className="operatorPanel" aria-label="MEL signal state reference (illustrative)">
      <div className="panelChrome">
        <div className="panelChromeDots" aria-hidden="true">
          <span className="panelChromeDot panelChromeDotR" />
          <span className="panelChromeDot panelChromeDotY" />
          <span className="panelChromeDot panelChromeDotG" />
        </div>
        <span className="panelChromeTitle">mel · posture model</span>
      </div>
      <div className="panelBody">
        <div>
          <p className="panelSectionLabel">Ingest posture</p>
          <div className="signalRow">
            <span className="sigDot sigLive" aria-hidden="true" />
            <span className="sigLabel">OBSERVED</span>
            <span className="sigDesc">deterministic evidence</span>
          </div>
          <div className="signalRow">
            <span className="sigDot sigInferred" aria-hidden="true" />
            <span className="sigLabel">INFERRED</span>
            <span className="sigDesc">labeled non-canonical</span>
          </div>
          <div className="signalRow">
            <span className="sigDot sigStale" aria-hidden="true" />
            <span className="sigLabel">STALE</span>
            <span className="sigDesc">freshness expired</span>
          </div>
          <div className="signalRow">
            <span className="sigDot sigDegraded" aria-hidden="true" />
            <span className="sigLabel">DEGRADED</span>
            <span className="sigDesc">partial or unknown</span>
          </div>
        </div>
        <hr className="panelDivider" />
        <div>
          <p className="panelSectionLabel">Truth origin</p>
          <div className="truthRow truthPrimary">
            <span className="truthNum">①</span>
            <span>runtime evidence</span>
          </div>
          <div className="truthRow">
            <span className="truthNum">②</span>
            <span>bounded calculators</span>
          </div>
          <div className="truthRow truthDim">
            <span className="truthNum">③</span>
            <span>labeled inference</span>
          </div>
          <div className="truthRow truthDim" style={{ opacity: 0.4 }}>
            <span className="truthNum">④</span>
            <span>narrative (follows)</span>
          </div>
        </div>
        <hr className="panelDivider" />
        <div>
          <p className="panelSectionLabel">Deployment</p>
          <div className="truthRow truthPrimary">
            <span className="truthNum" style={{ color: 'var(--accent-2)' }}>✓</span>
            <span>local-first binary</span>
          </div>
          <div className="truthRow truthDim">
            <span className="truthNum" style={{ color: 'var(--muted-2)' }}>–</span>
            <span>no mandatory cloud</span>
          </div>
        </div>
      </div>
    </aside>
  );
}
