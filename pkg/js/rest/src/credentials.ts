import type { AuthenticationApi } from "./api";
import { EmailSchema, PasswordSchema, RoleSchema, ShortCodeSchema } from "./form";
import { type Token, TokenSchema } from "./token";

import { HTTP_HEADERS, isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * An account record: its identifier, current email and role, and lifecycle timestamps. The
 * `createdAt` and `updatedAt` fields arrive as ISO strings and are parsed into `Date` objects.
 */
export const CredentialsSchema = z.object({
  id: z.string(),
  email: z.string(),
  role: z.string(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link CredentialsSchema} validates. */
export type Credentials = z.infer<typeof CredentialsSchema>;

/**
 * An email change awaiting confirmation: the requested address and when its confirmation code expires. It lasts
 * until the user confirms it, requests another change, or lets the code expire.
 */
export const PendingEmailSchema = z.object({
  email: z.string(),
  expiresAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link PendingEmailSchema} validates. */
export type PendingEmail = z.infer<typeof PendingEmailSchema>;

/** The authenticated user's own account, with the email change awaiting confirmation, if any. */
export const CredentialsSelfSchema = CredentialsSchema.extend({
  pendingEmail: PendingEmailSchema.optional(),
});

/** The value {@link CredentialsSelfSchema} validates. */
export type CredentialsSelf = z.infer<typeof CredentialsSelfSchema>;

/** New-account details: login email, password, and the short code emailed to confirm the address. */
export const CredentialsCreateRequestSchema = z.object({
  email: EmailSchema,
  password: PasswordSchema,
  shortCode: ShortCodeSchema,
});

/** The value {@link CredentialsCreateRequestSchema} validates. */
export type CredentialsCreateRequest = z.infer<typeof CredentialsCreateRequestSchema>;

/** The email address to test for an existing account. */
export const CredentialsExistsRequestSchema = z.object({
  email: EmailSchema,
});

/** The value {@link CredentialsExistsRequestSchema} validates. */
export type CredentialsExistsRequest = z.infer<typeof CredentialsExistsRequestSchema>;

/** The identifier of the account to fetch. */
export const CredentialsGetRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link CredentialsGetRequestSchema} validates. */
export type CredentialsGetRequest = z.infer<typeof CredentialsGetRequestSchema>;

/** Pagination window and an optional role filter for listing accounts. */
export const CredentialsListRequestSchema = z.object({
  limit: z.int().max(100).optional(),
  offset: z.int().min(0).optional(),
  roles: z.array(RoleSchema).max(10).optional(),
});

/** The value {@link CredentialsListRequestSchema} validates. */
export type CredentialsListRequest = z.infer<typeof CredentialsListRequestSchema>;

/** Reset details for the forgotten-password flow: the new password, the emailed short code, and the target account. */
export const CredentialsResetPasswordRequestSchema = z.object({
  password: PasswordSchema,
  shortCode: ShortCodeSchema,
  userID: z.uuid(),
});

/** The value {@link CredentialsResetPasswordRequestSchema} validates. */
export type CredentialsResetPasswordRequest = z.infer<typeof CredentialsResetPasswordRequestSchema>;

/** The target account and the short code emailed to confirm its new address. */
export const CredentialsUpdateEmailRequestSchema = z.object({
  userID: z.uuid(),
  shortCode: ShortCodeSchema,
});

/** The value {@link CredentialsUpdateEmailRequestSchema} validates. */
export type CredentialsUpdateEmailRequest = z.infer<typeof CredentialsUpdateEmailRequestSchema>;

/** The new password, guarded by the current one to prove the caller owns the account. */
export const CredentialsUpdatePasswordRequestSchema = z.object({
  password: PasswordSchema,
  currentPassword: PasswordSchema,
});

/** The value {@link CredentialsUpdatePasswordRequestSchema} validates. */
export type CredentialsUpdatePasswordRequest = z.infer<typeof CredentialsUpdatePasswordRequestSchema>;

/** The target account and the role to grant it. */
export const CredentialsUpdateRoleRequestSchema = z.object({
  userID: z.uuid(),
  role: RoleSchema,
});

/** The value {@link CredentialsUpdateRoleRequestSchema} validates. */
export type CredentialsUpdateRoleRequest = z.infer<typeof CredentialsUpdateRoleRequestSchema>;

/** Fetches a single account by its identifier. */
export async function credentialsGet(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsGetRequest
): Promise<Credentials> {
  const params = new URLSearchParams();
  params.set("id", form.id);

  return await api.fetch(`/v2/credentials?${params.toString()}`, CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

/**
 * Fetches the account of the user bearing the access token, an OAuth client's included, so a client can name the
 * person it signed in. Throws a 403 for an anonymous token, and a 404 when the account no longer exists.
 */
export async function credentialsGetSelf(api: AuthenticationApi, accessToken: string): Promise<CredentialsSelf> {
  return await api.fetch("/v2/credentials/self", CredentialsSelfSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

/** Reports whether an account exists for the given email, resolving to `false` on a 404 rather than throwing. */
export async function credentialsExists(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsExistsRequest
): Promise<boolean> {
  const params = new URLSearchParams();
  params.set("email", form.email);

  return await api
    .fetchVoid(`/v2/credentials?${params.toString()}`, {
      headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
      method: "HEAD",
    })
    .then(() => true)
    .catch(falseOnNotFound);
}

/** Resolves a 404 to `false` and rethrows any other failure. */
function falseOnNotFound(err: unknown): false {
  if (isHttpStatusError(err, 404)) return false;
  throw err;
}

/** Lists accounts within the request's pagination window, defaulting to the first 100. */
export async function credentialsList(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsListRequest
): Promise<Credentials[]> {
  const params = new URLSearchParams();
  params.set("limit", `${form.limit || 100}`);
  params.set("offset", `${form.offset || 0}`);
  form.roles?.forEach((role) => params.append("roles", role));

  return await api.fetch(`/v2/credentials/all?${params.toString()}`, z.array(CredentialsSchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

/** Registers a new account and returns the token pair for its opening session. */
export async function credentialsCreate(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsCreateRequest
): Promise<Token> {
  return await api.fetch("/v2/credentials", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/** Applies a short-code-confirmed email change and returns the updated account. */
export async function credentialsUpdateEmail(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdateEmailRequest
): Promise<Credentials> {
  return await api.fetch("/v2/credentials/email", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

/** Changes the password of the authenticated account, verified by its current password, and returns the updated account. */
export async function credentialsUpdatePassword(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdatePasswordRequest
): Promise<Credentials> {
  return await api.fetch("/v2/credentials/password", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

/** Sets a new password through the forgotten-password flow, authorized by an emailed short code, and returns the updated account. */
export async function credentialsResetPassword(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsResetPasswordRequest
): Promise<Credentials> {
  return await api.fetch("/v2/credentials/password", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

/** Grants a new role to the target account and returns the updated account. */
export async function credentialsUpdateRole(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdateRoleRequest
): Promise<Credentials> {
  return await api.fetch("/v2/credentials/role", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}
