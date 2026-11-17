import { accountPlatform, callerApi } from "./caller";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Role,
  TokenRefreshRequestSchema,
  claimsGet,
  tokenCreate,
  tokenCreateAnon,
  tokenRefresh,
  tokenRevoke,
} from "@a-novel/service-authentication-rest";
import { registerUser, superAdminCredentials } from "@a-novel/service-authentication-rest-test";

describe("tokenCreate", () => {
  it("logs in with correct credentials", async () => {
    const api = callerApi(accountPlatform);

    const res = await tokenCreate(api, superAdminCredentials());

    expect(res.accessToken).toBeTruthy();

    const claims = await claimsGet(api, res.accessToken);

    expect(claims).toStrictEqual({
      userID: claims.userID,
      roles: [Role.SuperAdmin],
      refreshTokenID: claims.refreshTokenID,
      // A sign-in opens a session of its own, named after its refresh token, and ending with it.
      sessionID: claims.refreshTokenID,
      sessionExp: jwtPayload(res.refreshToken).exp,
    });
  });

  it("signs the session's end into the access token", async () => {
    const api = callerApi(accountPlatform);

    const res = await tokenCreate(api, superAdminCredentials());

    // A sign-in opens its own session, which ends with its refresh token. JSON Keys must keep the claim
    // the session checks read.
    expect(jwtPayload(res.accessToken).sessionExp).toBe(jwtPayload(res.refreshToken).exp);
  });

  it("returns unauthorized when email does not exist", async () => {
    const api = callerApi(accountPlatform);

    await expectStatus(tokenCreate(api, { ...superAdminCredentials(), email: "does-not-exist@gmail.com" }), 401);
  });

  it("returns unauthorized when password is incorrect", async () => {
    const api = callerApi(accountPlatform);

    await expectStatus(tokenCreate(api, { ...superAdminCredentials(), password: "incorrect-password" }), 401);
  });
});

describe("tokenCreateAnon", () => {
  it("logs in with generic credentials", async () => {
    const api = callerApi(accountPlatform);

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
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

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
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

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
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    expect(TokenRefreshRequestSchema.safeParse(token).success).toBe(true);
    expect(TokenRefreshRequestSchema.safeParse({ ...token, accessToken: btoa("not a jwt") }).success).toBe(false);
  });

  it("can only use refresh token issued with original access token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    const altToken = await tokenCreate(api, superAdminCredentials());

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
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    await expect(tokenRevoke(api, token.accessToken, { refreshToken: token.refreshToken })).resolves.toBeUndefined();

    await expectStatus(tokenRefresh(api, token), 403);
  });

  it("refuses every access token of the session", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());
    const renewed = await tokenRefresh(api, token);

    await tokenRevoke(api, renewed.accessToken, { refreshToken: renewed.refreshToken });

    await expectStatus(claimsGet(api, token.accessToken), 401);
    await expectStatus(claimsGet(api, renewed.accessToken), 401);
  });

  it("succeeds again on a revoked refresh token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    await tokenRevoke(api, token.accessToken, { refreshToken: token.refreshToken });

    await expect(tokenRevoke(api, token.accessToken, { refreshToken: token.refreshToken })).resolves.toBeUndefined();
  });

  it("leaves the user's other sessions untouched", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());
    const otherToken = await tokenCreate(api, superAdminCredentials());

    await tokenRevoke(api, token.accessToken, { refreshToken: token.refreshToken });

    const renewed = await tokenRefresh(api, otherToken);
    expect(renewed.accessToken).toBeTruthy();

    const claims = await claimsGet(api, otherToken.accessToken);
    expect(claims.sessionID).toBe(claims.refreshTokenID);
  });

  it("refuses another user's refresh token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());
    const otherUser = await registerUser(api);

    await expectStatus(tokenRevoke(api, token.accessToken, { refreshToken: otherUser.token.refreshToken }), 403);

    const renewed = await tokenRefresh(api, otherUser.token);
    const claims = await claimsGet(api, renewed.accessToken);
    expect(claims.userID).toBe(otherUser.claims.userID);
  });

  it("returns unauthorized without an access token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    await expectStatus(tokenRevoke(api, "", { refreshToken: token.refreshToken }), 401);

    const renewed = await tokenRefresh(api, token);
    expect(renewed.accessToken).toBeTruthy();
  });

  it("returns forbidden for an anonymous session", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());
    const anonToken = await tokenCreateAnon(api);

    await expectStatus(tokenRevoke(api, anonToken.accessToken, { refreshToken: token.refreshToken }), 403);
  });

  it("returns forbidden for a token that is not a refresh token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    await expectStatus(tokenRevoke(api, token.accessToken, { refreshToken: token.accessToken }), 403);

    const renewed = await tokenRefresh(api, token);
    expect(renewed.accessToken).toBeTruthy();
  });

  it("returns unprocessable entity for a blank token", async () => {
    const api = callerApi(accountPlatform);

    const token = await tokenCreate(api, superAdminCredentials());

    await expectStatus(tokenRevoke(api, token.accessToken, { refreshToken: "" }), 422);
  });
});

/** Decodes the payload of a compact JWT, without verifying it. */
function jwtPayload(token: string): Record<string, unknown> {
  return JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString());
}
