import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      "/api/test-extract": {
        target: process.env.DETAIL_EXTRACTER_URL || "http://localhost:8091",
        rewrite: (path) => path.replace(/^\/api\/test-extract/, "/test-extract"),
        changeOrigin: true,
      },
      "/api/extract": {
        target: process.env.DETAIL_EXTRACTER_URL || "http://localhost:8091",
        rewrite: (path) => path.replace(/^\/api\/extract/, "/extract"),
        changeOrigin: true,
      },
      "/api": {
        target: process.env.CHAT_AGENT_URL || "http://localhost:8090",
        rewrite: (path) => path.replace(/^\/api/, ""),
        changeOrigin: true,
      },
      "/passenger": {
        target: process.env.VITE_API_BASE_URL || "http://localhost:8081",
        changeOrigin: true,
      },
    },
  },
});
