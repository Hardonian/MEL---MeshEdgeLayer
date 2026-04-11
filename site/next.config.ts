import path from 'path';
import type { NextConfig } from 'next';

// Pin the Turbopack workspace root to this directory. A root-level
// package.json and package-lock.json exist solely so Vercel's framework
// detection sees "next" (see /package.json and /vercel.json); without this
// override, Turbopack walks up to the repo root and warns about "multiple
// lockfiles". Pinning the workspace root here silences the warning and keeps
// HMR and file watchers scoped to ./site.
const nextConfig: NextConfig = {
  reactStrictMode: true,
  turbopack: {
    root: path.resolve(__dirname),
  },
};

export default nextConfig;
