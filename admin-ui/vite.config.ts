import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// The contract layer (../frontend/src) lives outside this app's root, so Vite's
// dev-server file-system guard needs to be told it's OK to serve it.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    fs: {
      allow: [path.resolve(__dirname, ".."), path.resolve(__dirname, "../frontend")],
    },
    // Neither cmd/agentgate nor cmd/g1-mock-authz sends CORS headers (they're
    // server-to-server/CLI-facing, not browser-facing). Proxying same-origin
    // through the dev server avoids needing CORS support in either binary.
    proxy: {
      "/proxy/agentgate": {
        target: "http://localhost:8090",
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/proxy\/agentgate/, ""),
      },
      "/proxy/mock-authz": {
        target: "http://localhost:8091",
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/proxy\/mock-authz/, ""),
      },
    },
  },
});
