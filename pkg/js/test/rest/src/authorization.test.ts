import { accountPlatform, callerApi, studioClient } from "./caller";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  OAuthError,
  authorizationCodeCreate,
  authorizationCodeExchange,
  authorizationRequestValidate,
  claimsGet,
  credentialsGetSelf,
  credentialsList,
  refreshTokenGrant,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

// The S256 challenge and the verifier of RFC 7636's appendix B.
const codeChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM";
const codeVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";

// The redirect URI the test stack registers for studio.
const redirectURI = "https://studio.example.com/auth/callback";

const authorizationRequest = new URLSearchParams({
  response_type: "code",
  client_id: "studio",
  redirect_uri: redirectURI,
  state: "state",
  code_challenge: codeChallenge,
  code_challenge_method: "S256",
});

/** Signs the configured super-admin in on the account platform, opening a new session. */
async function superAdminSession() {
  return await tokenCreate(callerApi(accountPlatform), {
    email: process.env.SUPER_ADMIN_EMAIL!,
    password: process.env.SUPER_ADMIN_PASSWORD!,
  });
}

/**
 * Signs the super-admin in on studio as the platforms do: the account platform issues a code in a new session, and
 * studio exchanges it. Returns the session, the grant studio sent and the tokens it received.
 */
async function signInOnStudio() {
  const session = await superAdminSession();
  const { code } = await authorizationCodeCreate(callerApi(accountPlatform), session.accessToken, authorizationRequest);
  const grant = { code, code_verifier: codeVerifier, redirect_uri: redirectURI };
  const tokens = await authorizationCodeExchange(callerApi(studioClient), grant);

  return { session, grant, tokens };
}

/** Requires the token request to fail with an OAuth error response of code `error`. */
async function expectOAuthError(promise: Promise<unknown>, error: string) {
  const failure = await promise.then(
    () => undefined,
    (err: unknown) => err
  );

  expect(failure).toBeInstanceOf(OAuthError);
  expect((failure as OAuthError).status).toBe(400);
  expect((failure as OAuthError).response.error).toBe(error);
}

describe("authorizationRequestValidate", () => {
  it("accepts a registered client's request", async () => {
    const api = callerApi(accountPlatform);

    await expect(authorizationRequestValidate(api, authorizationRequest)).resolves.toStrictEqual({
      redirect_uri: redirectURI,
    });
  });

  it("refuses a caller other than the account platform", async () => {
    const api = callerApi(studioClient);

    await expectStatus(authorizationRequestValidate(api, authorizationRequest), 403);
  });
});

describe("authorizationCodeCreate", () => {
  it("issues a code for the person signed in", async () => {
    const api = callerApi(accountPlatform);
    const session = await superAdminSession();

    const authorization = await authorizationCodeCreate(api, session.accessToken, authorizationRequest);

    expect(authorization.code).toBeTruthy();
    expect(authorization.redirect_uri).toBe(redirectURI);
  });

  it("refuses a caller other than the account platform", async () => {
    const api = callerApi(studioClient);
    const session = await superAdminSession();

    await expectStatus(authorizationCodeCreate(api, session.accessToken, authorizationRequest), 403);
  });

  it("refuses an anonymous session", async () => {
    const api = callerApi(accountPlatform);
    const token = await tokenCreateAnon(api);

    await expectStatus(authorizationCodeCreate(api, token.accessToken, authorizationRequest), 403);
  });
});

describe("authorizationCodeExchange", () => {
  it("signs the person in on the client, in the code's session", async () => {
    const { session, tokens } = await signInOnStudio();

    expect(tokens.token_type).toBe("Bearer");
    expect(tokens.refresh_token).toBeTruthy();

    const claims = await claimsGet(callerApi(studioClient), tokens.access_token);
    const sessionClaims = await claimsGet(callerApi(accountPlatform), session.accessToken);
    expect(claims.userID).toBe(sessionClaims.userID);
    expect(claims.sessionID).toBe(sessionClaims.sessionID);
  });

  it("gives the client a token that names the person and manages no account", async () => {
    const api = callerApi(studioClient);
    const { tokens } = await signInOnStudio();

    const credentials = await credentialsGetSelf(api, tokens.access_token);
    expect(credentials.email).toBe(process.env.SUPER_ADMIN_EMAIL);

    // The super-admin lists accounts on the account platform, never through a client.
    await expectStatus(credentialsList(api, tokens.access_token, {}), 403);
  });

  it("ends the session when the code is exchanged twice", async () => {
    const api = callerApi(studioClient);
    const { session, grant, tokens } = await signInOnStudio();

    await expectOAuthError(authorizationCodeExchange(api, grant), "invalid_grant");

    await expectOAuthError(refreshTokenGrant(api, tokens.refresh_token!), "invalid_grant");
    await expectStatus(claimsGet(callerApi(accountPlatform), session.accessToken), 401);
  });

  it("refuses a caller outside the client registry", async () => {
    const api = callerApi(accountPlatform);

    await expectOAuthError(
      authorizationCodeExchange(api, { code: codeChallenge, code_verifier: codeVerifier, redirect_uri: redirectURI }),
      "invalid_client"
    );
  });
});

describe("refreshTokenGrant", () => {
  it("renews the client's access token in the same session", async () => {
    const api = callerApi(studioClient);
    const { tokens } = await signInOnStudio();

    const renewed = await refreshTokenGrant(api, tokens.refresh_token!);

    expect(renewed.token_type).toBe("Bearer");
    expect(renewed.refresh_token).toBeUndefined();
    expect(await claimsGet(api, renewed.access_token)).toStrictEqual(await claimsGet(api, tokens.access_token));
  });

  it("refuses a caller outside the client registry", async () => {
    const api = callerApi(accountPlatform);

    await expectOAuthError(refreshTokenGrant(api, "refresh-token"), "invalid_client");
  });
});
