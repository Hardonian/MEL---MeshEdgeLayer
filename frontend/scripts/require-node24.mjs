#!/usr/bin/env node

const [major] = process.versions.node.split('.').map((part) => Number.parseInt(part, 10))

if (major !== 24) {
  const got = process.version
  console.error(
    `[runtime-contract] frontend requires Node 24.x for deterministic MEL verification. Detected ${got}. ` +
      `Run '. ./scripts/dev-env.sh' from repo root (or nvm use 24) before npm install/test/build in ./frontend.`,
  )
  process.exit(1)
}
