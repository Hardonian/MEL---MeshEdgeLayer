import type { Metadata } from 'next';
import Link from 'next/link';
import { PageHeader, Section } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export const metadata: Metadata = {
  title: 'Compare',
  description: 'MEL vs generic observability — a sharp contrast on evidence, control, and deployment posture.',
};

export default function ComparePage() {
  return (
    <>
      <PageHeader
        kicker="positioning"
        title="MEL vs generic observability"
        subtitle="MEL optimizes for operator truth, evidence hierarchy, and governable control — not for the greenest dashboard screenshot."
      />

      <Section
        title="What you are comparing"
        id="context"
        kicker="context"
      >
        <p>
          Generic NMS, cloud IoT suites, and classic metrics stacks excel at volume, integrations, and enterprise
          packaging. MEL is narrower and stricter:{' '}
          <strong>mesh and edge operations</strong> where{' '}
          <strong>stale must not pose as live</strong> and where{' '}
          <strong>control intent must stay attributable</strong>.
        </p>
      </Section>

      <Section
        title="Comparison matrix"
        id="matrix"
        kicker="dimension by dimension"
        accent="green"
        description="Key dimensions where MEL makes different product decisions."
      >
        <table className="compareTable" style={{ marginTop: 'var(--space-sm)' }}>
          <caption className="srOnly">MEL versus generic observability tools comparison</caption>
          <thead>
            <tr>
              <th scope="col">Dimension</th>
              <th scope="col">Generic NMS / Cloud IoT</th>
              <th scope="col">MEL</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <th scope="row">Data states</th>
              <td className="compareGeneric">
                Often collapses to OK / warn / crit without persisted rationale.
              </td>
              <td className="compareMel">
                Keeps live, stale, historical, imported, partial, degraded, and unknown as first-class
                semantics tied to ingest and audit records.
              </td>
            </tr>
            <tr>
              <th scope="row">Transport claims</th>
              <td className="compareGeneric">
                May imply connectivity or health from partial signals or map overlays.
              </td>
              <td className="compareMel">
                Refuses to claim RF routing, coverage, or delivery unless evidence supports it.
                Unsupported paths stay labeled.
              </td>
            </tr>
            <tr>
              <th scope="row">Control lifecycle</th>
              <td className="compareGeneric">
                Can blur &ldquo;clicked run&rdquo; with &ldquo;executed safely.&rdquo;
              </td>
              <td className="compareMel">
                Separates submission, approval, dispatch, execution, and audit as distinct states with
                attribution at each step.
              </td>
            </tr>
            <tr>
              <th scope="row">Deployment model</th>
              <td className="compareGeneric">
                Hosted SaaS by default — data leaves your boundary, vendor controls the plane.
              </td>
              <td className="compareMel">
                Local-first binary. Base viability without a mandatory cloud control plane. No hidden
                telemetry defaults.
              </td>
            </tr>
            <tr>
              <th scope="row">AI / inference</th>
              <td className="compareGeneric">
                May surface AI output without clear distinction from ground truth.
              </td>
              <td className="compareMel">
                Assistive inference is labeled non-canonical. Deterministic layers always win conflicts.
              </td>
            </tr>
            <tr>
              <th scope="row">Evidence exports</th>
              <td className="compareGeneric">
                Exports may omit freshness context, control attribution, or degraded-state language.
              </td>
              <td className="compareMel">
                Proofpack-style exports preserve evidence chain, control-path attribution, and explicit
                posture at export time.
              </td>
            </tr>
          </tbody>
        </table>
      </Section>

      <Section title="When MEL is the wrong tool" id="wrong-tool" kicker="limitations" accent="blue">
        <ul>
          <li>You need a full NMS for non-mesh SNMP-heavy estates — MEL&apos;s workflow focus does not fit.</li>
          <li>
            You want RF routing, automatic transmit, or coverage proof as a product feature without bringing
            your own evidence discipline.
          </li>
          <li>
            You want assistive AI output treated as ground truth — MEL&apos;s contract says otherwise.
          </li>
          <li>
            You need high-volume metrics aggregation across thousands of nodes — MEL is incident and control
            focused, not a metrics platform.
          </li>
        </ul>
      </Section>

      <Section title="When MEL fits" id="right-tool" kicker="fit criteria">
        <ul>
          <li>You own your stack and need honest runtime truth without vendor-controlled health defaults.</li>
          <li>You operate in environments where stale data must be visible, not hidden.</li>
          <li>
            You need attributable control: who submitted, who approved, what executed, what the result was.
          </li>
          <li>You want a self-hostable binary with no mandatory cloud dependency for base functionality.</li>
          <li>You need evidence-chain exports that preserve posture at time of export.</li>
        </ul>
      </Section>

      <Section title="Read next" id="read-next">
        <ul>
          <li>
            <a href={melGithubFile('docs/product/WHY_MEL.md')} rel="noreferrer" target="_blank">
              Why MEL (docs)
            </a>
          </li>
          <li>
            <a href={melGithubFile('docs/product/DIFFERENTIATION_AND_MOAT.md')} rel="noreferrer" target="_blank">
              Differentiation and moat (docs)
            </a>
          </li>
          <li>
            <Link href="/quickstart">Quick start</Link> ·{' '}
            <Link href="/architecture">Architecture</Link> ·{' '}
            <Link href="/trust">Trust &amp; privacy</Link>
          </li>
        </ul>
      </Section>
    </>
  );
}
