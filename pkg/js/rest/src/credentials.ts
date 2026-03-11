import type { AuthenticationApi } from "./api";
import { EmailSchema, PasswordSchema, RoleSchema, ShortCodeSchema } from "./form";
import { type Token, TokenSchema } from "./token";

import { HTTP_HEADERS, isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const CredentialsSchema = z.object({
  id: z.string(),
  email: z.string(),
  role: z.string(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type Credentials = z.infer<typeof CredentialsSchema>;

export const CredentialsCreateRequestSchema = z.object({
  email: EmailSchema,
  password: PasswordSchema,
  shortCode: ShortCodeSchema,
});

export type CredentialsCreateRequest = z.infer<typeof CredentialsCreateRequestSchema>;

export const CredentialsExistsRequestSchema = z.object({
  email: EmailSchema,
});

export type CredentialsExistsRequest = z.infer<typeof CredentialsExistsRequestSchema>;

export const CredentialsGetRequestSchema = z.object({
  id: z.uuid(),
});

export type CredentialsGetRequest = z.infer<typeof CredentialsGetRequestSchema>;

export const CredentialsListRequestSchema = z.object({
  limit: z.int().max(100).optional(),
  offset: z.int().min(0).optional(),
  roles: z.array(RoleSchema).max(10).optional(),
});

export type CredentialsListRequest = z.infer<typeof CredentialsListRequestSchema>;

export const CredentialsResetPasswordRequestSchema = z.object({
  password: PasswordSchema,
  shortCode: ShortCodeSchema,
  userID: z.uuid(),
});

export type CredentialsResetPasswordRequest = z.infer<typeof CredentialsResetPasswordRequestSchema>;

export const CredentialsUpdateEmailRequestSchema = z.object({
  userID: z.uuid(),
  shortCode: ShortCodeSchema,
});

export type CredentialsUpdateEmailRequest = z.infer<typeof CredentialsUpdateEmailRequestSchema>;

export const CredentialsUpdatePasswordRequestSchema = z.object({
  password: PasswordSchema,
  currentPassword: PasswordSchema,
});

export type CredentialsUpdatePasswordRequest = z.infer<typeof CredentialsUpdatePasswordRequestSchema>;

export const CredentialsUpdateRoleRequestSchema = z.object({
  userID: z.uuid(),
  role: RoleSchema,
});

export type CredentialsUpdateRoleRequest = z.infer<typeof CredentialsUpdateRoleRequestSchema>;

export async function credentialsGet(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsGetRequest
): Promise<Credentials> {
  const params = new URLSearchParams();
  params.set("id", form.id);

  return await api.fetch(`/credentials?${params.toString()}`, CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function credentialsExists(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsExistsRequest
): Promise<boolean> {
  const params = new URLSearchParams();
  params.set("email", form.email);

  return await api
    .fetchVoid(`/credentials?${params.toString()}`, {
      headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
      method: "HEAD",
    })
    .then(() => true)
    .catch((err) => {
      if (isHttpStatusError(err, 404)) return false;
      throw err;
    });
}

export async function credentialsList(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsListRequest
): Promise<Credentials[]> {
  const params = new URLSearchParams();
  params.set("limit", `${form.limit || 100}`);
  params.set("offset", `${form.offset || 0}`);
  form.roles?.forEach((role) => params.append("roles", role));

  return await api.fetch(`/credentials/all?${params.toString()}`, z.array(CredentialsSchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function credentialsCreate(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsCreateRequest
): Promise<Token> {
  return await api.fetch("/credentials", TokenSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function credentialsUpdateEmail(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdateEmailRequest
): Promise<Credentials> {
  return await api.fetch("/credentials/email", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

export async function credentialsUpdatePassword(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdatePasswordRequest
): Promise<Credentials> {
  return await api.fetch("/credentials/password", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

export async function credentialsResetPassword(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsResetPasswordRequest
): Promise<Credentials> {
  return await api.fetch("/credentials/password", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function credentialsUpdateRole(
  api: AuthenticationApi,
  accessToken: string,
  form: CredentialsUpdateRoleRequest
): Promise<Credentials> {
  return await api.fetch("/credentials/role", CredentialsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}
