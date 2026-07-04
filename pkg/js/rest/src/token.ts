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

export type Token = z.infer<typeof TokenSchema>;

/** Credentials to open a session: the account's email and password. */
export const TokenCreateRequestSchema = z.object({
  email: EmailSchema,
  password: PasswordSchema,
});

export type TokenCreateRequest = z.infer<typeof TokenCreateRequestSchema>;

/** The current token pair to renew, each as its base64-encoded value. */
export const TokenRefreshRequestSchema = z.object({
  accessToken: z.base64().max(1024),
  refreshToken: z.base64().max(1024),
});

export type TokenRefreshRequest = z.infer<typeof TokenRefreshRequestSchema>;

/** Opens an authenticated session from an email and password, returning a fresh token pair. */
export async function tokenCreate(api: AuthenticationApi, form: TokenCreateRequest): Promise<Token> {
  return await api.fetch("/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/** Opens an anonymous session, returning a token pair whose claims carry no user. */
export async function tokenCreateAnon(api: AuthenticationApi): Promise<Token> {
  return await api.fetch("/session/anon", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
  });
}

/** Exchanges an expiring token pair for a new one, preserving the session's claims. */
export async function tokenRefresh(api: AuthenticationApi, form: TokenRefreshRequest): Promise<Token> {
  return await api.fetch("/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}
