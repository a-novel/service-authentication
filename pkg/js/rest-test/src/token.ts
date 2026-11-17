import { type AuthenticationApi, tokenCreate, tokenCreateAnon } from "@a-novel/service-authentication-rest";

/** Returns the super-admin's sign-in, from `SUPER_ADMIN_EMAIL` and `SUPER_ADMIN_PASSWORD` or the test stacks' defaults. */
export function superAdminCredentials(): { email: string; password: string } {
  return {
    email: process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com",
    password: process.env.SUPER_ADMIN_PASSWORD ?? "admin",
  };
}

/** Signs the super-admin in, opening a new session, and returns its access token. */
export async function superAdminAccessToken(api: AuthenticationApi): Promise<string> {
  const { accessToken } = await tokenCreate(api, superAdminCredentials());
  return accessToken;
}

/** Opens an anonymous session and returns its access token. */
export async function anonymousAccessToken(api: AuthenticationApi): Promise<string> {
  const { accessToken } = await tokenCreateAnon(api);
  return accessToken;
}
