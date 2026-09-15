import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Served as static assets embedded in the quality-gate Go binary — no
  // Node server at runtime. The dashboard reads its data client-side from
  // /api/* endpoints served by that same Go binary.
  output: "export",
  images: { unoptimized: true },
};

export default nextConfig;
