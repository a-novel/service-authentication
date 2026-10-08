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
        src: "pkg/js/rest-test/src/index.ts",
      },
      name,
      formats: ["es"],
      fileName: (format, entryName) =>
        entryName === "index" ? `${entryName}.${format}.js` : `${entryName}/index.${format}.js`,
    },
    sourcemap: true,
    rollupOptions: {
      // Peers and their subpaths, such as nodelib-browser/http, resolve from the consumer.
      external: (id) =>
        NODE_BUILT_IN_MODULES.includes(id) ||
        Object.keys(peerDependencies).some((peer) => id === peer || id.startsWith(`${peer}/`)),
    },
  },
});
