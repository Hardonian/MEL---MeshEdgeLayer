import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { repoBlob } from '@/lib/repo';

export default function QuickStartPage() {
  return (
    <>
      <PageHeader
        title="Quick start"
        subtitle="Operator-first path from zero to a running MEL instance. Commands below mirror the repository quickstart."
      />

      <Section title="Prerequisites">
        <ul>
          <li>Go 1.24+ for CLI build targets.</li>
          <li>Node 24.x only when running frontend verification commands.</li>
          <li>Local shell access with permission to run `make` and execute `./bin/mel`.</li>
        </ul>
      </Section>

      <Section title="Day 0 run path (10–15 minutes)">
        <TerminalBlock title="Initialize and serve">
{`make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json`}
        </TerminalBlock>
        <p>Open <code>http://127.0.0.1:8080</code> and verify status, transport state clarity, and incident queue visibility.</p>
      </Section>

      <Section title="Fixture-backed mode (no radio required)">
        <TerminalBlock title="Deterministic demo seed">
{`make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json`}
        </TerminalBlock>
        <p>Use this path when evaluating the UI without active devices. It is simulation data, not live transport proof.</p>
      </Section>

      <Section title="First-run expectations and caveats">
        <ul>
          <li>`mel doctor` may report warnings on fresh installs with no configured transports.</li>
          <li>`/healthz` is liveness only; use readiness and status endpoints for runtime truth.</li>
          <li>BLE ingest and HTTP ingest are currently unsupported and should remain labeled unsupported.</li>
        </ul>
      </Section>

      <Section title="Next steps">
        <p>
          Continue with <Link href="/help">orientation</Link>, then the repo guides:{' '}
          <a href={repoBlob('docs/getting-started/QUICKSTART.md')}>QUICKSTART.md</a>,{' '}
          <a href={repoBlob('docs/ops/support-matrix.md')}>support matrix</a>,{' '}
          <a href={repoBlob('docs/ops/limitations.md')}>known limitations</a>.
        </p>
      </Section>
    </>
  );
}
