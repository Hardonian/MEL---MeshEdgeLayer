import Link from 'next/link';
import { PageHeader, Section, TerminalBlock } from '@/components/marketing';
import { melGithubFile } from '@/lib/repo';

export default function NotFound() {
  return (
    <>
      <PageHeader title="404 — route not found" subtitle="This path does not exist on the public orientation site." />
      <Section title="Operator recovery">
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
        <p>If you followed a stale bookmark, prefer the docs tree or the embedded UI served by `./bin/mel serve`.</p>
      </Section>
      <TerminalBlock title="Local console (when you meant the product UI)">
{`./bin/mel serve --config .tmp/mel.json
# open http://127.0.0.1:8080`}
      </TerminalBlock>
    </>
  );
}
