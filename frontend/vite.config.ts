import { fileURLToPath, URL } from "node:url";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [svelte()],
  // Compile Kit UI's Svelte/TypeScript sources through the normal pipeline.
  optimizeDeps: { exclude: ["@kenn-io/kit-ui"] },
  resolve: {
    alias: { $lib: fileURLToPath(new URL("./src/lib", import.meta.url)) },
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.HOMELAB_CONSOLE_FRONTEND_PORT || 5173),
    strictPort: true,
    proxy: {
      "/api": process.env.HOMELAB_CONSOLE_API_TARGET || "http://127.0.0.1:8080",
    },
  },
});
