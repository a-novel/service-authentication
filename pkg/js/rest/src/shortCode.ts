import type { AuthenticationApi } from "./api";
import { EmailSchema, LangSchema, PasswordSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * The account's current password, the new address to confirm and the language of the confirmation email. The user
 * submits the emailed short code to `credentialsUpdateEmail`.
 */
export const ShortCodeCreateEmailUpdateRequestSchema = z.object({
  currentPassword: PasswordSchema,
  email: EmailSchema,
  lang: LangSchema,
});

/** The value {@link ShortCodeCreateEmailUpdateRequestSchema} validates. */
export type ShortCodeCreateEmailUpdateRequest = z.infer<typeof ShortCodeCreateEmailUpdateRequestSchema>;

/**
 * The account email and the language of the reset email. The user submits the emailed short code to
 * `credentialsResetPassword`.
 */
export const ShortCodeCreatePasswordResetRequestSchema = z.object({
  email: EmailSchema,
  lang: LangSchema,
});

/** The value {@link ShortCodeCreatePasswordResetRequestSchema} validates. */
export type ShortCodeCreatePasswordResetRequest = z.infer<typeof ShortCodeCreatePasswordResetRequestSchema>;

/**
 * The address to invite and the language of the invitation. The user submits the emailed short code
 * to `credentialsCreate`.
 */
export const ShortCodeCreateRegisterRequestSchema = z.object({
  email: EmailSchema,
  lang: LangSchema,
});

/** The value {@link ShortCodeCreateRegisterRequestSchema} validates. */
export type ShortCodeCreateRegisterRequest = z.infer<typeof ShortCodeCreateRegisterRequestSchema>;

/**
 * Emails a short code that authorizes changing the account's email address. Throws a 403 when the current password is
 * wrong.
 */
export async function shortCodeCreateEmailUpdate(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreateEmailUpdateRequest
): Promise<void> {
  return await api.fetchVoid("/v2/short-code/update-email", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/**
 * Cancels the email change awaiting confirmation, so the account keeps its current address and the emailed short code
 * no longer confirms the change. Throws a 404 when no change awaits confirmation, and a 401 once the caller's own
 * session is signed out or has ended.
 */
export async function shortCodeCancelEmailUpdate(api: AuthenticationApi, accessToken: string): Promise<void> {
  return await api.fetchVoid("/v2/short-code/update-email", {
    headers: { Authorization: `Bearer ${accessToken}` },
    method: "DELETE",
  });
}

/** Emails a short code that authorizes resetting the account's password. */
export async function shortCodeCreatePasswordReset(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreatePasswordResetRequest
): Promise<void> {
  return await api.fetchVoid("/v2/short-code/update-password", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/** Emails a registration invitation; the access token must belong to a super-admin. */
export async function shortCodeCreateRegister(
  api: AuthenticationApi,
  accessToken: string,
  form: ShortCodeCreateRegisterRequest
): Promise<void> {
  return await api.fetchVoid("/v2/short-code/register", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
