import path from "node:path";

import { Eslint } from "@a-novel-kit/nodelib-config";

import { defineConfig } from "eslint/config";

export default defineConfig(
  ...Eslint({
    gitIgnorePath: path.join(import.meta.dirname, ".gitignore"),
  })
);
