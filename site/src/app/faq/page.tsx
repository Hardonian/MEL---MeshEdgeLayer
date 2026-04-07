import type { ReactNode } from 'react';
import Link from 'next/link';
import { PageHeader, Section } from '@/components/marketing';

const items: { q: string; a: ReactNode }[] = [
  {
    q: 'Does MEL route mesh traffic or prove RF coverage?',
    a: (
      <>
        No. MEL is not the mesh routing stack. Maps and topology views reflect persisted evidence and interpretation — not guaranteed
        propagation unless your evidence supports it. See{' '}
        <Link href="/architecture">Architecture</Link> and the repo{' '}
        <a href="https://github.com/mel-project/mel/blob/main/docs/product/HONESTY_AND_BOUNDARIES.md" rel="noreferrer" target="_blank">
          honesty doc
        </a>
        .
      </>
    ),
  },
  {
    q: 'Which transports are actually supported?',
    a: (
      <>
        Direct serial/TCP and MQTT ingest are supported (with explicit partial/degraded semantics). BLE and HTTP ingest are{' '}
        <strong>unsupported</strong> and must stay labeled that way. Same matrix as the{' '}
        <a href="https://github.com/mel-project/mel/blob/main/README.md" rel="noreferrer" target="_blank">
          README
        </a>
        .
      </>
    ),
  },
  {
    q: 'What do “live” and “stale” mean?',
    a: (
      <>
        They describe evidence freshness in the database, not optimism. Canonical definitions:{' '}
        <a href="https://github.com/mel-project/mel/blob/main/docs/repo-os/terminology.md" rel="noreferrer" target="_blank">
          terminology.md
        </a>
        .
      </>
    ),
  },
  {
    q: 'Can I evaluate the UI without radios?',
    a: (
      <>
        Yes. Use <code>make demo-seed</code> and serve <code>demo_sandbox/mel.demo.json</code> — fixture-backed, not live proof.{' '}
        <Link href="/quickstart">Quick start</Link>.
      </>
    ),
  },
  {
    q: 'Is AI / local inference canonical truth?',
    a: (
      <>
        No. Assistive inference is non-canonical when present. Deterministic records and audit events win over narrative. See{' '}
        <a href="https://github.com/mel-project/mel/blob/main/AGENTS.md" rel="noreferrer" target="_blank">
          AGENTS.md
        </a>
        .
      </>
    ),
  },
  {
    q: 'What do I run before opening a PR?',
    a: (
      <>
        At minimum <code>make lint</code>, <code>make test</code>, <code>make build</code>, <code>make smoke</code>. For release-shaped
        gates, <code>make premerge-verify</code>.{' '}
        <a href="https://github.com/mel-project/mel/blob/main/CONTRIBUTING.md" rel="noreferrer" target="_blank">
          CONTRIBUTING.md
        </a>
        .
      </>
    ),
  },
];

export default function FaqPage() {
  return (
    <>
      <PageHeader
        title="FAQ"
        subtitle="Short answers bounded by repository truth. For depth, follow links into docs/ or run the binary locally."
      />

      <Section title="Questions">
        <dl className="faqList">
          {items.map((item) => (
            <div key={item.q} className="faqItem">
              <dt>{item.q}</dt>
              <dd>{item.a}</dd>
            </div>
          ))}
        </dl>
        <p>
          Full FAQ in-repo:{' '}
          <a href="https://github.com/mel-project/mel/blob/main/docs/FAQ.md" rel="noreferrer" target="_blank">
            docs/FAQ.md
          </a>
          .
        </p>
      </Section>
    </>
  );
}
