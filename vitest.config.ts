import { resolve } from "node:path";

import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    environment: "jsdom",
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
        extends: true,
        test: {
          name: "waitlist-writer",
          environment: "node",
          include: ["scripts/waitlist/*.test.ts"],
        },
      },
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
