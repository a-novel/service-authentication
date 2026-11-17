import manifest from "./package.json" with { type: "json" };

import { defineConfig } from "vite";

export default defineConfig({
  build: {
    outDir: "pkg/js/rest-test/dist",
    emptyOutDir: true,
    lib: {
      entry: {
        src: "pkg/js/rest-test/src/index.ts",
      },
      name: manifest.name,
      formats: ["es"],
      fileName: (format, entryName) => `${entryName}/index.${format}.js`,
    },
    sourcemap: true,
    rollupOptions: {
      // Node built-ins and peers, subpaths included, resolve from the consumer.
      external: (id) =>
        id.startsWith("node:") ||
        Object.keys(manifest.peerDependencies).some((peer) => id === peer || id.startsWith(`${peer}/`)),
    },
  },
});
