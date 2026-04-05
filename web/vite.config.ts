import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

function manualChunks(id: string): string | undefined {
  const normalizedID = id.replaceAll("\\", "/");

  if (!normalizedID.includes("/node_modules/")) {
    return undefined;
  }

  if (
    normalizedID.includes("/cytoscape/") ||
    normalizedID.includes("/cytoscape-dagre/")
  ) {
    return "graph-vendor";
  }

  if (normalizedID.includes("@chenglou/pretext")) {
    return "pretext-vendor";
  }

  if (
    normalizedID.includes("/react/") ||
    normalizedID.includes("/react-dom/") ||
    normalizedID.includes("/react-router-dom/")
  ) {
    return "react-vendor";
  }

  return "vendor";
}

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../cli/internal/webui/dist",
    emptyOutDir: true,
    sourcemap: false,
    chunkSizeWarningLimit: 600,
    rolldownOptions: {
      output: {
        entryFileNames: "assets/app.js",
        chunkFileNames: "assets/[name].js",
        manualChunks,
        assetFileNames: (assetInfo) => {
          if (assetInfo.names.some((name) => name.endsWith(".css"))) {
            return "assets/app.css";
          }
          return "assets/[name][extname]";
        },
      },
    },
  },
});
