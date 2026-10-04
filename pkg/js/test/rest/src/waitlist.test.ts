import { describe, expect, it, vi } from "vitest";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";
import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  Lang,
  MAX_EMAIL_LENGTH,
  WaitlistJoinConflictError,
  WaitlistJoinRequestSchema,
  tokenCreateAnon,
  waitlistJoin,
} from "@a-novel/service-authentication-rest";
import { generateRandomMail } from "@a-novel/service-authentication-rest-test";

describe("waitlistJoin", () => {
  it("exposes the duplicate warning returned by a configured writer", async () => {
    const api = new AuthenticationApi("https://auth.example.test");
    vi.spyOn(api, "fetchResponse").mockResolvedValue(Response.json({ code: "already_waitlisted" }, { status: 409 }));
    const request = waitlistJoin(api, "test-session", { email: "member@example.com", lang: Lang.En });
    await expect(request).rejects.toBeInstanceOf(WaitlistJoinConflictError);
    await expect(request).rejects.toMatchObject({ status: 409, code: "already_waitlisted" });
  });

  it("accepts an empty acknowledgement without trying to decode JSON", async () => {
    const api = new AuthenticationApi("https://auth.example.test");
    vi.spyOn(api, "fetchResponse").mockResolvedValue(new Response(null, { status: 202 }));
    await expect(
      waitlistJoin(api, "test-session", { email: "member@example.com", lang: Lang.Fr })
    ).resolves.toBeUndefined();
  });

  it("rejects an unknown conflict code instead of presenting a misleading membership warning", async () => {
    const api = new AuthenticationApi("https://auth.example.test");
    vi.spyOn(api, "fetchResponse").mockResolvedValue(Response.json({ code: "unknown" }, { status: 409 }));
    await expect(
      waitlistJoin(api, "test-session", { email: "member@example.com", lang: Lang.En })
    ).rejects.toMatchObject({
      name: "ZodError",
    });
  });

  it("reports an existing account with a typed conflict without contacting Google", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);
    const request = waitlistJoin(api, token.accessToken, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      lang: Lang.En,
    });
    await expect(request).rejects.toBeInstanceOf(WaitlistJoinConflictError);
    await expect(request).rejects.toMatchObject({ status: 409, code: "account_exists" });
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

  it("names the invalid fields of a rejected form", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);
    const response = await api.fetchResponse("/v2/waitlist", {
      headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${token.accessToken}` },
      method: "PUT",
      body: JSON.stringify({ email: "invalid", lang: "xx" }),
    });

    expect(response.status).toBe(422);
    expect(response.headers.get("Content-Type")).toBe("application/problem+json");
    expect(await response.json()).toEqual({
      type: "about:blank",
      title: "Unprocessable Entity",
      status: 422,
      tags: { invalidFields: { email: "email", lang: "langs" } },
    });
  });
});
