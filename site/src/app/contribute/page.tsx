import Link from 'next/link';
import { PageHeader, Section, PrincipleList, TerminalBlock } from '@/components/marketing';
import { MEL_GITHUB_REPO, REPO_ISSUES_URL, REPO_URL, melGithubFile, repoBlob } from '@/lib/repo';

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
        kicker="open source contribution"
        title="Contribute"
        subtitle="Systems craft: truth boundaries, runtime reliability, docs clarity, and operator trust. Interesting problems live where evidence, control paths, and degraded states meet."
      />

      <Section
        title="Start here"
        id="start"
        kicker="onboarding"
        accent="green"
        description="Small steps that match how maintainers review work."
      >
        <ul>
          <li>
            Read{' '}
            <a href={melGithubFile('CONTRIBUTING.md')} rel="noreferrer" target="_blank">
              CONTRIBUTING.md
            </a>{' '}
            and{' '}
            <a href={melGithubFile('AGENTS.md')} rel="noreferrer" target="_blank">
              AGENTS.md
            </a>{' '}
            for repo contract and verification expectations.
          </li>
          <li>
            Browse{' '}
            <a href={melGithubFile('docs/community/README.md')} rel="noreferrer" target="_blank">
              docs/community/
            </a>{' '}
            and{' '}
            <a href={melGithubFile('docs/community/WHY_CONTRIBUTE.md')} rel="noreferrer" target="_blank">
              Why contribute
            </a>{' '}
            for contributor-oriented entrypoints.
          </li>
          <li>
            Map the tree:{' '}
            <a href={melGithubFile('docs/contributor/ARCHITECTURE_MAP.md')} rel="noreferrer" target="_blank">
              ARCHITECTURE_MAP.md
            </a>{' '}
            and <Link href="/architecture">Architecture</Link> on this site.
          </li>
          <li>
            Open a focused issue or pick one:{' '}
            <a href={REPO_ISSUES_URL} rel="noreferrer" target="_blank">
              GitHub issues
            </a>
            .
          </li>
        </ul>
      </Section>

      <Section title="Clone and verify locally" id="clone">
        <TerminalBlock title="Repository">
{`git clone ${MEL_GITHUB_REPO}.git
cd MEL-MeshEdgeLayer`}
        </TerminalBlock>
        <p>
          Default verification before you claim behavior: <code>make lint</code>, <code>make test</code>, <code>make build</code>,{' '}
          <code>make smoke</code>. Broader stack signal: <code>make verify-stack</code> (same as <code>make check</code>).
        </p>
        <p>
          When semantics or capability wording changes: <code>make premerge-verify</code>. Node <strong>24.x</strong> for{' '}
          <code>frontend/</code> and <code>site/</code>; Go <strong>1.24+</strong> for the binary.
        </p>
      </Section>

      <Section title="Why contribute here" id="why">
        <p>
          MEL rewards disciplined engineering. Useful work improves operator clarity, strengthens deterministic behavior, and
          reduces ambiguity in incidents and control paths — hard to copy from UI polish alone.
        </p>
      </Section>

      <Section title="Lanes we want" id="lanes">
        <ul>
          <li>Go/runtime correctness and transport reliability hardening.</li>
          <li>Operator UX (embedded console) that improves truth visibility and degraded-state clarity.</li>
          <li>This public site (<code>site/</code>) when it helps orientation without overclaiming.</li>
          <li>Docs, runbooks, quickstart accuracy, and troubleshooting depth.</li>
          <li>Tests, fixtures, and regression verification.</li>
          <li>Reproducible bug reports and field validation write-ups.</li>
        </ul>
      </Section>

      <Section title="What a good PR includes" id="good-pr">
        <ul>
          <li>Bounded claim language that matches implementation truth.</li>
          <li>Verification commands and outcomes (`make lint`, `make test`, `make build`, `make smoke` minimum).</li>
          <li>Explicit degraded/unknown-state behavior when applicable.</li>
          <li>Residual risk and caveats instead of hidden assumptions.</li>
        </ul>
      </Section>

      <Section title="Contribution doctrine" id="doctrine" kicker="principles" accent="blue">
        <PrincipleList items={contributionPrinciples} />
      </Section>

      <Section title="Related" id="related">
        <ul>
          <li>
            <Link href="/quickstart">Quick start</Link> · <Link href="/guide">Guide</Link> ·{' '}
            <a href={repoBlob('docs/README.md')} rel="noreferrer" target="_blank">
              docs/README.md
            </a>
          </li>
        </ul>
      </Section>


      <Section title="Where to open work">
        <ul>
          <li>Repository: <a href={REPO_URL}>{REPO_URL}</a></li>
          <li>Issues: <a href={REPO_ISSUES_URL}>{REPO_ISSUES_URL}</a></li>
          <li>Discussions: use issue templates for scoped proposals and field reports.</li>
          <li>Contributor guide: <a href={melGithubFile('CONTRIBUTING.md')}>CONTRIBUTING.md</a></li>
        </ul>
      </Section>
    </>
  );
}
