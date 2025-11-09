import type { AuthenticationApi } from "./api";
import { RoleSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel/nodelib-browser/http";

import { z } from "zod";

export const ClaimsSchema = z.object({
  userID: z.string().optional(),
  roles: z.array(RoleSchema).optional(),
  refreshTokenID: z.string().optional(),
});

export type Claims = z.infer<typeof ClaimsSchema>;

export async function claimsGet(api: AuthenticationApi, accessToken: string): Promise<Claims> {
  return await api.fetch("/session", ClaimsSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}
