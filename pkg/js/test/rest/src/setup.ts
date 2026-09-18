import { setTimeout as delay } from "node:timers/promises";

import {
  AuthenticationApi,
  Role,
  claimsGet,
  credentialsCreate,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

import { JSDOM } from "jsdom";
import { simpleParser } from "mailparser";

interface MailpitSearchResponse {
  messages: Array<{ ID: string }>;
}

/** Completes the role-bearing invitation created by the standalone maintenance command. */
export default async function setup() {
  const restURL = requiredEnvironment("REST_URL");
  const mailHost = process.env.MAIL_HOST ?? requiredEnvironment("MAIL_UI_URL");
  const email = process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com";
  const password = process.env.SUPER_ADMIN_PASSWORD ?? "admin";
  const api = new AuthenticationApi(restURL);

  try {
    await tokenCreate(api, { email, password });

    return;
  } catch {
    // A fresh integration database has an invitation instead of credentials.
  }

  const rawMessage = await waitForMail(mailHost, email);
  const mail = await simpleParser(rawMessage, { skipImageLinks: true });
  if (typeof mail.html !== "string") {
    throw new Error("maintenance registration email has no HTML body");
  }

  const document = new JSDOM(mail.html).window.document;
  const registrationURL = document.querySelector<HTMLAnchorElement>("a")?.href;
  if (!registrationURL) {
    throw new Error("maintenance registration email has no registration link");
  }

  const url = new URL(registrationURL);
  const shortCode = url.searchParams.get("shortCode");
  const target = url.searchParams.get("target");
  if (!shortCode || !target || Buffer.from(target, "base64url").toString() !== email) {
    throw new Error("maintenance registration link is invalid");
  }

  const anonymousToken = await tokenCreateAnon(api);
  const token = await credentialsCreate(api, anonymousToken.accessToken, {
    email,
    password,
    shortCode,
  });
  const claims = await claimsGet(api, token.accessToken);
  if (claims.roles?.length !== 1 || claims.roles[0] !== Role.SuperAdmin) {
    throw new Error("maintenance registration did not assign the requested role");
  }
}

async function waitForMail(mailHost: string, email: string): Promise<string> {
  const deadline = Date.now() + 10_000;

  while (Date.now() < deadline) {
    const searchResponse = await fetch(
      `${mailHost}/api/v1/search?query=${encodeURIComponent(`to:"${email}"`)}&limit=1`,
      { headers: { accept: "application/json" } }
    );
    if (!searchResponse.ok) {
      throw new Error(`mail search failed with ${searchResponse.status}`);
    }

    const search = (await searchResponse.json()) as MailpitSearchResponse;
    const messageID = search.messages[0]?.ID;
    if (messageID) {
      const rawResponse = await fetch(`${mailHost}/api/v1/message/${messageID}/raw`);
      if (!rawResponse.ok) {
        throw new Error(`mail read failed with ${rawResponse.status}`);
      }

      return rawResponse.text();
    }

    await delay(100);
  }

  throw new Error(`maintenance registration email was not delivered to ${email}`);
}

function requiredEnvironment(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required for REST integration setup`);
  }

  return value;
}
