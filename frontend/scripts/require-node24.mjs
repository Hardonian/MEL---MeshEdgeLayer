#!/usr/bin/env node

const [major] = process.versions.node.split('.').map((part) => Number.parseInt(part, 10))

if (major < 20) {
  console.error(
    `[runtime-contract] MEL frontend requires Node.js 20 or later. Detected ${process.version}. ` +
      `Please upgrade Node.js to 20.x or later.`,
  )
  process.exit(1)
}

if (major === 20) {
  console.warn(
    `[runtime-contract] MEL frontend was tested with Node 24.x. Detected ${process.version}. ` +
      `Node 20 may work but Node 24+ is recommended for full verification parity.`,
  )
}

if (major >= 24) {
  console.info(`[runtime-contract] Node ${process.version} meets recommended version requirement.`)
}
