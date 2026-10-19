import { checkEmail, getHtmlMail } from "./checkEmail";

import * as crypto from "node:crypto";

import { expect } from "vitest";

import {
  type AuthenticationApi,
  Lang,
  claimsGet,
  credentialsCreate,
  shortCodeCreateRegister,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

/** Returns a random email address. */
export function generateRandomMail() {
  return crypto.randomBytes(12).toString("hex") + "@provider.com";
}

/** Returns a random password the service accepts. */
export function generateRandomPassword() {
  return crypto.randomBytes(12).toString("hex");
}

/** A registration invitation: the invited address and the short code its email carries. */
export interface PreRegisterData {
  email: string;
  shortCode: string;
}

/** Requests a registration link as the configured super-admin and returns its email address and short code. */
export async function preRegisterUser(
  api: AuthenticationApi,
  mailHost: string,
  userEmail: string = generateRandomMail()
) {
  const superAdminToken = await tokenCreate(api, {
    email: process.env.SUPER_ADMIN_EMAIL!,
    password: process.env.SUPER_ADMIN_PASSWORD!,
  });

  await shortCodeCreateRegister(api, superAdminToken.accessToken, {
    email: userEmail,
    lang: Lang.En,
  });

  const mailData = await checkEmail(mailHost, `to:"${userEmail}"`);

  expect(["Registration Request.", "Create your Agora Storyverse account", "Create your Agora account"]).toContain(
    mailData.subject
  );
  expect(mailData.html).toBeTruthy();
  const links = getHtmlMail(mailData.html as string, "a");
  expect([1, 2]).toContain(links.length);

  const registrationUrl = (links[0] as HTMLAnchorElement).href;

  expect(registrationUrl).toBeTruthy();
  for (const link of links) {
    expect((link as HTMLAnchorElement).href).toBe(registrationUrl);
  }

  const parsedUrl = new URL(registrationUrl);
  const shortCode = parsedUrl.searchParams.get("shortCode");
  const target = parsedUrl.searchParams.get("target");

  expect(shortCode).toBeTruthy();
  expect(target).toBeTruthy();
  expect(atob(target!)).toBe(userEmail);

  return {
    email: userEmail,
    shortCode: shortCode as string,
  };
}

/** Completes a registration invitation with a random password, returning the new account and its session. */
export async function registerUser(api: AuthenticationApi, data: PreRegisterData) {
  const userPassword = generateRandomPassword();

  const anonToken = await tokenCreateAnon(api);

  const token = await credentialsCreate(api, anonToken.accessToken, {
    email: data.email,
    password: userPassword,
    shortCode: data.shortCode,
  });

  const claims = await claimsGet(api, token.accessToken);

  return {
    email: data.email,
    password: userPassword,
    claims: claims,
    token,
  };
}
