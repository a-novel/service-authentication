import type { AuthenticationApi } from "./api";

import { HttpError, decodeHttpResponse, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** The media type of an OAuth 2.0 request body (RFC 6749 Appendix B). */
const FORM_HEADERS = { "Content-Type": "application/x-www-form-urlencoded" };

/**
 * An OAuth 2.0 error response (RFC 6749 §4.1.2.1, §5.2). From the authorization routes, `redirect_uri` is where the
 * account platform sends the error, with the request's `state` and its own `iss`; without it, the account platform
 * shows the error and redirects nowhere.
 */
export const OAuthErrorResponseSchema = z.object({
  error: z.string(),
  error_description: z.string().optional(),
  redirect_uri: z.string().optional(),
});

/** The value {@link OAuthErrorResponseSchema} validates. */
export type OAuthErrorResponse = z.infer<typeof OAuthErrorResponseSchema>;

/**
 * An HTTP 400 that carries an OAuth 2.0 error response, such as `invalid_grant`. It keeps the name HttpError, so the
 * shared HttpError status checks match it too; `instanceof OAuthError` tells it apart.
 */
export class OAuthError extends HttpError {
  /** The error response the server answered. */
  readonly response: OAuthErrorResponse;

  constructor(text: string, response: OAuthErrorResponse) {
    super(400, text);
    this.response = response;
  }
}

/** An authorization request the server accepts: where its response goes once the person signed in. */
export const AuthorizationRequestSchema = z.object({
  redirect_uri: z.string(),
});

/** The value {@link AuthorizationRequestSchema} validates. */
export type AuthorizationRequest = z.infer<typeof AuthorizationRequestSchema>;

/**
 * An authorization code, valid one minute for the client and redirect URI it was issued to, and the redirect URI to
 * send it to (RFC 6749 §4.1.2).
 */
export const AuthorizationCodeSchema = z.object({
  code: z.string(),
  redirect_uri: z.string(),
});

/** The value {@link AuthorizationCodeSchema} validates. */
export type AuthorizationCode = z.infer<typeof AuthorizationCodeSchema>;

/**
 * A successful token response (RFC 6749 §5.1). A refresh carries no `refresh_token`: the client keeps its own. Members
 * the schema does not name are ignored, as the RFC asks of a client.
 */
export const TokenResponseSchema = z.object({
  access_token: z.string(),
  token_type: z.string(),
  expires_in: z.number().int().positive(),
  refresh_token: z.string().optional(),
});

/** The value {@link TokenResponseSchema} validates. */
export type TokenResponse = z.infer<typeof TokenResponseSchema>;

/**
 * A code grant (RFC 6749 §4.1.3, RFC 7636 §4.5): the code, its PKCE verifier of 43 to 128 unreserved characters, and
 * the redirect URI, required when the authorization request named it.
 */
export const AuthorizationCodeGrantSchema = z.object({
  code: z.string(),
  code_verifier: z.string().regex(/^[A-Za-z0-9._~-]{43,128}$/),
  redirect_uri: z.string().optional(),
});

/** The value {@link AuthorizationCodeGrantSchema} validates. */
export type AuthorizationCodeGrant = z.infer<typeof AuthorizationCodeGrantSchema>;

/**
 * Checks, for the account platform, the authorization request its authorization endpoint received, given as that
 * request's query parameters. Throws an {@link OAuthError} for a refused request, whose `redirect_uri` says whether to
 * redirect it, and an HttpError 403 when the caller is not the account platform.
 */
export async function authorizationRequestValidate(
  api: AuthenticationApi,
  params: URLSearchParams
): Promise<AuthorizationRequest> {
  return await fetchOAuth(api, `/v2/oauth/authorization-request?${params.toString()}`, AuthorizationRequestSchema, {
    method: "GET",
  });
}

/**
 * Issues, for the account platform, a code that signs the bearer of `accessToken` in on a client, within their
 * session, for the authorization request `params` hold. Redirect to the answered `redirect_uri` with the code, the
 * request's `state` and the account platform's `iss`. Throws as {@link authorizationRequestValidate} does, and an
 * HttpError 401 when the session ended.
 */
export async function authorizationCodeCreate(
  api: AuthenticationApi,
  accessToken: string,
  params: URLSearchParams
): Promise<AuthorizationCode> {
  return await fetchOAuth(api, "/v2/oauth/authorization-code", AuthorizationCodeSchema, {
    headers: { ...FORM_HEADERS, Authorization: `Bearer ${accessToken}` },
    method: "POST",
    body: params,
  });
}

/**
 * Trades, for a client, an authorization code for a token pair in the session it was issued in. The code exchanges
 * once: a second exchange signs the person out everywhere. Throws an {@link OAuthError} for every refusal, such as
 * `invalid_grant`; bring the person back to the signed-out page with a retry.
 */
export async function authorizationCodeExchange(
  api: AuthenticationApi,
  grant: AuthorizationCodeGrant
): Promise<TokenResponse> {
  return await tokenRequest(api, { grant_type: "authorization_code", ...grant });
}

/**
 * Trades, for a client, its refresh token for a new access token, while the session lasts. Throws an
 * {@link OAuthError} with `invalid_grant` once the session ended or was signed out: the person signs in again.
 */
export async function refreshTokenGrant(api: AuthenticationApi, refreshToken: string): Promise<TokenResponse> {
  return await tokenRequest(api, { grant_type: "refresh_token", refresh_token: refreshToken });
}

/** Sends a form-encoded token request to the token endpoint. Parameters without a value are left out. */
async function tokenRequest(
  api: AuthenticationApi,
  params: Record<string, string | undefined>
): Promise<TokenResponse> {
  const body = new URLSearchParams();
  for (const [name, value] of Object.entries(params)) {
    if (value) body.set(name, value);
  }

  return await fetchOAuth(api, "/v2/oauth/token", TokenResponseSchema, { headers: FORM_HEADERS, method: "POST", body });
}

/** Sends an OAuth request, and throws an {@link OAuthError} for an OAuth 2.0 error response. */
async function fetchOAuth<T>(api: AuthenticationApi, input: string, validator: z.ZodType<T>, init: RequestInit) {
  const response = await api.fetchResponse(input, init);

  if (response.status === 400) {
    const text = await response.text();
    const body = OAuthErrorResponseSchema.safeParse(parseJSON(text));
    throw body.success ? new OAuthError(text, body.data) : new HttpError(400, text);
  }

  return await handleHttpResponse(response).then(decodeHttpResponse(validator));
}

/** Parses text as JSON, and returns undefined for text that is not. */
function parseJSON(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return undefined;
  }
}
