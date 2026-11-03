import { accountPlatform, callerApi } from "./caller";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  claimsGet,
  credentialsGetSelf,
  credentialsUpdateLocationConsent,
  sessionDelete,
  sessionList,
  sessionRevoke,
  sessionRevokeOthers,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";

// The managed local test rail supplies a dynamic URL; legacy CI still exports MAIL_HOST.
const mailUrl = (() => {
  const value = process.env.MAIL_UI_URL ?? process.env.MAIL_HOST;
  if (!value) throw new Error("MAIL_UI_URL or MAIL_HOST must be set");
  return value;
})();

/** The visitor the account platform describes in the tests, as its gateway would. */
const visitor = {
  "X-Client-User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0",
  "X-Client-Region": "FR",
  "X-Client-City": "Paris",
};

/** Returns a client of the service that calls it as the account platform, describing {@link visitor}. */
function visitorApi(): AuthenticationApi {
  const api = callerApi(accountPlatform);

  return new AuthenticationApi("", async (input, init) => {
    const headers = new Headers(init?.headers);
    Object.entries(visitor).forEach(([name, value]) => headers.set(name, value));

    return await api.fetchResponse(String(input), { ...init, headers });
  });
}

/**
 * Registers a new account, and signs it in a second time. Revoking sessions of the shared super-admin would sign out
 * the other suites, so each test works on an account of its own.
 */
async function userWithTwoSessions() {
  const api = visitorApi();
  const user = await registerUser(api, await preRegisterUser(callerApi(accountPlatform), mailUrl));
  const second = await tokenCreate(api, { email: user.email, password: user.password });

  return { api, user, first: user.token, second };
}

describe("sessionList", () => {
  it("lists the account's devices, and marks the caller's own", async () => {
    const { api, first, second } = await userWithTwoSessions();

    const sessions = await sessionList(api, second.accessToken);
    expect(sessions).toHaveLength(2);

    const secondClaims = await claimsGet(api, second.accessToken);
    const current = sessions.filter((session) => session.current);
    expect(current).toHaveLength(1);
    expect(current[0].id).toBe(secondClaims.sessionID);

    const firstClaims = await claimsGet(api, first.accessToken);
    expect(sessions.map((session) => session.id)).toContain(firstClaims.sessionID);

    // A new account consents to no location, so the sessions record the user agent alone.
    for (const session of sessions) {
      expect(session.loggedOut).toBe(false);
      expect(session.userAgent).toBe(visitor["X-Client-User-Agent"]);
      expect(session.region).toBeUndefined();
      expect(session.city).toBeUndefined();
      expect(session.ipAddress).toBeUndefined();
    }
  });

  it("refuses an anonymous token", async () => {
    const api = callerApi(accountPlatform);
    const anonymousToken = await tokenCreateAnon(api);

    await expectStatus(sessionList(api, anonymousToken.accessToken), 403);
  });
});

describe("sessionRevoke", () => {
  it("signs another device out, and keeps it listed as logged out", async () => {
    const { api, first, second } = await userWithTwoSessions();
    const firstClaims = await claimsGet(api, first.accessToken);

    await sessionRevoke(api, second.accessToken, { id: firstClaims.sessionID! });

    await expectStatus(claimsGet(api, first.accessToken), 401);

    // The active session comes first.
    const sessions = await sessionList(api, second.accessToken);
    expect(sessions.map(({ current, loggedOut }) => ({ current, loggedOut }))).toStrictEqual([
      { current: true, loggedOut: false },
      { current: false, loggedOut: true },
    ]);
    expect(sessions[1].id).toBe(firstClaims.sessionID);

    // Signing it out again succeeds.
    await sessionRevoke(api, second.accessToken, { id: firstClaims.sessionID! });
  });

  it("does not reach another account's device", async () => {
    const { api, first } = await userWithTwoSessions();
    const other = await userWithTwoSessions();
    const otherClaims = await claimsGet(api, other.first.accessToken);

    await expectStatus(sessionRevoke(api, first.accessToken, { id: otherClaims.sessionID! }), 404);
    await claimsGet(api, other.first.accessToken);
  });

  it("returns 404 for an unknown session", async () => {
    const { api, first } = await userWithTwoSessions();

    await expectStatus(sessionRevoke(api, first.accessToken, { id: "00000000-0000-0000-0000-000000000000" }), 404);
  });
});

describe("sessionRevokeOthers", () => {
  it("signs every other device out, and keeps the caller's", async () => {
    const { api, first, second } = await userWithTwoSessions();

    await sessionRevokeOthers(api, second.accessToken);

    await expectStatus(claimsGet(api, first.accessToken), 401);
    await claimsGet(api, second.accessToken);

    const sessions = await sessionList(api, second.accessToken);
    expect(sessions.map(({ current, loggedOut }) => ({ current, loggedOut }))).toStrictEqual([
      { current: true, loggedOut: false },
      { current: false, loggedOut: true },
    ]);
  });
});

describe("sessionDelete", () => {
  it("removes a logged-out device, which stays signed out", async () => {
    const { api, first, second } = await userWithTwoSessions();
    const firstClaims = await claimsGet(api, first.accessToken);

    await sessionRevoke(api, second.accessToken, { id: firstClaims.sessionID! });
    await sessionDelete(api, second.accessToken, { id: firstClaims.sessionID! });

    const sessions = await sessionList(api, second.accessToken);
    expect(sessions.map((session) => session.id)).not.toContain(firstClaims.sessionID);
    await expectStatus(claimsGet(api, first.accessToken), 401);
  });

  it("refuses to remove an active device", async () => {
    const { api, first, second } = await userWithTwoSessions();
    const firstClaims = await claimsGet(api, first.accessToken);

    await expectStatus(sessionDelete(api, second.accessToken, { id: firstClaims.sessionID! }), 409);
    await claimsGet(api, first.accessToken);
  });

  it("does not reach another account's device", async () => {
    const { api, first } = await userWithTwoSessions();
    const other = await userWithTwoSessions();
    const otherClaims = await claimsGet(api, other.first.accessToken);

    await sessionRevoke(api, other.second.accessToken, { id: otherClaims.sessionID! });

    await expectStatus(sessionDelete(api, first.accessToken, { id: otherClaims.sessionID! }), 404);
  });
});

describe("credentialsUpdateLocationConsent", () => {
  it("records the location under consent, and erases it when the consent ends", async () => {
    const { api, user, second } = await userWithTwoSessions();

    await credentialsUpdateLocationConsent(api, second.accessToken, { enabled: true });
    expect((await credentialsGetSelf(api, second.accessToken)).locationConsent).toBe(true);

    const third = await tokenCreate(api, { email: user.email, password: user.password });
    const located = (await sessionList(api, third.accessToken)).find((session) => session.current);
    expect(located?.region).toBe(visitor["X-Client-Region"]);
    expect(located?.city).toBe(visitor["X-Client-City"]);
    expect(located?.ipAddress).toBeTruthy();

    await credentialsUpdateLocationConsent(api, third.accessToken, { enabled: false });
    expect((await credentialsGetSelf(api, third.accessToken)).locationConsent).toBe(false);

    for (const session of await sessionList(api, third.accessToken)) {
      expect(session.region).toBeUndefined();
      expect(session.city).toBeUndefined();
      expect(session.ipAddress).toBeUndefined();
    }
  });
});
