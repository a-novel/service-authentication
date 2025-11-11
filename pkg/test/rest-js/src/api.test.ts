import { AuthenticationApi } from "@a-novel/service-authentication-rest";

import { describe, it, expect } from "vitest";

describe("ping", () => {
  it("returns success", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
      "client:smtp": { status: "up" },
      "api:jsonKeys": { status: "up" },
    });
  });
});
