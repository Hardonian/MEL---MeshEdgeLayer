import type { Metadata } from 'next';
import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { repoBlob } from '@/lib/repo';
import { MEL_BOOTSTRAP_COMMANDS, MEL_DEMO_COMMANDS, MEL_FIRST_PROOF_COMMANDS } from '@/lib/orientation';

export const metadata: Metadata = {
  title: 'Quick Start',
  description: 'From clone to a running binary and embedded console. Deterministic commands with expected outcomes.',
};

export default function QuickStartPage() {
  return (
    <>
      <PageHeader
        kicker="getting started"
        title="Quick start"
        subtitle="From clone to a running binary and embedded console. Commands align with the repository; depth lives in docs and in the running process."
      />

      <Section title="Prerequisites" id="prereqs" kicker="environment">
        <ul>
          <li>Go 1.24+ for CLI build targets.</li>
          <li>Node 24.x only when running frontend verification commands.</li>
          <li>Local shell access with permission to run <code>make</code> and execute <code>./bin/mel</code>.</li>
          <li>Port <code>8080</code> available for local <code>mel serve</code>.</li>
        </ul>
      </Section>

      <Section title="Day 0 run path" id="day-0" kicker="bootstrap" accent="green">
        <TerminalBlock title="shell — initialize and serve">
          {MEL_BOOTSTRAP_COMMANDS}
        </TerminalBlock>
        <p>
          Open <code>http://127.0.0.1:8080</code> and verify status, transport state clarity, and incident
          queue visibility.
        </p>
      </Section>

      <Section
        title="What first success looks like"
        id="success"
        kicker="expected outcomes"
        description="Verify these before investing further in configuration."
      >
        <ul>
          <li><code>mel serve</code> stays running with no crash loop.</li>
          <li>The embedded console loads on <code>127.0.0.1:8080</code>.</li>
          <li>Health endpoints respond: <code>/healthz</code>, <code>/readyz</code>, and <code>/api/v1/status</code>.</li>
          <li>
            If no transports are connected yet, state shows as <strong>warning/degraded</strong> — not &ldquo;healthy.&rdquo;
            This is correct behavior.
          </li>
        </ul>
      </Section>

      <Section title="Triage when first run fails" id="triage" kicker="fast triage" accent="blue">
        <TerminalBlock title="shell — fast triage checks">
{`./bin/mel doctor --config .tmp/mel.json
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
curl -fsS http://127.0.0.1:8080/api/v1/status`}
        </TerminalBlock>
        <ul>
          <li>Permission errors: confirm config file permissions are locked down (<code>chmod 600</code>).</li>
          <li>Port conflicts: stop the process already using <code>8080</code>, then rerun <code>mel serve</code>.</li>
          <li>Doctor warnings on fresh installs are expected until real transports are connected.</li>
        </ul>
      </Section>

      <Section
        title="Fixture-backed mode"
        id="fixture"
        kicker="no radio required"
        description="Use this path when evaluating the UI without active devices. It is simulation data, not live transport proof."
      >
        <TerminalBlock title="shell — deterministic demo seed">
          {MEL_DEMO_COMMANDS}
        </TerminalBlock>
      </Section>

      <Section title="Fastest first-proof command" id="first-proof" kicker="one-command proof" accent="green">
        <TerminalBlock title="shell — first proof">
          {MEL_FIRST_PROOF_COMMANDS}
        </TerminalBlock>
        <p>
          This path writes deterministic evidence artifacts and seeded records so operators can validate incident
          and control workflows without claiming live RF routing or unsupported ingest surfaces.
        </p>
      </Section>

      <Section title="Caveats and first-run expectations" id="caveats" kicker="known caveats">
        <ul>
          <li><code>mel doctor</code> may report warnings on fresh installs with no configured transports.</li>
          <li><code>/healthz</code> is liveness only; use readiness and status endpoints for runtime truth.</li>
          <li>BLE ingest and HTTP ingest are currently unsupported — they should remain labeled unsupported.</li>
          <li>MEL is not the RF routing stack; do not treat fixture mode as propagation proof.</li>
        </ul>
      </Section>

      <Section title="Next steps" id="next">
        <p>
          See <Link href="/guide">how this site relates to the console and docs</Link>, then{' '}
          <Link href="/help">Help</Link> for UI semantics. In-repo:{' '}
          <a href={repoBlob('docs/getting-started/QUICKSTART.md')} rel="noreferrer" target="_blank">
            QUICKSTART.md
          </a>
          ,{' '}
          <a href={repoBlob('docs/ops/support-matrix.md')} rel="noreferrer" target="_blank">
            support matrix
          </a>
          ,{' '}
          <a href={repoBlob('docs/ops/limitations.md')} rel="noreferrer" target="_blank">
            known limitations
          </a>
          .
        </p>
      </Section>
    </>
  );
}
