import { PageHeader, Section, PrincipleList } from '@/components/marketing';

const contributionPrinciples = [
  { name: 'No theatre', detail: 'No fake transport support, no fake live state, no overclaiming UI language.' },
  { name: 'Operator truth first', detail: 'If implementation and wording conflict, tighten wording or strengthen implementation.' },
  { name: 'Evidence over claims', detail: 'Verification output, tests, and runtime artifacts beat speculative confidence.' },
  { name: 'Local-first and privacy respect', detail: 'Do not widen telemetry or trust boundaries silently.' },
];

export default function ContributePage() {
  return (
    <>
      <PageHeader
        title="Contribute"
        subtitle="MEL contributions are systems craft: truth boundaries, runtime reliability, docs clarity, and operator trust."
      />

      <Section title="Why contribute here">
        <p>
          MEL rewards disciplined engineering. Useful contributions improve operator clarity, strengthen deterministic behavior,
          and reduce ambiguity in incidents and control paths.
        </p>
      </Section>

      <Section title="Contribution lanes we actively want">
        <ul>
          <li>Go/runtime correctness and transport reliability hardening.</li>
          <li>Frontend/operator UX that improves truth visibility and degraded-state clarity.</li>
          <li>Docs/runbook quality, quickstart correctness, and troubleshooting depth.</li>
          <li>Test hardening, fixture scenarios, and regression verification.</li>
          <li>Field validation reports, bug reports, and issue triage evidence.</li>
        </ul>
      </Section>

      <Section title="Local dev workflow orientation">
        <ul>
          <li>Start with `CONTRIBUTING.md`, `AGENTS.md`, and role paths in `docs/community/`.</li>
          <li>Run verification chain: `make lint`, `make test`, `make build`, and `make smoke` before strong claims.</li>
          <li>Use Node 24.x for frontend targets; use Go 1.24+ for runtime targets.</li>
        </ul>
      </Section>

      <Section title="Contribution doctrine">
        <PrincipleList items={contributionPrinciples} />
      </Section>
    </>
  );
}
