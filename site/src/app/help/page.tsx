import { PageHeader, Section } from '@/components/marketing';

export default function HelpPage() {
  return (
    <>
      <PageHeader
        title="Help / orientation"
        subtitle="Fast context for operators: what surfaces exist, what state labels mean, and where to go when evidence looks incomplete."
      />

      <Section title="Main surfaces in MEL">
        <div className="grid">
          <article className="card">
            <h3>Status and diagnostics</h3>
            <p>Check readiness posture, transport visibility, and host-level checks via status endpoints and `mel doctor`.</p>
          </article>
          <article className="card">
            <h3>Incidents and evidence</h3>
            <p>Incident queue, timeline context, proofpacks, and action history tie outcomes to persisted records.</p>
          </article>
          <article className="card">
            <h3>Transports and messages</h3>
            <p>Observe ingest path health, stale/degraded conditions, dead letters, and replay context.</p>
          </article>
          <article className="card">
            <h3>Control lifecycle</h3>
            <p>Submission, approval, dispatch, execution result, and audit state are separate for trust and attribution.</p>
          </article>
        </div>
      </Section>

      <Section title="Semantic labels you should trust">
        <ul>
          <li><strong>Live</strong>: recent persisted ingest evidence exists.</li>
          <li><strong>Stale</strong>: evidence exists but is old for runtime confidence.</li>
          <li><strong>Historical/imported</strong>: context for analysis, not direct proof of current runtime.</li>
          <li><strong>Partial/degraded/unknown</strong>: known gaps, missing context, or unsupported conditions are explicit.</li>
        </ul>
      </Section>

      <Section title="Common first questions">
        <ul>
          <li>“Is MEL routing RF traffic?” → No. MEL is not the mesh routing stack.</li>
          <li>“Why is doctor warning?” → On first run, missing active transports is normal and intentionally visible.</li>
          <li>“Why does a map or panel look incomplete?” → Treat missing data as degraded/unknown, then inspect status, diagnostics, and transport logs.</li>
        </ul>
      </Section>

      <Section title="Troubleshooting entrypoints">
        <ul>
          <li>Run `./bin/mel doctor --config ...` for host and runtime checks.</li>
          <li>Check `/api/v1/status`, `/readyz`, and `/api/v1/readyz` before assuming healthy ingest.</li>
          <li>Use docs in `docs/ops/troubleshooting.md` and `docs/runbooks/` for targeted recovery paths.</li>
        </ul>
      </Section>
    </>
  );
}
