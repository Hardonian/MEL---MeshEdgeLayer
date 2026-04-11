import type { Metadata } from 'next';
import { PageHeader, Section } from '@/components/marketing';
import { DOCS_ENTRYPOINTS } from '@/lib/orientation';

export const metadata: Metadata = {
  title: 'Docs',
  description: 'Documentation entrypoints — canonical docs and operational contracts live in the repository docs tree.',
};

export default function DocsRoutePage() {
  return (
    <>
      <PageHeader
        kicker="documentation"
        title="Documentation entrypoints"
        subtitle="The public site is orientation; canonical docs and operational contracts live in the repository docs tree."
      />

      <Section
        title="Start with these canonical docs"
        kicker="repo docs"
        accent="blue"
        description="All links open the live repository — content is authoritative, not mirrored here."
      >
        <ul>
          {DOCS_ENTRYPOINTS.map((entry) => (
            <li key={entry.href}>
              <a href={entry.href} rel="noreferrer" target="_blank">
                {entry.label}
              </a>{' '}
              — {entry.detail}
            </li>
          ))}
        </ul>
      </Section>
    </>
  );
}
