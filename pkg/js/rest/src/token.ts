import type { AuthenticationApi } from "./api";
import { EmailSchema, PasswordSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const TokenSchema = z.object({
  accessToken: z.string(),
  refreshToken: z.string(),
});

export type Token = z.infer<typeof TokenSchema>;

export const TokenCreateRequestSchema = z.object({
  email: EmailSchema,
  password: PasswordSchema,
});

export type TokenCreateRequest = z.infer<typeof TokenCreateRequestSchema>;

export const TokenRefreshRequestSchema = z.object({
  accessToken: z.base64().max(1024),
  refreshToken: z.base64().max(1024),
});

export type TokenRefreshRequest = z.infer<typeof TokenRefreshRequestSchema>;

export async function tokenCreate(api: AuthenticationApi, form: TokenCreateRequest): Promise<Token> {
  return await api.fetch("/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function tokenCreateAnon(api: AuthenticationApi): Promise<Token> {
  return await api.fetch("/session/anon", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PUT",
  });
}

export async function tokenRefresh(api: AuthenticationApi, form: TokenRefreshRequest): Promise<Token> {
  return await api.fetch("/session", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}
