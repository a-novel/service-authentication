import manifest from "./package.json" with { type: "json" };

import { ViteLibrary } from "@a-novel-kit/nodelib-config/vite-library";

export default ViteLibrary(import.meta.dirname, { src: "src/index.ts" }, manifest);
