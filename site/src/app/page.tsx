import Link from 'next/link';
import { PrincipleList, Section, TerminalBlock } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';
import { MEL_BOOTSTRAP_COMMANDS } from '@/lib/orientation';

const principles = [
  { name: 'Evidence before narrative', detail: 'Runtime records and audit events are canonical; commentary follows evidence.' },
  { name: 'Explicit degraded states', detail: 'Partial, stale, imported, and unknown states stay visible instead of being painted healthy.' },
  { name: 'Trusted control lifecycle', detail: 'Submission, approval, dispatch, execution, and audit are separate states with attribution.' },
  { name: 'Local-first posture', detail: 'MEL remains useful without mandatory cloud dependencies or hidden telemetry defaults.' },
];

export default function HomePage() {
  return (
    <>
      <section className="hero" aria-labelledby="hero-heading">
        <p className="kicker">MeshEdgeLayer</p>
        <h1 id="hero-heading" className="heroTitle">
          MEL is an evidence-first operator OS for incidents and trusted control under degraded conditions.
        </h1>
        <p className="heroLead">
          Built for mesh/edge operators who need honest runtime truth: keep evidence and action history attributable, and keep
          partial, stale, and unknown posture visible.
        </p>
        <ul className="heroMeta" aria-label="Product positioning">
          <li>For field operators and maintainers who own the stack</li>
          <li>Self-hosted by default; optional integrations stay explicit</li>
          <li>CLI and embedded UI; this site is orientation only</li>
        </ul>
        <div className="ctaRow">
          <span className="ctaLabel">Start here</span>
          <Link href="/quickstart" className="btn primary">
            Quick start
          </Link>
          <Link href="/help" className="btn">
            Help / orientation
          </Link>
          <Link href="/contribute" className="btn">
            Contribute
          </Link>
          <a href={melGithubFile('docs/README.md')} className="btn" rel="noreferrer" target="_blank">
            Documentation hub
            <span className="srOnly"> (opens in new tab)</span>
          </a>
        </div>
        <div className="ctaRow ctaRowSecondary">
          <span className="ctaLabel">Boundaries</span>
          <a href={melGithubFile('docs/ops/support-matrix.md')} className="btn" rel="noreferrer" target="_blank">
            Support matrix
            <span className="srOnly"> (opens in new tab)</span>
          </a>
          <a href={melGithubFile('docs/ops/limitations.md')} className="btn" rel="noreferrer" target="_blank">
            Known limitations
            <span className="srOnly"> (opens in new tab)</span>
          </a>
          <Link href="/trust" className="btn">
            Trust & privacy
          </Link>
          <Link href="/guide" className="btn">Site vs console vs docs</Link>
        </div>
      </section>

      <Section title="Launch funnel" id="funnel" description="One path from first read to real operation.">
        <ol>
          <li>
            <strong>What MEL is:</strong> this page + <Link href="/help">Help</Link>.
          </li>
          <li>
            <strong>Try MEL:</strong> <Link href="/quickstart">Quick start</Link> with deterministic commands.
          </li>
          <li>
            <strong>Understand boundaries:</strong>{' '}
            <a href={melGithubFile('docs/ops/support-matrix.md')} rel="noreferrer" target="_blank">
              support matrix
            </a>{' '}
            and{' '}
            <a href={melGithubFile('docs/community/claims-vs-reality.md')} rel="noreferrer" target="_blank">
              claims vs reality
            </a>
            .
          </li>
          <li>
            <strong>Contribute:</strong> <Link href="/contribute">Contribute</Link>.
          </li>
          <li>
            <strong>Operate for real:</strong>{' '}
            <a href={melGithubFile('docs/ops/OPERATIONS_RUNBOOK.md')} rel="noreferrer" target="_blank">
              operations runbook
            </a>
            .
          </li>
        </ol>
      </Section>

      <Section
        title="What MEL is"
        id="what-is"
        description="A concise contract before you invest time in build and deploy."
      >
        <p>
          MEL is built for incident workflows where maps and green badges are not enough: you need persisted evidence, separable
          control states, and language that admits gaps.
        </p>
        <div className="grid" style={{ marginTop: 'var(--space-md)' }}>
          <article className="card">
            <h3>Operator console</h3>
            <p>
              The product UI ships inside <code>./bin/mel serve</code> (React + Vite from <code>frontend/</code>), not on this
              marketing surface.
            </p>
          </article>
          <article className="card">
            <h3>CLI-first runtime</h3>
            <p>
              Init, doctor, serve, and demo paths are the spine. No mandatory hosted control plane for base viability.
            </p>
          </article>
          <article className="card">
            <h3>Evidence and audit</h3>
            <p>Ingest records, timelines, proofpack-style exports, and control-path attribution stay first-class.</p>
          </article>
        </div>
      </Section>

      <Section title="What MEL is not" id="what-not">
        <ul>
          <li>Not a mesh RF routing stack or proof of coverage unless your persisted evidence supports it.</li>
          <li>Not a generic dashboard skin that hides missing data behind “healthy.”</li>
          <li>Not canonical truth from assistive inference — deterministic layers win when they disagree.</li>
        </ul>
      </Section>

      <Section title="Supported ingest (truth matrix)" id="transport">
        <p>
          Claim only what your configuration and ingest workers actually persist. Unsupported paths stay labeled unsupported in UI
          and docs.
        </p>
        <table className="matrix" style={{ marginTop: 'var(--space-md)' }}>
          <caption className="srOnly">MEL ingest support posture</caption>
          <thead>
            <tr>
              <th scope="col">Path</th>
              <th scope="col">Posture</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <th scope="row">Direct serial / TCP</th>
              <td>Supported — bounded by what the worker connected and stored.</td>
            </tr>
            <tr>
              <th scope="row">MQTT</th>
              <td>Supported — broker disconnects and partial ingest must remain visible.</td>
            </tr>
            <tr>
              <th scope="row">BLE / HTTP ingest</th>
              <td>Unsupported — do not imply partial product support.</td>
            </tr>
          </tbody>
        </table>
      </Section>

      <Section title="Why MEL exists" id="why">
        <p>
          Incident response fails when stale data looks live, when intent and execution blur, and when exports omit context. MEL
          pushes the opposite: explicit freshness and lifecycle separation, with wording tied to evidence.
        </p>
      </Section>

      <Section title="Console emphasis (when you run the binary)" id="console">
        <div className="grid">
          <article className="card">
            <h3>Incidents and queue</h3>
            <p>Workflow centered on incident state, rationale, and replayable evidence context.</p>
          </article>
          <article className="card">
            <h3>Transport diagnostics</h3>
            <p>Readiness, stale signals, dead letters, and doctor output surfaced with bounded semantics.</p>
          </article>
          <article className="card">
            <h3>Proofpacks and bundles</h3>
            <p>Exports and handoff-oriented flows for triage and review — not narrative substitutes.</p>
          </article>
          <article className="card">
            <h3>Local runtime</h3>
            <p>Single binary serves API and embedded UI; optional services remain optional.</p>
          </article>
        </div>
      </Section>

      <Section title="Operator-truth doctrine" id="doctrine">
        <PrincipleList items={principles} />
      </Section>

      <Section title="Clone and run (summary)" id="cli-summary">
        <p>
          Full commands and caveats: <Link href="/quickstart">Quick start</Link>. This block is the minimal spine.
        </p>
        <TerminalBlock title="Terminal">
{MEL_BOOTSTRAP_COMMANDS}
        </TerminalBlock>
        <p className="callout" role="note">
          This public site does not connect to your MEL instance. Operator truth lives in the process you serve locally (or on
          your host), not here.
        </p>
      </Section>
    </>
  );
}
