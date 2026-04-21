import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");

  return {
    plugins: [react()],
    server: {
      host: true,
      port: 5173,
      // When exposing the dev server via ngrok, Vite will block unknown hosts by default.
      // Add your current ngrok domain here (or set VITE_ALLOWED_HOSTS as a comma-separated list).
      allowedHosts: [
        "unpolemical-spinier-chana.ngrok-free.dev",
        ...(env.VITE_ALLOWED_HOSTS || "")
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
          target: env.VITE_API_BASE_URL || "http://localhost:8081",
          rewrite: (path) => path.replace(/^\/gateway/, ""),
          changeOrigin: true,
        },
        "/media": {
          target: env.VITE_API_BASE_URL || "http://localhost:8081",
          changeOrigin: true,
        },
        "/api/test-extract": {
          target: env.DETAIL_EXTRACTER_URL || "http://localhost:8091",
          rewrite: (path) => path.replace(/^\/api\/test-extract/, "/test-extract"),
          changeOrigin: true,
        },
        "/api/extract": {
          target: env.VITE_API_BASE_URL || "http://localhost:8081",
          rewrite: (path) => path.replace(/^\/api\/extract/, "/extract"),
          changeOrigin: true,
        },
        "/analytics": {
          target: env.ANALYTICS_URL || "http://localhost:8092",
          changeOrigin: true,
        },
        "/api": {
          target: env.CHAT_AGENT_URL || "http://localhost:8090",
          rewrite: (path) => path.replace(/^\/api/, ""),
          changeOrigin: true,
          ws: true,
        },
      },
    },
  };
});
