import type { AuthenticationApi } from "./api";
import { EmailSchema, PasswordSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * A session's token pair. The short-lived access token authenticates individual requests and
 * carries the session claims; the longer-lived refresh token renews the pair once it expires.
 */
export const TokenSchema = z.object({
  accessToken: z.string(),
  refreshToken: z.string(),
});

/** The value {@link TokenSchema} validates. */
export type Token = z.infer<typeof TokenSchema>;

/** Credentials to open a session: the account's email and password. */
export const TokenCreateRequestSchema = z.object({
  email: EmailSchema,
  password: PasswordSchema,
});

/** The value {@link TokenCreateRequestSchema} validates. */
export type TokenCreateRequest = z.infer<typeof TokenCreateRequestSchema>;

/** The current token pair to renew, each as a compact JWT (base64url header.payload.signature). */
export const TokenRefreshRequestSchema = z.object({
  accessToken: z.jwt().max(1024),
  refreshToken: z.jwt().max(1024),
});

/** The value {@link TokenRefreshRequestSchema} validates. */
export type TokenRefreshRequest = z.infer<typeof TokenRefreshRequestSchema>;

/** The refresh token of the session to end, as a compact JWT. */
export const TokenRevokeRequestSchema = z.object({
  refreshToken: z.jwt().max(1024),
});

/** The value {@link TokenRevokeRequestSchema} validates. */
export type TokenRevokeRequest = z.infer<typeof TokenRevokeRequestSchema>;

/** Opens an authenticated session from an email and password, returning a fresh token pair. */
export async function tokenCreate(api: AuthenticationApi, form: TokenCreateRequest): Promise<Token> {
  return await api.fetch("/v2/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/** Opens an anonymous session, returning a token pair whose claims carry no user. */
export async function tokenCreateAnon(api: AuthenticationApi): Promise<Token> {
  return await api.fetch("/v2/session/anon", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
  });
}

/** Exchanges an expiring token pair for a new one, preserving the session's claims. */
export async function tokenRefresh(api: AuthenticationApi, form: TokenRefreshRequest): Promise<Token> {
  return await api.fetch("/v2/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

/**
 * Signs a session out: its refresh token can no longer renew it. Revoking a token already revoked or
 * expired succeeds, so a retry is safe. Access tokens the session already issued stay valid until they
 * expire, so discard them too.
 */
export async function tokenRevoke(api: AuthenticationApi, form: TokenRevokeRequest): Promise<void> {
  return await api.fetchVoid("/v2/session/revoke", {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
