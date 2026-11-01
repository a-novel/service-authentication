import type { AuthenticationApi } from "./api";
import { type Token, TokenSchema } from "./token";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** A base64url value without padding, 43 characters long: 256 bits, such as a SHA-256 digest. */
const Base64Url256Schema = z.string().regex(/^[A-Za-z0-9_-]{43}$/);

/**
 * The client of an OAuth 2.0 authorization request (RFC 6749 §4.1.1), under its standard parameter names: the
 * client ID, and the redirect URI the client registry must hold for it, compared as exact strings.
 */
export const AuthorizationRequestSchema = z.object({
  client_id: z.string().min(1).max(256),
  redirect_uri: z.string().min(1).max(2048),
});

/** The value {@link AuthorizationRequestSchema} validates. */
export type AuthorizationRequest = z.infer<typeof AuthorizationRequestSchema>;

/**
 * An authorization request with its PKCE challenge (RFC 7636 §4.3): the base64url SHA-256 of the client's code
 * verifier. `S256` is the only method the service accepts.
 */
export const AuthorizationCodeCreateRequestSchema = AuthorizationRequestSchema.extend({
  code_challenge: Base64Url256Schema,
  code_challenge_method: z.literal("S256"),
});

/** The value {@link AuthorizationCodeCreateRequestSchema} validates. */
export type AuthorizationCodeCreateRequest = z.infer<typeof AuthorizationCodeCreateRequestSchema>;

/** A one-time authorization code, valid one minute, for the client and redirect URI it was issued to. */
export const AuthorizationCodeSchema = z.object({
  code: Base64Url256Schema,
});

/** The value {@link AuthorizationCodeSchema} validates. */
export type AuthorizationCode = z.infer<typeof AuthorizationCodeSchema>;

/**
 * A client's code exchange (RFC 6749 §4.1.3): the code its redirect URI received, the PKCE verifier it drew (RFC
 * 7636 §4.1), and that redirect URI.
 */
export const AuthorizationCodeExchangeRequestSchema = z.object({
  code: Base64Url256Schema,
  code_verifier: z.string().min(43).max(128),
  redirect_uri: z.string().min(1).max(2048),
});

/** The value {@link AuthorizationCodeExchangeRequestSchema} validates. */
export type AuthorizationCodeExchangeRequest = z.infer<typeof AuthorizationCodeExchangeRequestSchema>;

/**
 * Checks, for the account platform, that an authorization request names a registered client and its redirect URI.
 * Show an error page and redirect nowhere when it throws: 404 for an unknown client or another redirect URI, 403
 * when the caller is not the account platform.
 */
export async function authorizationRequestValidate(
  api: AuthenticationApi,
  request: AuthorizationRequest
): Promise<void> {
  const params = new URLSearchParams(request);

  return await api.fetchVoid(`/v2/authorization-request?${params.toString()}`, { method: "GET" });
}

/**
 * Issues, for the account platform, a code that signs the bearer of `accessToken` in on a client, within their
 * session. Redirect to the client's redirect URI with the code, the request's `state` and the account platform's
 * `iss`. Throws 401 when the session ended, 403 when the caller is not the account platform, and 404 as
 * {@link authorizationRequestValidate} does.
 */
export async function authorizationCodeCreate(
  api: AuthenticationApi,
  accessToken: string,
  form: AuthorizationCodeCreateRequest
): Promise<AuthorizationCode> {
  return await api.fetch("/v2/authorization-code", AuthorizationCodeSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/**
 * Trades, for a client, an authorization code for a token pair in the session it was issued in. The code
 * exchanges once: a second exchange throws 403 and signs the person out everywhere. Every other refusal throws 403
 * too, and so does a caller outside the client registry. Bring the person back to the signed-out page with a retry.
 */
export async function authorizationCodeExchange(
  api: AuthenticationApi,
  form: AuthorizationCodeExchangeRequest
): Promise<Token> {
  return await api.fetch("/v2/session/authorization-code", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
