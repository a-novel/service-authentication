import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  Role,
  TokenRefreshRequestSchema,
  claimsGet,
  tokenCreate,
  tokenCreateAnon,
  tokenRefresh,
  tokenRevoke,
} from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";

// The managed local test rail supplies a dynamic URL; legacy CI still exports MAIL_HOST.
const mailUrl = (() => {
  const value = process.env.MAIL_UI_URL ?? process.env.MAIL_HOST;
  if (!value) throw new Error("MAIL_UI_URL or MAIL_HOST must be set");
  return value;
})();

/** Signs the configured super-admin in, opening a new session. */
async function superAdminSession(api: AuthenticationApi) {
  return await tokenCreate(api, {
    email: process.env.SUPER_ADMIN_EMAIL!,
    password: process.env.SUPER_ADMIN_PASSWORD!,
  });
}

describe("tokenCreate", () => {
  it("logs in with correct credentials", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const res = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    expect(res.accessToken).toBeTruthy();

    const claims = await claimsGet(api, res.accessToken);

    expect(claims).toStrictEqual({
      userID: claims.userID,
      roles: [Role.SuperAdmin],
      refreshTokenID: claims.refreshTokenID,
    });
  });

  it("returns unauthorized when email does not exist", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(
      tokenCreate(api, {
        email: "does-not-exist@gmail.com",
        password: process.env.SUPER_ADMIN_PASSWORD!,
      }),
      401
    );
  });

  it("returns unauthorized when password is incorrect", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(
      tokenCreate(api, {
        email: process.env.SUPER_ADMIN_EMAIL!,
        password: "incorrect-password",
      }),
      401
    );
  });
});

describe("tokenCreateAnon", () => {
  it("logs in with generic credentials", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const res = await tokenCreateAnon(api);

    expect(res.accessToken).toBeTruthy();

    const claims = await claimsGet(api, res.accessToken);

    expect(claims).toStrictEqual({
      roles: [Role.Anon],
    });
  });
});

describe("tokenRefresh", () => {
  it("refreshes token once", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    expect(token.accessToken).toBeTruthy();
    expect(token.refreshToken).toBeTruthy();

    const originalClaims = await claimsGet(api, token.accessToken);

    const newToken = await tokenRefresh(api, {
      accessToken: token.accessToken,
      refreshToken: token.refreshToken!,
    });

    expect(newToken.accessToken).toBeTruthy();
    expect(newToken.accessToken).not.toBe(token.accessToken);
    expect(newToken.refreshToken).toBeTruthy();

    const newClaims = await claimsGet(api, newToken.accessToken);

    expect(newClaims).toStrictEqual(originalClaims);
  });

  it("refreshes token twice", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    expect(token.accessToken).toBeTruthy();
    expect(token.refreshToken).toBeTruthy();

    const originalClaims = await claimsGet(api, token.accessToken);

    const newToken = await tokenRefresh(api, {
      accessToken: token.accessToken,
      refreshToken: token.refreshToken!,
    });

    const newNewToken = await tokenRefresh(api, {
      accessToken: newToken.accessToken,
      refreshToken: newToken.refreshToken!,
    });

    const newClaims = await claimsGet(api, newNewToken.accessToken);

    expect(newClaims).toStrictEqual(originalClaims);
  });

  it("validates an issued token pair with its request schema", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    expect(TokenRefreshRequestSchema.safeParse(token).success).toBe(true);
    expect(TokenRefreshRequestSchema.safeParse({ ...token, accessToken: btoa("not a jwt") }).success).toBe(false);
  });

  it("can only use refresh token issued with original access token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const altToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    await expectStatus(
      tokenRefresh(api, {
        accessToken: token.accessToken,
        refreshToken: altToken.refreshToken!,
      }),
      403
    );
  });
});

describe("tokenRevoke", () => {
  it("stops the refresh token from renewing the session", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await superAdminSession(api);

    await expect(tokenRevoke(api, { refreshToken: token.refreshToken })).resolves.toBeUndefined();

    await expectStatus(tokenRefresh(api, token), 403);
  });

  it("succeeds again on a revoked refresh token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await superAdminSession(api);

    await tokenRevoke(api, { refreshToken: token.refreshToken });

    await expect(tokenRevoke(api, { refreshToken: token.refreshToken })).resolves.toBeUndefined();
  });

  it("leaves the user's other sessions untouched", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await superAdminSession(api);
    const otherToken = await superAdminSession(api);

    await tokenRevoke(api, { refreshToken: token.refreshToken });

    const renewed = await tokenRefresh(api, otherToken);
    expect(renewed.accessToken).toBeTruthy();
  });

  it("leaves another user's session untouched", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await superAdminSession(api);
    const otherUser = await registerUser(api, await preRegisterUser(api, mailUrl));

    await tokenRevoke(api, { refreshToken: token.refreshToken });

    const renewed = await tokenRefresh(api, otherUser.token);
    const claims = await claimsGet(api, renewed.accessToken);
    expect(claims.userID).toBe(otherUser.claims.userID);
  });

  it("returns forbidden for a token that is not a refresh token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    const token = await superAdminSession(api);

    await expectStatus(tokenRevoke(api, { refreshToken: token.accessToken }), 403);

    const renewed = await tokenRefresh(api, token);
    expect(renewed.accessToken).toBeTruthy();
  });

  it("returns unprocessable entity for a blank token", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(tokenRevoke(api, { refreshToken: "" }), 422);
  });
});
