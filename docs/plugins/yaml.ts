import type { Plugin } from "vite";

import { readFileSync } from "node:fs";
import { parse } from "yaml";

// Most plugins for loading yaml in vite use the js-yaml package, which has not receive updates for a long time. This
// is a plugin for loading yaml files using eemeli/yaml.

const yamlLoader: Plugin = {
  name: "yaml",

  load(id) {
    if (id.endsWith(".yaml")) {
      const content = readFileSync(id, "utf8");
      const yaml = parse(content);

      return {
        code: `export default ${JSON.stringify(yaml)}`,
        map: { mappings: "" },
      };
    }
  },
};

export default yamlLoader;
