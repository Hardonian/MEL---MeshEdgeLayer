import { PageHeader, Section } from '@/components/marketing';
import { DOCS_ENTRYPOINTS } from '@/lib/orientation';

export default function DocsRoutePage() {
  return (
    <>
      <PageHeader
        title="Documentation entrypoints"
        subtitle="The public site is orientation; canonical docs and operational contracts live in the repository docs tree."
      />

      <Section title="Start with these canonical docs">
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
