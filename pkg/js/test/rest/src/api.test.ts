import { accountPlatform, callerApi } from "./caller";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { AuthenticationApi } from "@a-novel/service-authentication-rest";

describe("ping", () => {
  it("returns success without a service token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    const api = callerApi(accountPlatform);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
      "client:smtp": { status: "up" },
      "api:jsonKeys": { status: "up" },
    });
  });

  it("returns unauthorized without a service token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    await expectStatus(api.health(), 401);
  });
});
