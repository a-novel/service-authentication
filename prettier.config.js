const config = {
  trailingComma: "es5",
  tabWidth: 2,
  semi: true,
  singleQuote: false,
  printWidth: 120,
  language: "postgresql",
  paramTypes: `{ numbered: ["?"] }`,
  plugins: ["prettier-plugin-sql", "prettier-plugin-packagejson"],
};

export default config;
