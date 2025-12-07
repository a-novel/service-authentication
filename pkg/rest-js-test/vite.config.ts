import { name, peerDependencies } from "./package.json";

import { builtinModules } from "node:module";

import { defineConfig } from "vite";

const NODE_BUILT_IN_MODULES = builtinModules.filter((m) => !m.startsWith("_"));
NODE_BUILT_IN_MODULES.push(...NODE_BUILT_IN_MODULES.map((m) => `node:${m}`));

export default defineConfig({
  optimizeDeps: {
    exclude: NODE_BUILT_IN_MODULES,
  },
  build: {
    lib: {
      entry: {
        src: "pkg/rest-js-test/src/index.ts",
      },
      name,
      formats: ["es"],
      fileName: (format, entryName) =>
        entryName === "index" ? `${entryName}.${format}.js` : `${entryName}/index.${format}.js`,
    },
    sourcemap: true,
    rollupOptions: {
      external: [...Object.keys(peerDependencies), ...NODE_BUILT_IN_MODULES],
    },
  },
});
