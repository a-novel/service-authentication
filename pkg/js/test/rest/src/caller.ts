import { AuthenticationApi } from "@a-novel/service-authentication-rest";

/** The account platform's caller, the only one that validates authorization requests and issues codes. */
export const accountPlatform = "system:serviceaccount:account:web";

/** The caller of the test stack's one OAuth client, `studio`. */
export const studioClient = "system:serviceaccount:studio:web";

const serviceTokens: Record<string, Promise<string>> = {};

/**
 * Returns a client of the service that calls it as `caller`, as a platform does: each request carries the service
 * token the stack's issuer signs for that caller.
 */
export function callerApi(caller: string): AuthenticationApi {
  return new AuthenticationApi(process.env.REST_URL!, async (input, init) => {
    const headers = new Headers(init?.headers);
    headers.set("X-Service-Authorization", await (serviceTokens[caller] ??= issueServiceToken(caller)));

    return await fetch(input, { ...init, headers });
  });
}

/** Asks the stack's issuer for a service token of `caller`, valid one hour, which the service accepts. */
async function issueServiceToken(caller: string): Promise<string> {
  const issuer = process.env.SERVICE_TOKEN_ISSUER_URL;
  if (!issuer) throw new Error("SERVICE_TOKEN_ISSUER_URL must be set");

  const params = new URLSearchParams({ subject: caller, audience: "authentication" });
  const response = await fetch(`${issuer}/token?${params.toString()}`);
  if (!response.ok) throw new Error(`the service-token issuer answered ${response.status}`);

  return await response.text();
}
