import type { AuthenticationApi } from "./api";
import { RoleSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * Identity encoded in a session's access token: the authenticated user, their roles, the identifier
 * of the refresh token that issued it, and the session it belongs to. Signing out ends that whole
 * session. An anonymous session carries roles but no user, so every field is optional. Claims the
 * schema does not name pass through unchecked.
 */
export const ClaimsSchema = z.looseObject({
  userID: z.string().optional(),
  roles: z.array(RoleSchema).optional(),
  refreshTokenID: z.string().optional(),
  sessionID: z.string().optional(),
});

/** The value {@link ClaimsSchema} validates. */
export type Claims = z.infer<typeof ClaimsSchema>;

/**
 * Returns the claims carried by the given access token, as resolved by the service. A token whose
 * session was signed out fails with 401, like an invalid one.
 */
export async function claimsGet(api: AuthenticationApi, accessToken: string): Promise<Claims> {
  return await api.fetch("/v2/session", ClaimsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}
