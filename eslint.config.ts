import path from "node:path";

import { Eslint } from "@a-novel-kit/nodelib-config";

import { defineConfig } from "eslint/config";

export default defineConfig(
  ...Eslint({
    gitIgnorePath: path.join(import.meta.dirname, ".gitignore"),
  }),
  {
    files: ["scripts/waitlist/Code.js"],
    languageOptions: {
      sourceType: "script",
      globals: {
        PropertiesService: "readonly",
        Utilities: "readonly",
        LockService: "readonly",
        SpreadsheetApp: "readonly",
        Sheets: "readonly",
        ContentService: "readonly",
      },
    },
  }
);
