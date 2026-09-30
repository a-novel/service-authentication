import type { AuthenticationApi } from "./api";
import { EmailSchema, LangSchema } from "./form";

import { HTTP_HEADERS, HttpError, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** Invitation-list fields, using the same email and language limits as registration. */
export const WaitlistJoinRequestSchema = z.object({ email: EmailSchema, lang: LangSchema });

/** An invitation request; this neither creates an account nor sends a verification link. */
export type WaitlistJoinRequest = z.infer<typeof WaitlistJoinRequestSchema>;

/** Stable membership reasons returned by HTTP 409; no submitted address is echoed. */
export const WaitlistJoinConflictSchema = z.object({ code: z.enum(["account_exists", "already_waitlisted"]) });

/** A membership warning the UI can translate independently of transport error wording. */
export type WaitlistJoinConflict = z.infer<typeof WaitlistJoinConflictSchema>;

/** A 409 response with a validated code, also compatible with the shared HttpError status checks. */
export class WaitlistJoinConflictError extends HttpError {
  /** Identifies the warning to display without parsing the error message. */
  readonly code: WaitlistJoinConflict["code"];

  constructor(code: WaitlistJoinConflict["code"]) {
    super(409, code);
    this.code = code;
  }
}

/**
 * Joins the temporary invitation list using any session, including an anonymous token.
 * Throws WaitlistJoinConflictError for existing accounts or duplicate requests.
 * Throws on validation, capacity or storage errors; an acknowledgement is not an invitation.
 */
export async function waitlistJoin(
  api: AuthenticationApi,
  accessToken: string,
  form: WaitlistJoinRequest
): Promise<void> {
  const response = await api.fetchResponse("/v2/waitlist", {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
  if (response.status === 409) {
    const conflict = WaitlistJoinConflictSchema.parse(await response.json());
    throw new WaitlistJoinConflictError(conflict.code);
  }
  await handleHttpResponse(response);
}
