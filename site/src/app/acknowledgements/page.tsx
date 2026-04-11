import type { Metadata } from 'next';
import { PageHeader, Section } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export const metadata: Metadata = {
  title: 'Acknowledgements',
  description: 'Upstream tooling and license posture — explicit credit, no implied endorsement.',
};

export default function AcknowledgementsPage() {
  return (
    <>
      <PageHeader
        kicker="stack & credits"
        title="Acknowledgements"
        subtitle="Upstream tooling and license posture — explicit credit, no implied endorsement. For depth, read the repo manifests, not marketing summaries."
      />

      <Section title="Built with" kicker="runtime stack" accent="blue">
        <div className="grid">
          <article className="card">
            <h3>Go runtime</h3>
            <p>
              Daemon and CLI are Go 1.24+. Third-party modules listed in <code>go.mod</code> — e.g.
              SQLite via <code>modernc.org/sqlite</code>, TUI libraries for CLI surfaces.
            </p>
          </article>
          <article className="card">
            <h3>React operator UI stack</h3>
            <p>Console interface is React + Vite + TypeScript, compiled and embedded into the MEL binary.</p>
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

      <Section title="Dependency honesty" kicker="posture">
        <p>
          MEL keeps base viability local-first and self-hosted friendly. Optional integrations are explicit
          optional layers, not hidden requirements. If something is unsupported or degraded, wording must say
          so plainly.
        </p>
      </Section>

      <Section title="License" kicker="open source" accent="green">
        <p>
          MEL is open source under the <strong>GNU General Public License v3.0</strong> — see the{' '}
          <a href={melGithubFile('LICENSE')} rel="noreferrer" target="_blank">
            LICENSE
          </a>{' '}
          file in the repository root. Contributor-facing docs are first-class runtime support surfaces, not
          afterthoughts.
        </p>
        <p>
          Third-party packages (Go modules, npm) remain under their respective licenses; see{' '}
          <a href={melGithubFile('docs/community/dependency-license-inventory.md')} rel="noreferrer" target="_blank">
            dependency-license-inventory.md
          </a>
          .
        </p>
      </Section>
    </>
  );
}
