import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  env: {
    // Injected at build time via Docker ARG / CI. Falls back to "dev" for
    // local development where the ARG is not supplied.
    NEXT_PUBLIC_APP_VERSION: process.env.NEXT_PUBLIC_APP_VERSION ?? "dev",
  },
};

export default nextConfig;
