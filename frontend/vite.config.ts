import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9246,
    strictPort: true,
  },
  // A desktop app loads its bundle from disk, so one larger file is fine.
  build: { chunkSizeWarningLimit: 1500 },
  plugins: [react(), tailwindcss(), wails("./bindings")],
});
