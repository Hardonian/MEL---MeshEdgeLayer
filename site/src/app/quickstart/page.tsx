import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { repoBlob } from '@/lib/repo';
import { MEL_BOOTSTRAP_COMMANDS, MEL_DEMO_COMMANDS, MEL_FIRST_PROOF_COMMANDS } from '@/lib/orientation';

export default function QuickStartPage() {
  return (
    <>
      <PageHeader
        title="Quick start"
        subtitle="From clone to a running binary and embedded console. Commands align with the repository; depth lives in docs and in the running process."
      />

      <Section title="Prerequisites">
        <ul>
          <li>Go 1.24+ for CLI build targets.</li>
          <li>Node 24.x only when running frontend verification commands.</li>
          <li>Local shell access with permission to run `make` and execute `./bin/mel`.</li>
          <li>Port <code>8080</code> available for local `mel serve`.</li>
        </ul>
      </Section>

      <Section title="Day 0 run path">
        <TerminalBlock title="Initialize and serve">
{MEL_BOOTSTRAP_COMMANDS}
        </TerminalBlock>
        <p>Open <code>http://127.0.0.1:8080</code> and verify status, transport state clarity, and incident queue visibility.</p>
      </Section>

      <Section title="What first success looks like">
        <ul>
          <li>`mel serve` stays running with no crash loop.</li>
          <li>The embedded console loads on <code>127.0.0.1:8080</code>.</li>
          <li>Health endpoints respond: <code>/healthz</code>, <code>/readyz</code>, and <code>/api/v1/status</code>.</li>
          <li>If no transports are connected yet, state is shown as warning/degraded instead of “healthy.”</li>
        </ul>
      </Section>

      <Section title="If first run fails, check these first">
        <TerminalBlock title="Fast triage checks">
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

      <Section title="Fixture-backed mode (no radio required)">
        <TerminalBlock title="Deterministic demo seed">
{MEL_DEMO_COMMANDS}
        </TerminalBlock>
        <p>Use this path when evaluating the UI without active devices. It is simulation data, not live transport proof.</p>
      </Section>

      <Section title="Fastest first-proof command">
        <TerminalBlock title="One-command first proof">
{MEL_FIRST_PROOF_COMMANDS}
        </TerminalBlock>
        <p>
          This path writes deterministic evidence artifacts and seeded records so operators can validate incident + control workflows without
          claiming live RF routing or unsupported ingest surfaces.
        </p>
      </Section>

      <Section title="First-run expectations and caveats">
        <ul>
          <li>`mel doctor` may report warnings on fresh installs with no configured transports.</li>
          <li>`/healthz` is liveness only; use readiness and status endpoints for runtime truth.</li>
          <li>BLE ingest and HTTP ingest are currently unsupported and should remain labeled unsupported.</li>
          <li>MEL is not the RF routing stack; do not treat fixture mode as propagation proof.</li>
        </ul>
      </Section>

      <Section title="Next steps">
        <p>
          See <Link href="/guide">how this site relates to the console and docs</Link>, then <Link href="/help">Help</Link> for
          UI semantics. In-repo:{' '}
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
