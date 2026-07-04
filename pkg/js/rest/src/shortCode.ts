import type { AuthenticationApi } from "./api";
import { EmailSchema, LangSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

// A short code is a one-time secret the service emails to a user to authorize a sensitive action.
// The request here asks the service to generate and send one; the user later submits the received
// code back through the matching credentials endpoint to complete the action. Each request names
// the target email and the language to send the email in.

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

/** Emails a short code that authorizes changing the account's email address. */
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

/** Emails a short code that authorizes resetting the account's password. */
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

/** Emails a short code that authorizes registering a new account under the given address. */
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
