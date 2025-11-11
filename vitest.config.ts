import { builtinModules } from "node:module";
import { resolve } from "node:path";
import { defineConfig } from "vitest/config";

const NODE_BUILT_IN_MODULES = builtinModules.filter((m) => !m.startsWith("_"));
NODE_BUILT_IN_MODULES.push(...NODE_BUILT_IN_MODULES.map((m) => `node:${m}`));

export default defineConfig({
  optimizeDeps: {
    exclude: NODE_BUILT_IN_MODULES,
  },
  build: {
    minify: false,
  },
  test: {
    globals: true,
    environment: "jsdom",
    provide: {
      globalConfigValue: true,
    },
    alias: {
      "@a-novel/service-authentication-rest": resolve("./pkg/rest-js/src/index"),
      "@a-novel/service-authentication-rest-test": resolve("./pkg/rest-js-test/src/index"),
    },
    coverage: {
      enabled: true,
      clean: true,
      provider: "v8",
      reporter: ["text", "json", "html", "lcov"],
      reportsDirectory: "coverage",
      include: ["pkg/rest-js/src/**/*.{ts,tsx}"],
      allowExternal: true,
    },
    projects: [
      {
        root: "pkg/test/rest-js",
        extends: true,
      },
    ],
  },
});
