import { describe, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  authorizationCodeCreate,
  authorizationCodeExchange,
  authorizationRequestValidate,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

// The test stack verifies no service token, so its requests carry no caller, and the routes that only the account
// platform or a client may call refuse them. The Go handler and core tests cover the callers.

const authorizationRequest = {
  client_id: "studio",
  redirect_uri: "https://studio.example.com/auth/callback",
};

// The S256 challenge and the verifier of RFC 7636's appendix B.
const codeChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM";
const codeVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";

describe("authorizationRequestValidate", () => {
  it("refuses a caller other than the account platform", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(authorizationRequestValidate(api, authorizationRequest), 403);
  });

  it("returns 422 without a client ID", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(authorizationRequestValidate(api, { ...authorizationRequest, client_id: "" }), 422);
  });
});

describe("authorizationCodeCreate", () => {
  it("refuses a caller other than the account platform", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    await expectStatus(
      authorizationCodeCreate(api, token.accessToken, {
        ...authorizationRequest,
        code_challenge: codeChallenge,
        code_challenge_method: "S256",
      }),
      403
    );
  });

  it("refuses an anonymous session", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);
    const token = await tokenCreateAnon(api);

    await expectStatus(
      authorizationCodeCreate(api, token.accessToken, {
        ...authorizationRequest,
        code_challenge: codeChallenge,
        code_challenge_method: "S256",
      }),
      403
    );
  });
});

describe("authorizationCodeExchange", () => {
  it("refuses a caller outside the client registry", async () => {
    const api = new AuthenticationApi(process.env.REST_URL!);

    await expectStatus(
      authorizationCodeExchange(api, {
        code: codeChallenge,
        code_verifier: codeVerifier,
        redirect_uri: authorizationRequest.redirect_uri,
      }),
      403
    );
  });
});
