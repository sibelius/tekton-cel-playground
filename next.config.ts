import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // In dev, proxy /api/evaluate to the local Go server (port 3002).
  // In production on Vercel, /api/evaluate is served by api/evaluate.go directly,
  // so no rewrite is needed.
  async rewrites() {
    if (process.env.NODE_ENV !== "development") return [];
    return [
      {
        source: "/api/evaluate",
        destination: "http://localhost:3002/api/evaluate",
      },
    ];
  },
};

export default nextConfig;
