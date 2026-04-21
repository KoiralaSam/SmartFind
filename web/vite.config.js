import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    // When exposing the dev server via ngrok, Vite will block unknown hosts by default.
    // Add your current ngrok domain here (or set VITE_ALLOWED_HOSTS as a comma-separated list).
    allowedHosts: [
      "unpolemical-spinier-chana.ngrok-free.dev",
      ...(process.env.VITE_ALLOWED_HOSTS || "")
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
    ],
    proxy: {
      // API gateway (auth + found-items + media uploads init)
      // Staff REST calls use `/gateway/staff/...` from `web/src/api/gateway.js`.
      // Do NOT proxy `/staff` here: React Router owns `/staff/*` (e.g. `/staff/dashboard`)
      // and full page loads must return the SPA; proxying `/staff` breaks reloads with 404.
      "/gateway": {
        target: process.env.VITE_API_BASE_URL || "http://localhost:8081",
        rewrite: (path) => path.replace(/^\/gateway/, ""),
        changeOrigin: true,
      },
      "/media": {
        target: process.env.VITE_API_BASE_URL || "http://localhost:8081",
        changeOrigin: true,
      },
      "/api/test-extract": {
        target: process.env.DETAIL_EXTRACTER_URL || "http://localhost:8091",
        rewrite: (path) =>
          path.replace(/^\/api\/test-extract/, "/test-extract"),
        changeOrigin: true,
      },
      "/api/extract": {
        target: process.env.VITE_API_BASE_URL || "http://localhost:8081",
        rewrite: (path) => path.replace(/^\/api\/extract/, "/extract"),
        changeOrigin: true,
      },
      "/api": {
        target: process.env.CHAT_AGENT_URL || "http://localhost:8090",
        rewrite: (path) => path.replace(/^\/api/, ""),
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
