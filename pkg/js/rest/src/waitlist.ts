import type { AuthenticationApi } from "./api";
import { EmailSchema, LangSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** Invitation-list fields, using the same email and language limits as registration. */
export const WaitlistJoinRequestSchema = z.object({ email: EmailSchema, lang: LangSchema });

/** An invitation request; this neither creates an account nor sends a verification link. */
export type WaitlistJoinRequest = z.infer<typeof WaitlistJoinRequestSchema>;

/**
 * Joins the temporary invitation list using any session, including an anonymous token.
 * Duplicate requests and existing accounts receive the same acknowledgement.
 * Throws on validation, capacity or storage errors; an acknowledgement is not an invitation.
 */
export async function waitlistJoin(
  api: AuthenticationApi,
  accessToken: string,
  form: WaitlistJoinRequest
): Promise<void> {
  return await api.fetchVoid("/v2/waitlist", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
