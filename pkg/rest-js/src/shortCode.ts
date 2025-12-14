import type { AuthenticationApi } from "./api";
import { EmailSchema, LangSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const ShortCodeCreateEmailUpdateRequestSchema = z.object({
  email: EmailSchema,
  lang: LangSchema,
});

export type ShortCodeCreateEmailUpdateRequest = z.infer<typeof ShortCodeCreateEmailUpdateRequestSchema>;

export const ShortCodeCreatePasswordResetRequestSchema = z.object({
  email: EmailSchema,
  lang: LangSchema,
});

export type ShortCodeCreatePasswordResetRequest = z.infer<typeof ShortCodeCreatePasswordResetRequestSchema>;

export const ShortCodeCreateRegisterRequestSchema = z.object({
  email: EmailSchema,
  lang: LangSchema,
});

export type ShortCodeCreateRegisterRequest = z.infer<typeof ShortCodeCreateRegisterRequestSchema>;

export async function shortCodeCreateEmailUpdate(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreateEmailUpdateRequest
): Promise<void> {
  return await api.fetchVoid("/short-code/update-email", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function shortCodeCreatePasswordReset(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreatePasswordResetRequest
): Promise<void> {
  return await api.fetchVoid("/short-code/update-password", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function shortCodeCreateRegister(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreateRegisterRequest
): Promise<void> {
  return await api.fetchVoid("/short-code/register", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
