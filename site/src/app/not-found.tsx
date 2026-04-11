import Link from 'next/link';
import { Section, TerminalBlock } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export default function NotFound() {
  return (
    <>
      <div className="errorPanel" aria-labelledby="error-heading">
        <p className="errorCode">404 — route not found</p>
        <h1 id="error-heading" style={{ margin: '0 0 0.5rem', fontSize: 'clamp(1.5rem, 3vw, 2.2rem)', letterSpacing: '-0.02em' }}>
          Path not recognized.
        </h1>
        <p style={{ color: 'var(--muted)', margin: '0', fontSize: '1.02rem' }}>
          This route does not exist on the public orientation site.
          The operator console runs locally via <code>mel serve</code>.
        </p>
      </div>

      <Section title="Operator recovery" id="recovery" kicker="recovery paths" accent="blue">
        <ul>
          <li>
            <Link href="/">Home</Link> — product framing and quick links
          </li>
          <li>
            <Link href="/quickstart">Quick start</Link> — zero-to-running path
          </li>
          <li>
            <Link href="/guide">Guide</Link> — site, console, and docs map
          </li>
          <li>
            <a href={melGithubFile('docs/README.md')} rel="noreferrer" target="_blank">
              Documentation hub
            </a>{' '}
            — canonical depth in the repository
          </li>
        </ul>
        <p className="callout" role="note">
          If you followed a stale bookmark, prefer the docs tree or the embedded UI served by{' '}
          <code>./bin/mel serve</code>.
        </p>
      </Section>

      <TerminalBlock title="shell — local console (when you meant the product UI)">
{`./bin/mel serve --config .tmp/mel.json
# open http://127.0.0.1:8080`}
      </TerminalBlock>
    </>
  );
}
