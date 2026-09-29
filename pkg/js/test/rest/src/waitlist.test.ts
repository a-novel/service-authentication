import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  Lang,
  MAX_EMAIL_LENGTH,
  WaitlistJoinRequestSchema,
  tokenCreateAnon,
  waitlistJoin,
} from "@a-novel/service-authentication-rest";
import { generateRandomMail } from "@a-novel/service-authentication-rest-test";

describe("waitlistJoin", () => {
  it("quietly acknowledges an existing account without a Google deployment", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);
    await expect(
      waitlistJoin(api, token.accessToken, {
        email: process.env.SUPER_ADMIN_EMAIL!,
        lang: Lang.En,
      })
    ).resolves.toBeUndefined();
  });

  it("reports an unavailable writer without pretending to store a new request", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);
    await expectStatus(waitlistJoin(api, token.accessToken, { email: generateRandomMail(), lang: Lang.Fr }), 503);
  });

  it("requires an anonymous or authenticated session", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    await expectStatus(waitlistJoin(api, "", { email: generateRandomMail(), lang: Lang.En }), 401);
  });

  it.each(["invalid", "a".repeat(MAX_EMAIL_LENGTH + 1) + "@example.com"])("rejects invalid email %s", async (email) => {
    const form = { email, lang: Lang.En };
    expect(WaitlistJoinRequestSchema.safeParse(form).success).toBe(false);
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);
    await expectStatus(waitlistJoin(api, token.accessToken, form), 422);
  });
});
