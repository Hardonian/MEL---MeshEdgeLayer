import { PageHeader, Section } from '@/components/marketing';

export default function AcknowledgementsPage() {
  return (
    <>
      <PageHeader
        title="Acknowledgements / dependencies"
        subtitle="MEL is built in the open and stands on serious upstream tooling. Credit is explicit, not hidden."
      />

      <Section title="Built with">
        <div className="grid">
          <article className="card">
            <h3>Go runtime and stdlib</h3>
            <p>Core daemon and CLI prioritize deterministic Go runtime behavior with stdlib-first discipline.</p>
          </article>
          <article className="card">
            <h3>React operator UI stack</h3>
            <p>Existing console interface is React + Vite + TypeScript, embedded into the MEL binary.</p>
          </article>
          <article className="card">
            <h3>Meshtastic protobuf schemas</h3>
            <p>In-repo schema files support compatibility parsing and transport-side evidence interpretation.</p>
          </article>
          <article className="card">
            <h3>SQLite operational tooling</h3>
            <p>The repo uses sqlite3 CLI workflows for deterministic checks and migrations in this environment.</p>
          </article>
        </div>
      </Section>

      <Section title="Dependency honesty">
        <p>
          MEL keeps base viability local-first and self-hosted friendly. Optional integrations are explicit optional layers,
          not hidden requirements. If something is unsupported or degraded, wording must say so plainly.
        </p>
      </Section>

      <Section title="License and openness posture">
        <p>
          MEL is open source under the <strong>GNU General Public License v3.0</strong> — see the{' '}
          <a href="https://github.com/mel-project/mel/blob/main/LICENSE" rel="noreferrer" target="_blank">
            LICENSE
          </a>{' '}
          file in the repository root. Contributor-facing docs are first-class runtime support surfaces, not afterthoughts.
        </p>
        <p>
          Third-party packages (Go modules, npm) remain under their respective licenses; see{' '}
          <a href="https://github.com/mel-project/mel/blob/main/docs/community/dependency-license-inventory.md" rel="noreferrer" target="_blank">
            dependency-license-inventory.md
          </a>
          .
        </p>
      </Section>
    </>
  );
}
