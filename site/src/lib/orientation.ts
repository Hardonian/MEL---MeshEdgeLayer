import { melGithubFile } from '@/lib/repo';

export const MEL_BOOTSTRAP_COMMANDS = `make build
./bin/mel init --config .tmp/mel.json
chmod 600 .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json` as const;

export const MEL_DEMO_COMMANDS = `make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json` as const;

export const MEL_FIRST_PROOF_COMMANDS = `make first-proof
./bin/mel serve --config demo_sandbox/mel.first-proof.json` as const;

export const DOCS_ENTRYPOINTS = [
  {
    label: 'Docs hub',
    href: melGithubFile('docs/README.md'),
    detail: 'Single index for getting started, trust boundaries, and operations.',
  },
  {
    label: 'Quickstart playbook',
    href: melGithubFile('docs/getting-started/QUICKSTART.md'),
    detail: 'Clone-to-running path with caveats and expected first-run diagnostics.',
  },
  {
    label: 'Support matrix',
    href: melGithubFile('docs/ops/support-matrix.md'),
    detail: 'Implemented vs unsupported capabilities and transport posture.',
  },
  {
    label: 'Known limitations',
    href: melGithubFile('docs/ops/limitations.md'),
    detail: 'Current constraints to keep claims bounded to implementation truth.',
  },
  {
    label: 'Verification matrix',
    href: melGithubFile('docs/repo-os/verification-matrix.md'),
    detail: 'Required checks by change type before merge or release claims.',
  },
  {
    label: 'Release readiness gate',
    href: melGithubFile('docs/repo-os/release-readiness.md'),
    detail: 'Final release truth checklist with explicit caveat handling.',
  },
] as const;
