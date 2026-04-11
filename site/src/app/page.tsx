import Link from 'next/link';
import {
  PrincipleList,
  Section,
  TerminalBlock,
  OpCard,
  SignalStateBar,
  TruthTierList,
  ArchGrid,
  FlowSteps,
  HeroMeshBg,
  OperatorSignalPanel,
} from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';
import { MEL_BOOTSTRAP_COMMANDS } from '@/lib/orientation';

const principles = [
  {
    name: 'Evidence before narrative',
    detail: 'Runtime records and audit events are canonical. Commentary follows evidence — never the reverse.',
  },
  {
    name: 'Explicit degraded states',
    detail: 'Partial, stale, imported, and unknown states stay visible instead of being collapsed to "healthy."',
  },
  {
    name: 'Trusted control lifecycle',
    detail: 'Submission, approval, dispatch, execution, and audit are separate states with attribution at each step.',
  },
  {
    name: 'Local-first posture',
    detail: 'MEL remains useful without mandatory cloud dependencies. No hidden telemetry defaults.',
  },
];

const truthTiers = [
  {
    num: '①',
    label: 'Runtime evidence',
    badge: 'canonical',
    detail: 'Typed deterministic records from ingest, state transitions, and audit events. Always wins conflicts.',
  },
  {
    num: '②',
    label: 'Bounded calculators',
    badge: 'deterministic',
    detail: 'Heuristics and calculators with explicit, documented inputs and bounded uncertainty.',
  },
  {
    num: '③',
    label: 'Assistive inference',
    badge: 'labeled',
    detail: 'AI or heuristic assistance — always labeled non-canonical when present. Never overrides ① or ②.',
  },
  {
    num: '④',
    label: 'Narrative and UI copy',
    badge: 'follows',
    detail: 'Must not contradict layers above. MEL enforces this at the product level.',
  },
];

const archBlocks = [
  {
    tag: 'entry point',
    name: 'CLI / Daemon',
    detail: 'cmd/mel — init, doctor, serve, demo, operator commands.',
  },
  {
    tag: 'persistence',
    name: 'SQLite Store',
    detail: 'Local operational database. Bounded claims require freshness awareness.',
  },
  {
    tag: 'transport',
    name: 'Ingest Workers',
    detail: 'Serial/TCP/MQTT paths persist evidence. Unsupported paths stay labeled.',
  },
  {
    tag: 'surface',
    name: 'Embedded Console',
    detail: 'React+Vite UI embedded in the binary. Served by mel serve, not this site.',
  },
];

export default function HomePage() {
  return (
    <>
      {/* ── Hero ─────────────────────────────────────────── */}
      <section className="hero" aria-labelledby="hero-heading">
        <HeroMeshBg />
        <div className="heroLayout">
          <div className="heroContent">
            <p className="kicker">MeshEdgeLayer · local-first · evidence-first</p>
            <h1 id="hero-heading" className="heroTitle">
              Incident intelligence for operators who need{' '}
              <em>honest runtime truth.</em>
            </h1>
            <p className="heroLead">
              MEL is a self-hostable operator OS for mesh and edge environments: persisted evidence, attributable control, and explicit degraded states — without collapse to &ldquo;all green.&rdquo;
            </p>
            <ul className="heroMeta" aria-label="Product positioning">
              <li>For field operators who own their stack</li>
              <li>Self-hosted binary; no mandatory cloud plane</li>
              <li>CLI and embedded console — this site is orientation only</li>
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
              <Link href="/trust" className="btn">Trust & privacy</Link>
              <Link href="/guide" className="btn">Site vs console vs docs</Link>
            </div>
          </div>
          <div className="heroAside">
            <OperatorSignalPanel />
          </div>
        </div>
      </section>

      {/* ── Signal State Semantics ────────────────────────── */}
      <Section
        title="Signal state semantics"
        id="signal-states"
        kicker="truth language"
        accent="green"
        description="MEL exposes these states across incident, transport, and node surfaces — never hidden behind a generic health indicator."
      >
        <SignalStateBar />
        <p className="callout" role="note" style={{ marginTop: 'var(--space-md)' }}>
          States are derived from ingest evidence and configuration — not from UI defaults or assumed connectivity.
          Unsupported paths remain labeled unsupported.
        </p>
      </Section>

      {/* ── What MEL Is ───────────────────────────────────── */}
      <Section
        title="What MEL is"
        id="what-is"
        kicker="product contract"
        description="A concise commitment before you invest time in build and deploy."
      >
        <div className="grid3">
          <OpCard
            kicker="surface"
            title="Operator console"
            body="The product UI ships inside ./bin/mel serve (React + Vite from frontend/), not on this marketing surface."
            icon={<ConsoleIcon />}
          />
          <OpCard
            kicker="runtime"
            title="CLI-first binary"
            body="Init, doctor, serve, and demo paths are the spine. No mandatory hosted control plane for base viability."
            icon={<CliIcon />}
          />
          <OpCard
            kicker="accountability"
            title="Evidence and audit"
            body="Ingest records, timelines, proofpack-style exports, and control-path attribution stay first-class."
            icon={<AuditIcon />}
          />
        </div>
      </Section>

      {/* ── What MEL Is Not ──────────────────────────────── */}
      <Section title="What MEL is not" id="what-not" kicker="non-goals">
        <ul>
          <li>Not a mesh RF routing stack or proof of coverage unless your persisted evidence supports it.</li>
          <li>Not a generic dashboard that hides missing data behind a green badge.</li>
          <li>Not canonical truth from assistive inference — deterministic layers win when they disagree.</li>
          <li>Not a hosted SaaS — your data stays in the process you control.</li>
        </ul>
      </Section>

      {/* ── Console Capabilities ─────────────────────────── */}
      <Section
        title="Console emphasis"
        id="console"
        kicker="when you run the binary"
        accent="blue"
        description="The embedded UI surfaces these categories. Depth lives in the running process, not this site."
      >
        <div className="grid">
          <OpCard
            kicker="workflow"
            title="Incidents and queue"
            body="State-centered incident workflow with rationale, replayable evidence context, and control lifecycle."
            icon={<IncidentIcon />}
          />
          <OpCard
            kicker="transport"
            title="Diagnostics"
            body="Readiness, stale signals, dead letters, and doctor output surfaced with bounded semantics."
            icon={<DiagIcon />}
          />
          <OpCard
            kicker="export"
            title="Proofpacks and bundles"
            body="Triage-oriented exports and handoff flows — evidence containers, not narrative substitutes."
            icon={<PackIcon />}
          />
          <OpCard
            kicker="posture"
            title="Local runtime"
            body="Single binary serves API and embedded UI. Optional services stay optional and labeled."
            icon={<LocalIcon />}
          />
        </div>
      </Section>

      {/* ── Truth Hierarchy ───────────────────────────────── */}
      <Section
        title="Truth hierarchy"
        id="truth"
        kicker="architecture"
        accent="green"
        description="MEL's evidence model is explicit about where truth comes from and which layer wins conflicts."
      >
        <TruthTierList tiers={truthTiers} />
        <p style={{ marginTop: 'var(--space-md)' }}>
          Full detail:{' '}
          <a href={melGithubFile('docs/architecture/overview.md')} rel="noreferrer" target="_blank">
            Architecture overview
          </a>{' '}
          ·{' '}
          <a href={melGithubFile('docs/product/ARCHITECTURE_TRUTH.md')} rel="noreferrer" target="_blank">
            Architecture truth
          </a>
        </p>
      </Section>

      {/* ── Component Map ─────────────────────────────────── */}
      <Section
        title="Component map"
        id="components"
        kicker="system"
        description="High-level runtime structure. Contributors: see ARCHITECTURE_MAP.md in docs."
      >
        <ArchGrid blocks={archBlocks} />
      </Section>

      {/* ── Ingest Truth Matrix ───────────────────────────── */}
      <Section title="Ingest posture matrix" id="transport" kicker="supported paths">
        <p>
          Claim only what configuration and ingest workers actually persist. Unsupported paths stay labeled
          unsupported in UI and docs.
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
        <p className="callout" role="note" style={{ marginTop: 'var(--space-md)' }}>
          Full matrix:{' '}
          <a href={melGithubFile('docs/ops/support-matrix.md')} rel="noreferrer" target="_blank">
            support-matrix.md
          </a>{' '}
          ·{' '}
          <a href={melGithubFile('docs/community/claims-vs-reality.md')} rel="noreferrer" target="_blank">
            claims vs reality
          </a>
        </p>
      </Section>

      {/* ── Operator Doctrine ─────────────────────────────── */}
      <Section title="Operator-truth doctrine" id="doctrine" kicker="principles" accent="green">
        <PrincipleList items={principles} />
      </Section>

      {/* ── Launch Flow ──────────────────────────────────── */}
      <Section
        title="Launch path"
        id="funnel"
        kicker="getting started"
        description="One path from first read to real operation."
      >
        <FlowSteps
          steps={[
            {
              title: 'Understand what MEL is',
              body: (
                <>
                  This page + <Link href="/help">Help / orientation</Link>. Read{' '}
                  <Link href="/guide">site vs console vs docs</Link> if the surfaces feel confusing.
                </>
              ),
            },
            {
              title: 'Try MEL locally',
              body: (
                <>
                  <Link href="/quickstart">Quick start</Link> with deterministic commands. Fixture mode available
                  — no radio hardware required.
                </>
              ),
            },
            {
              title: 'Understand operational boundaries',
              body: (
                <>
                  <a href={melGithubFile('docs/ops/support-matrix.md')} rel="noreferrer" target="_blank">
                    Support matrix
                  </a>{' '}
                  and{' '}
                  <a href={melGithubFile('docs/community/claims-vs-reality.md')} rel="noreferrer" target="_blank">
                    claims vs reality
                  </a>{' '}
                  before deploying.
                </>
              ),
            },
            {
              title: 'Contribute or extend',
              body: (
                <>
                  <Link href="/contribute">Contribution guide</Link> — issues, PRs, doc improvements, and
                  operator feedback welcome.
                </>
              ),
            },
            {
              title: 'Operate for real',
              body: (
                <>
                  <a href={melGithubFile('docs/ops/OPERATIONS_RUNBOOK.md')} rel="noreferrer" target="_blank">
                    Operations runbook
                  </a>{' '}
                  for field deployment and incident workflow.
                </>
              ),
            },
          ]}
        />
      </Section>

      {/* ── Bootstrap CLI ─────────────────────────────────── */}
      <Section
        title="Clone and run"
        id="cli-summary"
        kicker="bootstrap"
        description="Full commands and caveats in Quick start. This block is the minimal spine."
      >
        <TerminalBlock title="shell — mel bootstrap">
          {MEL_BOOTSTRAP_COMMANDS}
        </TerminalBlock>
        <p className="callout" role="note">
          This public site does not connect to your MEL instance. Operator truth lives in the process you serve
          locally (or on your host), not here.
        </p>
      </Section>

      {/* ── Why MEL Exists ────────────────────────────────── */}
      <Section title="Why MEL exists" id="why" kicker="motivation">
        <p>
          Incident response fails when stale data looks live, when intent and execution blur, and when exports
          omit context. MEL pushes the opposite: explicit freshness, lifecycle separation, and wording tied to
          evidence — not marketing comfort.
        </p>
        <p>
          It is not the only tool for mesh operations. It is a narrow, strict tool for operators who want
          evidence-first language, governable control, and a runtime that admits gaps instead of hiding them.
        </p>
      </Section>
    </>
  );
}

/* ─── Inline SVG Icons ──────────────────────────────── */

function ConsoleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2" y="3" width="20" height="14" rx="2" />
      <path d="M8 21h8M12 17v4" />
      <path d="M7 8l3 3-3 3" />
      <path d="M13 14h4" />
    </svg>
  );
}

function CliIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="4 17 10 11 4 5" />
      <line x1="12" y1="19" x2="20" y2="19" />
    </svg>
  );
}

function AuditIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="9" y1="15" x2="15" y2="15" />
      <line x1="9" y1="11" x2="15" y2="11" />
    </svg>
  );
}

function IncidentIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  );
}

function DiagIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  );
}

function PackIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="21 8 21 21 3 21 3 8" />
      <rect x="1" y="3" width="22" height="5" />
      <line x1="10" y1="12" x2="14" y2="12" />
    </svg>
  );
}

function LocalIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
      <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
      <line x1="6" y1="6" x2="6.01" y2="6" />
      <line x1="6" y1="18" x2="6.01" y2="18" />
    </svg>
  );
}
