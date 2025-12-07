import { checkEmail, getHtmlMail } from "./checkEmail";

import * as crypto from "node:crypto";

import { expect } from "vitest";

import {
  type AuthenticationApi,
  Lang,
  type Token,
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

export async function preRegisterUserAsAdmin(
  api: AuthenticationApi,
  mailHost: string,
  superAdminToken: Token,
  userEmail: string = generateRandomMail()
) {
  await shortCodeCreateRegister(api, superAdminToken.accessToken, {
    email: userEmail,
    lang: Lang.En,
  });

  const mailData = await checkEmail(mailHost, `to:"${userEmail}" subject:"Registration Request."`);

  expect(mailData.html).toBeTruthy();
  const links = getHtmlMail(mailData.html as string, "a");
  expect(links).toHaveLength(1);

  const registrationUrl = (links[0] as HTMLAnchorElement).href;

  expect(registrationUrl).toBeTruthy();

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
  };
}
