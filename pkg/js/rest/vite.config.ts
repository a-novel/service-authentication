import { name, peerDependencies } from "./package.json";

import { defineConfig } from "vite";

export default defineConfig({
  build: {
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
      external: Object.keys(peerDependencies),
    },
  },
});
