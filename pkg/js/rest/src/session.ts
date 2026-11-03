import type { AuthenticationApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * One login on the account platform, joined by the OAuth clients that sign in through it: a device the account signed
 * in on. `current` marks the caller's own. The location fields are absent unless the account consents to locations,
 * and the dates arrive as ISO strings parsed into `Date` objects.
 */
export const SessionSchema = z.object({
  id: z.uuid(),
  current: z.boolean(),
  /** Whether the session ended or was signed out. It stays listed until removed, or 30 days after it ends. */
  loggedOut: z.boolean(),
  /** The User-Agent of the device's browser, empty when unknown. */
  userAgent: z.string(),
  ipAddress: z.string().optional(),
  /** The CLDR region code of the device, such as `FR`. */
  region: z.string().optional(),
  city: z.string().optional(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  /** When the session's latest access token was issued. */
  lastActiveAt: z.iso.datetime().transform((value) => new Date(value)),
  /** When the session ends, unless it is signed out first. */
  expiresAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link SessionSchema} validates. */
export type Session = z.infer<typeof SessionSchema>;

/** The session to sign out. */
export const SessionRevokeRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link SessionRevokeRequestSchema} validates. */
export type SessionRevokeRequest = z.infer<typeof SessionRevokeRequestSchema>;

/** The logged-out session to remove. */
export const SessionDeleteRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link SessionDeleteRequestSchema} validates. */
export type SessionDeleteRequest = z.infer<typeof SessionDeleteRequestSchema>;

/**
 * Lists the sessions of the account bearing the access token, 100 at most: the active ones first, then the logged-out
 * ones, each latest activity first. Throws a 403 for an anonymous token or an OAuth client's, and a 401 once the
 * caller's own session is signed out or has ended.
 */
export async function sessionList(api: AuthenticationApi, accessToken: string): Promise<Session[]> {
  return await api.fetch("/v2/sessions", z.array(SessionSchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

/**
 * Signs one of the caller's sessions out everywhere, the caller's own included. The session stays listed as logged
 * out. Signing out a session already signed out succeeds. Throws a 404 for a session that ended, belongs to another
 * account, or never existed. Throws a 401 once the caller's own session is signed out or has ended.
 */
export async function sessionRevoke(
  api: AuthenticationApi,
  accessToken: string,
  form: SessionRevokeRequest
): Promise<void> {
  return await api.fetchVoid(`/v2/sessions/${encodeURIComponent(form.id)}/revoke`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    method: "PUT",
  });
}

/**
 * Signs out every active session of the caller but their own, everywhere. They stay listed as logged out. Throws a
 * 401 once the caller's own session is signed out or has ended.
 */
export async function sessionRevokeOthers(api: AuthenticationApi, accessToken: string): Promise<void> {
  return await api.fetchVoid("/v2/sessions/revoke-others", {
    headers: { Authorization: `Bearer ${accessToken}` },
    method: "PUT",
  });
}

/**
 * Removes one of the caller's logged-out sessions from their devices. It stays signed out. Throws a 409 for an active
 * session, which {@link sessionRevoke} signs out first, and a 404 for a session that belongs to another account or
 * never existed. Throws a 401 once the caller's own session is signed out or has ended.
 */
export async function sessionDelete(
  api: AuthenticationApi,
  accessToken: string,
  form: SessionDeleteRequest
): Promise<void> {
  return await api.fetchVoid(`/v2/sessions/${encodeURIComponent(form.id)}`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    method: "DELETE",
  });
}
