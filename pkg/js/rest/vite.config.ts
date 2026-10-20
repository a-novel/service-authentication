import { name, peerDependencies } from "./package.json" with { type: "json" };

import { defineConfig } from "vite";

export default defineConfig({
  build: {
    outDir: "pkg/js/rest/dist",
    emptyOutDir: true,
    lib: {
      entry: {
        src: "pkg/js/rest/src/index.ts",
      },
      name,
      formats: ["es"],
      fileName: (format, entryName) =>
        entryName === "index" ? `${entryName}.${format}.js` : `${entryName}/index.${format}.js`,
    },
    sourcemap: true,
    rollupOptions: {
      // Peers and their subpaths, such as nodelib-browser/http, resolve from the consumer, so errors
      // keep the consumer's HttpError instead of a stale bundled copy.
      external: (id) => Object.keys(peerDependencies).some((peer) => id === peer || id.startsWith(`${peer}/`)),
    },
  },
});
