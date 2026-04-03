#!/usr/bin/env node

const [major] = process.versions.node.split('.').map((part) => Number.parseInt(part, 10))

if (major !== 24) {
  const got = process.version
  console.error(
    `[runtime-contract] frontend requires Node 24.x for deterministic MEL verification. Detected ${got}. ` +
      `Use Node 24 before running npm install/test/build in ./frontend.`,
  )
  process.exit(1)
}
