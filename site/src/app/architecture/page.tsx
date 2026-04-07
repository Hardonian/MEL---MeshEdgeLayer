import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { MEL_GITHUB_REPO, melGithubFile } from '@/lib/repo';

export default function ArchitecturePage() {
  return (
    <>
      <PageHeader
        title="Architecture"
        subtitle="Where truth lives, how evidence flows, and what MEL deliberately does not pretend to know."
      />

      <Section title="Truth hierarchy (canonical)">
        <ol className="orderedList">
          <li>Typed deterministic runtime evidence (ingest records, state transitions, audit events).</li>
          <li>Deterministic calculators and bounded heuristics with explicit inputs.</li>
          <li>Assistive inference — labeled non-canonical when present.</li>
          <li>Narrative and UI copy — must not contradict the layers above.</li>
        </ol>
        <p>
          Full detail:{' '}
          <a href={melGithubFile('docs/architecture/overview.md')} rel="noreferrer" target="_blank">
            Architecture overview (docs)
          </a>
          ,{' '}
          <a href={melGithubFile('docs/product/ARCHITECTURE_TRUTH.md')} rel="noreferrer" target="_blank">
            Architecture truth (product)
          </a>
          .
        </p>
      </Section>

      <Section title="Component map (high level)">
        <ul>
          <li>
            <strong>CLI / daemon</strong> — <code>cmd/mel</code>: init, doctor, serve, demo, and operator commands.
          </li>
          <li>
            <strong>Embedded operator UI</strong> — React + Vite build copied into <code>internal/web/assets/</code> and served by the binary.
          </li>
          <li>
            <strong>Ingest workers</strong> — serial/TCP/MQTT paths persist evidence; unsupported paths stay explicitly labeled.
          </li>
          <li>
            <strong>SQLite store</strong> — local operational database; bounded claims require understanding freshness and scope.
          </li>
        </ul>
        <p>
          Folder-level map for contributors:{' '}
          <a href={melGithubFile('docs/contributor/ARCHITECTURE_MAP.md')} rel="noreferrer" target="_blank">
            ARCHITECTURE_MAP.md
          </a>
          .
        </p>
      </Section>

      <Section title="Transport and control boundaries">
        <p>
          MEL observes what ingest and configuration make visible. It does <strong>not</strong> implement mesh RF routing as a product
          feature, and it does <strong>not</strong> collapse degraded or unknown posture into “all green.”
        </p>
        <p>
          Control paths separate <strong>submission, approval, dispatch, execution result, and audit</strong> — see{' '}
          <Link href="/trust">Trust &amp; privacy</Link> and the docs on control-plane trust.
        </p>
      </Section>

      <Section title="Read next">
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
            <Link href="/quickstart">Quick start</Link> · <Link href="/help">Help / orientation</Link>
          </li>
        </ul>
      </Section>

      <TerminalBlock title="Clone and open the deep docs">
{`git clone ${MEL_GITHUB_REPO}.git
cd MEL-MeshEdgeLayer && ls docs/architecture/`}
      </TerminalBlock>
    </>
  );
}
