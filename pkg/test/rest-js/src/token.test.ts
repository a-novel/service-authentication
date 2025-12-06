import { describe, it, expect } from "vitest";

import { expectStatus } from "@a-novel/nodelib-test/http";
import {
  AuthenticationApi,
  claimsGet,
  Role,
  tokenCreate,
  tokenCreateAnon,
  tokenRefresh,
} from "@a-novel/service-authentication-rest";

describe("tokenCreate", () => {
  it("logs in with correct credentials", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

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

  it("returns not found when email does not exist", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    await expectStatus(
      tokenCreate(api, {
        email: "does-not-exist@gmail.com",
        password: process.env.SUPER_ADMIN_PASSWORD!,
      }),
      404
    );
  });

  it("returns forbidden when password is incorrect", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    await expectStatus(
      tokenCreate(api, {
        email: process.env.SUPER_ADMIN_EMAIL!,
        password: "incorrect-password",
      }),
      403
    );
  });
});

describe("tokenCreateAnon", () => {
  it("logs in with generic credentials", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

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
    const api = new AuthenticationApi(process.env.API_URL!);

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
    const api = new AuthenticationApi(process.env.API_URL!);

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

  it("can only use refresh token issued with original access token", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

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
