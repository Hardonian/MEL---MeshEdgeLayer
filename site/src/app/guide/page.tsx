import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';
import { MEL_BOOTSTRAP_COMMANDS } from '@/lib/orientation';

export default function GuidePage() {
  return (
    <>
      <PageHeader
        kicker="operator workflow"
        title="Operator guide"
        subtitle="A concise path through MEL’s real workflow: install, validate runtime truth, operate, and iterate."
      />

      <Section title="Start with deterministic setup" kicker="bootstrap" accent="green">
        <TerminalBlock title="Build + initialize + serve">
{MEL_BOOTSTRAP_COMMANDS}
        </TerminalBlock>
        <p>
          If transports are not configured yet, expect degraded or warning states. That is expected behavior, not a hidden failure.
        </p>
      </Section>

      <Section title="Understand truth boundaries before operating" kicker="truth model">
        <ul>
          <li>Ingest records and audit events are canonical runtime truth.</li>
          <li>Inference layers are assistive only and must not be treated as canonical state.</li>
          <li>Submission, approval, dispatch, execution, and audit are separate control states.</li>
        </ul>
      </Section>

      <Section title="Go deeper in canonical docs">
        <ul>
          <li>
            <a href={melGithubFile('docs/README.md')}>Documentation hub</a>
          </li>
          <li>
            <a href={melGithubFile('docs/getting-started/QUICKSTART.md')}>Quickstart playbook</a>
          </li>
          <li>
            <a href={melGithubFile('docs/repo-os/verification-matrix.md')}>Verification matrix</a>
          </li>
          <li>
            <a href={melGithubFile('docs/repo-os/release-readiness.md')}>Release readiness gate</a>
          </li>
        </ul>
        <p>
          Need contribution workflow details? Continue to <Link href="/contribute">Contribute</Link>.
        </p>
      </Section>
    </>
  );
}
