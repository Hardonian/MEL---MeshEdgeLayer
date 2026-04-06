import { PageHeader, Section } from '@/components/marketing';
import { repoBlob } from '@/lib/repo';

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
            <h3>Go runtime</h3>
            <p>
              Daemon and CLI are Go 1.24+. Third-party modules are listed in <code>go.mod</code> (not stdlib-only — e.g.
              SQLite via <code>modernc.org/sqlite</code>, TUI libraries for CLI surfaces).
            </p>
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

      <Section title="License">
        <p>
          The repository is licensed under the{' '}
          <a href={repoBlob('LICENSE')}>GNU General Public License v3.0</a>. This public site is part of the same repo and
          the same license applies unless a subdirectory states otherwise (it does not).
        </p>
        <p>
          Canonical dependency notes:{' '}
          <a href={repoBlob('docs/community/dependency-license-inventory.md')}>dependency-license-inventory.md</a>.
        </p>
      </Section>
    </>
  );
}
