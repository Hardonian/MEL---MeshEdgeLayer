import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { melGithubFile, MEL_GITHUB_REPO } from '@/lib/repo';

export default function GuidePage() {
  return (
    <>
      <PageHeader
        title="Guide"
        subtitle="How repository docs, this public site, and the embedded operator UI fit together. Depth stays in the repo; this page is a map."
      />

      <Section title="Three surfaces" id="surfaces">
        <div className="grid">
          <article className="card">
            <h3>This site (orientation)</h3>
            <p>
              Static pages: product framing, quick start summary, trust posture, and links into GitHub. No MEL backend required;
              nothing here proves your deployment is healthy.
            </p>
          </article>
          <article className="card">
            <h3>Embedded operator UI</h3>
            <p>
              Built from <code>frontend/</code>, shipped inside the binary, served at <code>http://127.0.0.1:8080</code> when you run{' '}
              <code>mel serve</code>. That is the live console for incidents, evidence, and runtime truth.
            </p>
          </article>
          <article className="card">
            <h3>Docs in the repository</h3>
            <p>
              Canonical procedures, architecture, limits, and runbooks live under <code>docs/</code> on GitHub. Start at the{' '}
              <a href={melGithubFile('docs/README.md')} rel="noreferrer" target="_blank">
                documentation hub
              </a>
              .
            </p>
          </article>
        </div>
      </Section>

      <Section title="Suggested reading order" id="reading-order">
        <ol className="orderedList">
          <li>
            <Link href="/quickstart">Quick start</Link> — run the binary locally (or demo seed).
          </li>
          <li>
            <Link href="/architecture">Architecture</Link> — truth hierarchy and component map.
          </li>
          <li>
            <a href={melGithubFile('docs/README.md')} rel="noreferrer" target="_blank">
              docs/README.md
            </a>{' '}
            — full index.
          </li>
          <li>
            <a href={melGithubFile('docs/ops/limitations.md')} rel="noreferrer" target="_blank">
              Known limitations
            </a>{' '}
            and{' '}
            <a href={melGithubFile('docs/ops/support-matrix.md')} rel="noreferrer" target="_blank">
              support matrix
            </a>
            .
          </li>
        </ol>
      </Section>

      <Section title="CLI entrypoints" id="cli">
        <p>Operators typically work from the repo root after <code>make build</code>:</p>
        <TerminalBlock title="Common commands">
{`./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json`}
        </TerminalBlock>
        <p>
          Repository:{' '}
          <a href={MEL_GITHUB_REPO} rel="noreferrer" target="_blank">
            {MEL_GITHUB_REPO.replace('https://', '')}
          </a>
        </p>
      </Section>
    </>
  );
}
