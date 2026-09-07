import { checkEmail, getHtmlMail } from "./checkEmail";

import * as crypto from "node:crypto";

import { expect } from "vitest";

import {
  type AuthenticationApi,
  Lang,
  claimsGet,
  credentialsCreate,
  shortCodeCreateRegister,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

export function generateRandomMail() {
  return crypto.randomBytes(12).toString("hex") + "@provider.com";
}

export function generateRandomPassword() {
  return crypto.randomBytes(12).toString("hex");
}

export interface PreRegisterData {
  email: string;
  shortCode: string;
}

/** Requests a registration link and returns its email address and short code. */
export async function preRegisterUser(
  api: AuthenticationApi,
  mailHost: string,
  userEmail: string = generateRandomMail()
) {
  const anonToken = await tokenCreateAnon(api);

  await shortCodeCreateRegister(api, anonToken.accessToken, {
    email: userEmail,
    lang: Lang.En,
  });

  const mailData = await checkEmail(mailHost, `to:"${userEmail}"`);

  expect(["Registration Request.", "Create your Agora Storyverse account"]).toContain(mailData.subject);
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
