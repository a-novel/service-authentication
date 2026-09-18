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
    env: {
      SUPER_ADMIN_EMAIL: process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com",
      SUPER_ADMIN_PASSWORD: process.env.SUPER_ADMIN_PASSWORD ?? "admin",
    },
    provide: {
      globalConfigValue: true,
    },
    alias: {
      "@a-novel/service-authentication-rest": resolve("./pkg/js/rest/src/index"),
      "@a-novel/service-authentication-rest-test": resolve("./pkg/js/rest-test/src/index"),
    },
    coverage: {
      enabled: true,
      clean: true,
      provider: "v8",
      reporter: ["text", "json", "html", "lcov"],
      reportsDirectory: "coverage",
      include: ["pkg/js/rest/src/**/*.{ts,tsx}"],
      allowExternal: true,
    },
    projects: [
      {
        root: "pkg/js/test/rest",
        extends: true,
        test: {
          name: "rest-integration",
          globalSetup: [resolve("./pkg/js/test/rest/src/setup.ts")],
        },
      },
    ],
  },
});
