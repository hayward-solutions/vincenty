import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  env: {
    // Injected at build time via Docker ARG / CI. Falls back to "dev" for
    // local development where the ARG is not supplied.
    NEXT_PUBLIC_APP_VERSION: process.env.NEXT_PUBLIC_APP_VERSION ?? "dev",
    // LiveKit WebSocket URL for real-time media (video calls, feeds, PTT).
    // Falls back to localhost for local development.
    NEXT_PUBLIC_LIVEKIT_URL:
      process.env.NEXT_PUBLIC_LIVEKIT_URL ?? "ws://localhost:7880",
  },
};

export default nextConfig;
