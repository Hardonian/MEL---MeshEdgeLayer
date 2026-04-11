import type { Metadata } from 'next';
import Link from 'next/link';
import { PageHeader, Section, TerminalBlock, TruthTierList, ArchGrid } from '@/components/marketing';
import { MEL_GITHUB_REPO, melGithubFile } from '@/lib/repo';

export const metadata: Metadata = {
  title: 'Architecture',
  description: 'Where truth lives, how evidence flows, and what MEL deliberately does not pretend to know.',
};

const truthTiers = [
  {
    num: '①',
    label: 'Runtime evidence',
    badge: 'canonical',
    detail: 'Typed deterministic records: ingest records, state transitions, audit events. Always wins on conflict.',
  },
  {
    num: '②',
    label: 'Bounded calculators',
    badge: 'deterministic',
    detail: 'Heuristics and derived values with explicit inputs and documented uncertainty bounds.',
  },
  {
    num: '③',
    label: 'Assistive inference',
    badge: 'labeled',
    detail: 'AI assistance labeled non-canonical when present. Never overrides deterministic layers.',
  },
  {
    num: '④',
    label: 'Narrative and UI copy',
    badge: 'follows',
    detail: 'Must not contradict layers above. Narrative follows evidence — not the reverse.',
  },
];

const archBlocks = [
  {
    tag: 'entry point',
    name: 'CLI / Daemon',
    detail: 'cmd/mel — init, doctor, serve, demo, operator commands. Single binary entry point.',
  },
  {
    tag: 'persistence',
    name: 'SQLite Store',
    detail: 'Local operational database. All evidence, incidents, and audit events. Freshness is explicit.',
  },
  {
    tag: 'transport',
    name: 'Ingest Workers',
    detail: 'Serial/TCP/MQTT paths. Unsupported paths (BLE, HTTP ingest) stay labeled — no implied support.',
  },
  {
    tag: 'api layer',
    name: 'Runtime API',
    detail: 'REST API serves embedded console and external clients. /healthz, /readyz, /api/v1/status.',
  },
  {
    tag: 'surface',
    name: 'Embedded Console',
    detail: 'React+Vite UI compiled into internal/web/assets/. Served by mel serve — not this site.',
  },
];

export default function ArchitecturePage() {
  return (
    <>
      <PageHeader
        kicker="system architecture"
        title="Architecture"
        subtitle="Where truth lives, how evidence flows, and what MEL deliberately does not pretend to know."
      />

      <Section
        title="Truth hierarchy"
        id="truth-hierarchy"
        kicker="canonical model"
        accent="green"
        description="MEL's evidence model is explicit about source authority. Lower layers never override higher ones."
      >
        <TruthTierList tiers={truthTiers} />
        <p style={{ marginTop: 'var(--space-md)' }}>
          Full detail:{' '}
          <a href={melGithubFile('docs/architecture/overview.md')} rel="noreferrer" target="_blank">
            Architecture overview (docs)
          </a>{' '}
          ·{' '}
          <a href={melGithubFile('docs/product/ARCHITECTURE_TRUTH.md')} rel="noreferrer" target="_blank">
            Architecture truth (product)
          </a>
        </p>
      </Section>

      <Section
        title="Component map"
        id="components"
        kicker="system"
        description="High-level runtime structure. Each layer has explicit boundaries and documented ingest constraints."
      >
        <ArchGrid blocks={archBlocks} />
        <p style={{ marginTop: 'var(--space-md)' }}>
          Folder-level map for contributors:{' '}
          <a href={melGithubFile('docs/contributor/ARCHITECTURE_MAP.md')} rel="noreferrer" target="_blank">
            ARCHITECTURE_MAP.md
          </a>
        </p>
      </Section>

      <Section title="Transport and control boundaries" id="boundaries" kicker="constraints">
        <p>
          MEL observes what ingest and configuration make visible. It does <strong>not</strong> implement mesh RF
          routing as a product feature, and it does <strong>not</strong> collapse degraded or unknown posture
          into &ldquo;all green.&rdquo;
        </p>
        <p>
          Control paths separate <strong>submission, approval, dispatch, execution result, and audit</strong> —
          each is a distinct state with attribution. See{' '}
          <Link href="/trust">Trust &amp; privacy</Link> and the docs on control-plane trust.
        </p>
      </Section>

      <Section title="Operational boundaries" id="op-bounds" kicker="non-goals" accent="blue">
        <ul>
          <li>MEL is not the RF routing stack — it persists evidence from what your ingest workers connect.</li>
          <li>MEL does not claim RF coverage, mesh delivery, or propagation proof.</li>
          <li>BLE ingest and HTTP ingest are unsupported — labels remain in UI and docs.</li>
          <li>Assistive AI output is non-canonical — deterministic layers always win conflicts.</li>
        </ul>
      </Section>

      <Section title="Read next" id="read-next">
        <ul>
          <li>
            <a href={melGithubFile('docs/architecture/transport-flow.md')} rel="noreferrer" target="_blank">
              Transport flow
            </a>
          </li>
          <li>
            <a href={melGithubFile('docs/architecture/OPERATIONAL_BOUNDARIES.md')} rel="noreferrer" target="_blank">
              Operational boundaries
            </a>
          </li>
          <li>
            <Link href="/quickstart">Quick start</Link> ·{' '}
            <Link href="/help">Help / orientation</Link> ·{' '}
            <Link href="/compare">Compare</Link>
          </li>
        </ul>
      </Section>

      <TerminalBlock title="shell — open deep docs">
{`git clone ${MEL_GITHUB_REPO}.git
cd MEL-MeshEdgeLayer && ls docs/architecture/`}
      </TerminalBlock>
    </>
  );
}
