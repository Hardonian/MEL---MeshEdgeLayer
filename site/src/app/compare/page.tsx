import Link from 'next/link';
import { PageHeader, Section } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export default function ComparePage() {
  return (
    <>
      <PageHeader
        title="MEL vs generic observability"
        subtitle="A sharp contrast on purpose: MEL optimizes for operator truth, evidence hierarchy, and governable control — not for the greenest dashboard screenshot."
      />

      <Section title="What you are comparing">
        <p>
          Generic NMS, cloud IoT suites, and classic metrics stacks excel at volume, integrations, and enterprise packaging. MEL is narrower
          and stricter: <strong>mesh and edge operations</strong> where <strong>stale must not pose as live</strong> and where{' '}
          <strong>control intent must stay attributable</strong>.
        </p>
      </Section>

      <Section title="Truth and evidence">
        <ul>
          <li>
            <strong>Generic tools</strong> often collapse states into OK/warn/crit without a persisted, replayable notion of why.
          </li>
          <li>
            <strong>MEL</strong> keeps <strong>live, stale, historical, imported, partial, degraded, and unknown</strong> as first-class
            semantics tied to ingest and audit records — see{' '}
            <a href={melGithubFile('docs/repo-os/terminology.md')} rel="noreferrer" target="_blank">
              terminology
            </a>
            .
          </li>
        </ul>
      </Section>

      <Section title="Transport and RF claims">
        <ul>
          <li>
            <strong>Generic maps</strong> can imply connectivity or health from partial signals.
          </li>
          <li>
            <strong>MEL</strong> refuses to be a <strong>mesh routing stack</strong> or to claim RF routing, coverage, or delivery unless
            evidence supports it — see the matrix in the{' '}
            <a href={melGithubFile('README.md')} rel="noreferrer" target="_blank">
              repository README
            </a>
            .
          </li>
        </ul>
      </Section>

      <Section title="Control and safety">
        <ul>
          <li>
            <strong>Generic automation</strong> can blur &quot;clicked run&quot; with &quot;executed safely.&quot;
          </li>
          <li>
            <strong>MEL</strong> treats submission, approval, dispatch, execution, and audit as separable ideas — see{' '}
            <a href={melGithubFile('docs/ops/CONTROL_PLANE_TRUST.md')} rel="noreferrer" target="_blank">
              control-plane trust
            </a>
            .
          </li>
        </ul>
      </Section>

      <Section title="Deployment posture">
        <ul>
          <li>
            <strong>Hosted SaaS</strong> optimizes for vendor-operated scale; data leaves your boundary by default.
          </li>
          <li>
            <strong>MEL</strong> is <strong>local-first</strong>: base viability without a mandatory cloud control plane — see{' '}
            <Link href="/trust">Trust &amp; privacy</Link> and{' '}
            <a href={melGithubFile('docs/release/PRIVACY_AND_DATA_POSTURE.md')} rel="noreferrer" target="_blank">
              privacy and data posture
            </a>
            .
          </li>
        </ul>
      </Section>

      <Section title="When MEL is the wrong tool">
        <ul>
          <li>You need a full NMS for non-mesh SNMP-heavy estates and MEL&apos;s workflow focus does not fit.</li>
          <li>You want RF routing, automatic transmit, or coverage proof as a product feature without bringing your own evidence discipline.</li>
          <li>You want assistive AI output treated as ground truth — MEL&apos;s contract says otherwise.</li>
        </ul>
      </Section>

      <Section title="Read next">
        <ul>
          <li>
            <a href={melGithubFile('docs/product/WHY_MEL.md')} rel="noreferrer" target="_blank">
              Why MEL
            </a>
          </li>
          <li>
            <a href={melGithubFile('docs/product/DIFFERENTIATION_AND_MOAT.md')} rel="noreferrer" target="_blank">
              Differentiation and moat
            </a>
          </li>
          <li>
            <Link href="/quickstart">Quick start</Link>
          </li>
        </ul>
      </Section>
    </>
  );
}
