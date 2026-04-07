import Link from 'next/link';
import { PrincipleList, Section, TerminalBlock } from '@/components/marketing';

const principles = [
  { name: 'Evidence before narrative', detail: 'Runtime records and audit events are canonical; commentary follows evidence.' },
  { name: 'Explicit degraded states', detail: 'Partial, stale, imported, and unknown states stay visible instead of being painted healthy.' },
  { name: 'Trusted control lifecycle', detail: 'Submission, approval, dispatch, execution, and audit are separate states with attribution.' },
  { name: 'Local-first posture', detail: 'MEL remains useful without mandatory cloud dependencies or hidden telemetry defaults.' },
];

export default function HomePage() {
  return (
    <>
      <section className="hero">
        <p className="kicker">MeshEdgeLayer / operator surface</p>
        <h1>Truth-preserving incident intelligence for mesh and edge operators.</h1>
        <p>
          MEL is an incident-intelligence and trusted-control operating system. It ingests evidence, keeps action history
          attributable, and makes degraded or unknown states explicit.
        </p>
        <div className="ctaRow">
          <Link href="/quickstart" className="btn primary">
            Quick start in 90 seconds
          </Link>
          <Link href="/architecture" className="btn">
            Architecture
          </Link>
          <Link href="/trust" className="btn">
            Trust &amp; privacy
          </Link>
          <Link href="/contribute" className="btn">
            Contribute
          </Link>
          <a href="https://github.com/mel-project/mel/blob/main/docs/README.md" className="btn" rel="noreferrer" target="_blank">
            Full docs
          </a>
          <a href="https://meshtastic.org/docs/introduction" className="btn" rel="noreferrer" target="_blank">
            Meshtastic reference
          </a>
        </div>
      </section>

      <Section title="Why MEL exists">
        <p>
          Operators need more than pretty maps and optimistic status labels. MEL exists to keep incident operations tied to
          persisted evidence, preserve control-path accountability, and keep uncertainty visible instead of hidden.
        </p>
      </Section>

      <Section title="What MEL is / is not">
        <div className="grid">
          <article className="card">
            <h3>What it is</h3>
            <p>Local-first operator OS for mesh observability, incident workflows, and trusted control.</p>
          </article>
          <article className="card">
            <h3>What it is not</h3>
            <p>Not a mesh routing stack. Not a fake AI autopilot. Not a dashboard skin hiding evidence gaps.</p>
          </article>
          <article className="card">
            <h3>Transport truth</h3>
            <p>Direct serial/TCP and MQTT ingest are supported. BLE and HTTP ingest are explicitly unsupported.</p>
          </article>
        </div>
      </Section>

      <Section title="Field-ready capabilities">
        <div className="grid">
          <article className="card">
            <h3>Incident timeline and queue</h3>
            <p>Operator workflow centered on incident state, rationale, and replayable evidence context.</p>
          </article>
          <article className="card">
            <h3>Truthful transport diagnostics</h3>
            <p>Readiness, stale signals, dead letters, and doctor outputs are surfaced with bounded semantics.</p>
          </article>
          <article className="card">
            <h3>Proofpack and support bundle flow</h3>
            <p>Evidence exports and runbook-oriented operations for handoffs, triage, and review.</p>
          </article>
          <article className="card">
            <h3>CLI and local runtime core</h3>
            <p>Build, initialize, doctor, and serve from a local binary with no mandatory hosted control plane.</p>
          </article>
        </div>
      </Section>

      <Section title="Operator-truth doctrine">
        <PrincipleList items={principles} />
      </Section>

      <TerminalBlock title="Quick run path">
{`make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json`}
      </TerminalBlock>
    </>
  );
}
