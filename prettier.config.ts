import type { Config } from "prettier";

const config: Config = {
  trailingComma: "es5",
  tabWidth: 2,
  semi: true,
  singleQuote: false,
  printWidth: 120,
  importOrder: [
    // Test imports are always first.
    "^#/",
    // Static assets.
    "^(.*).(css|yaml|json)$",
    "^(.*).(svg|png|jpg|jpeg)$",
    // App imports.
    "^@/",
    "^[./]",
    // A-novel imports.
    "^@a-novel(/(.*))?$",
    // Important.
    "^react(/(.*))?$",
    "^vite(/(.*))?$",
    // Other.
    "<THIRD_PARTY_MODULES>",
  ],
  importOrderSeparation: true,
  htmlWhitespaceSensitivity: "strict",
  language: "postgresql",
  paramTypes: `{ numbered: ["?"] }`,
  plugins: ["@trivago/prettier-plugin-sort-imports", "prettier-plugin-sql", "prettier-plugin-packagejson"],
};

export default config;
