import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:18080",
        changeOrigin: true,
      },
    },
  },
  plugins: [vue()],
});