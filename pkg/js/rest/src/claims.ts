import type { AuthenticationApi } from "./api";
import { RoleSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * Identity encoded in a session's access token: the authenticated user, their roles, and the
 * identifier of the refresh token that issued the session. An anonymous session carries roles
 * but no user, so every field is optional.
 */
export const ClaimsSchema = z.object({
  userID: z.string().optional(),
  roles: z.array(RoleSchema).optional(),
  refreshTokenID: z.string().optional(),
});

export type Claims = z.infer<typeof ClaimsSchema>;

/** Returns the claims carried by the given access token, as resolved by the service. */
export async function claimsGet(api: AuthenticationApi, accessToken: string): Promise<Claims> {
  return await api.fetch("/session", ClaimsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}
