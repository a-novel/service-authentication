import { checkEmail, mailLinks } from "./mail.js";
import { anonymousAccessToken, superAdminAccessToken, superAdminCredentials } from "./token.js";

import { randomBytes } from "node:crypto";

import { isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import {
  type AuthenticationApi,
  Lang,
  claimsGet,
  credentialsCreate,
  shortCodeCreateRegister,
} from "@a-novel/service-authentication-rest";

/** Returns a random email address. */
export function generateRandomMail() {
  return randomBytes(12).toString("hex") + "@provider.com";
}

/** Returns a random password the service accepts. */
export function generateRandomPassword() {
  return randomBytes(12).toString("hex");
}

/** A registration invitation: the invited address and the short code its email carries. */
export interface PreRegisterData {
  email: string;
  shortCode: string;
}

/** Invites an address as the super-admin, and returns the short code its invitation email carries. */
export async function preRegisterUser(
  api: AuthenticationApi,
  email: string = generateRandomMail()
): Promise<PreRegisterData> {
  await shortCodeCreateRegister(api, await superAdminAccessToken(api), { email, lang: Lang.En });

  return { email, shortCode: await invitationShortCode(email) };
}

/**
 * Completes a registration invitation with a random password, and returns the new account with its first session.
 * Without an invitation, it invites a random address first.
 */
export async function registerUser(api: AuthenticationApi, invitation?: PreRegisterData) {
  const { email, shortCode } = invitation ?? (await preRegisterUser(api));
  const password = generateRandomPassword();

  const token = await credentialsCreate(api, await anonymousAccessToken(api), { email, password, shortCode });
  const claims = await claimsGet(api, token.accessToken);

  return { email, password, claims, token };
}

/** Registers a new account, which holds the user role, and returns its access token. */
export async function userAccessToken(api: AuthenticationApi): Promise<string> {
  const { token } = await registerUser(api);
  return token.accessToken;
}

/**
 * Completes the invitation the service mails the super-admin at startup, unless the super-admin already signs in.
 * Call it from the suite's vitest globalSetup, so every test can sign in as the super-admin.
 */
export async function registerSuperAdmin(api: AuthenticationApi): Promise<void> {
  try {
    await superAdminAccessToken(api);

    return;
  } catch (error) {
    // A fresh database holds the invitation instead of credentials.
    if (!isHttpStatusError(error, 401)) throw error;
  }

  const { email, password } = superAdminCredentials();
  // A global setup runs outside the tests' timeout, so it waits as long as a loaded stack may take to send the mail.
  const shortCode = await invitationShortCode(email, 15_000);
  await credentialsCreate(api, await anonymousAccessToken(api), { email, password, shortCode });
}

/** Waits for the invitation mailed to an address, and returns the short code its link carries for that address. */
async function invitationShortCode(email: string, timeout?: number): Promise<string> {
  const [link] = mailLinks((await checkEmail(`to:"${email}"`, timeout)).HTML);
  if (!link) throw new Error(`the invitation to ${email} carries no link`);

  const { searchParams } = new URL(link);
  const shortCode = searchParams.get("shortCode");
  const target = Buffer.from(searchParams.get("target") ?? "", "base64url").toString();
  if (!shortCode || target !== email) throw new Error(`the invitation link ${link} does not register ${email}`);

  return shortCode;
}
