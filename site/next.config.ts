import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Disable Turbopack to avoid workspace root detection issues in monorepo
  // turbopack: {
  //   root: __dirname,
  // },
};

export default nextConfig;
