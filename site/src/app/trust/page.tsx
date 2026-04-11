import type { Metadata } from 'next';
import Link from 'next/link';
import { PageHeader, Section } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export const metadata: Metadata = {
  title: 'Trust & Privacy',
  description: 'Honest posture for self-hosted operators: what MEL assumes, what it does not promise, and where to read the full contracts.',
};

export default function TrustPage() {
  return (
    <>
      <PageHeader
        kicker="trust, privacy, security"
        title="Trust, privacy, and security"
        subtitle="Honest posture for self-hosted operators: what MEL assumes, what it does not promise, and where to read the full contracts."
      />

      <Section title="Local-first and data ownership" kicker="data posture" accent="green">
        <p>
          MEL is designed to run on hardware you control. Base viability does not depend on a mandatory cloud control plane. What leaves
          the host — MQTT, backups, exports — is a <strong>configuration choice</strong> you must review.
        </p>
        <p>
          Deep read:{' '}
          <a href={melGithubFile('docs/release/PRIVACY_AND_DATA_POSTURE.md')} rel="noreferrer" target="_blank">
            Privacy and data posture
          </a>
          ,{' '}
          <a href={melGithubFile('docs/privacy/posture.md')} rel="noreferrer" target="_blank">
            Privacy posture
          </a>
          .
        </p>
      </Section>

      <Section title="Telemetry and outbound behavior" kicker="explicit defaults">
        <p>
          There is no “quiet phone home” narrative here — defaults and optional integrations must stay explicit in config and docs. If you
          enable broker backhaul or remote exposure, treat that as an operator decision with reviewable scope.
        </p>
      </Section>

      <Section title="Security reporting" kicker="vulnerability disclosure">
        <p>
          Report vulnerabilities responsibly: see{' '}
          <a href={melGithubFile('SECURITY.md')} rel="noreferrer" target="_blank">
            SECURITY.md
          </a>{' '}
          in the repository. Do not post secrets or precise location payloads in public issues.
        </p>
      </Section>

      <Section title="Control-plane safeguards" kicker="trusted control" accent="blue">
        <p>
          Trusted control means separable lifecycle states and attributable records — not flashy buttons that blur intent and execution.
        </p>
        <p>
          <a href={melGithubFile('docs/ops/CONTROL_PLANE_TRUST.md')} rel="noreferrer" target="_blank">
            Control-plane trust (operations)
          </a>
          {' · '}
          <a href={melGithubFile('docs/architecture/CONTROL_PLANE_TRUST_MODEL.md')} rel="noreferrer" target="_blank">
            Control-plane trust model (architecture)
          </a>
          .
        </p>
      </Section>

      <Section title="License">
        <p>
          MEL is open source under <strong>GPL-3.0</strong> — see{' '}
          <a href={melGithubFile('LICENSE')} rel="noreferrer" target="_blank">
            LICENSE
          </a>
          . Bundled dependencies (Go modules, npm) carry their own licenses.
        </p>
      </Section>

      <Section title="Related">
        <ul>
          <li>
            <Link href="/acknowledgements">Credits &amp; dependencies</Link>
          </li>
          <li>
            <Link href="/faq">FAQ</Link>
          </li>
          <li>
            <a href={melGithubFile('docs/ops/limitations.md')} rel="noreferrer" target="_blank">
              Known limitations
            </a>
          </li>
        </ul>
      </Section>
    </>
  );
}
