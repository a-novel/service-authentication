import { setTimeout as delay } from "node:timers/promises";

import { HTTP_HEADERS, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

/** A MailPit message, reduced to the fields the tests read. */
export interface MailpitMessage {
  ID: string;
  Subject: string;
  HTML: string;
}

/** A MailPit search result, reduced to the IDs of the messages, newest first. */
interface MailpitSearch {
  messages: { ID: string }[];
}

/** Matches the target of each link of an HTML email. */
const LINK = /<a\s[^>]*href="([^"]*)"/g;

/**
 * Waits up to `timeout` milliseconds for the newest MailPit message matching a search, deletes it so the next search
 * finds a later one, and returns it. The default three seconds stay inside vitest's five-second test timeout. The search
 * uses MailPit's syntax, unencoded, such as `to:"user+tag@example.com"`.
 */
export async function checkEmail(search: string, timeout = 3000): Promise<MailpitMessage> {
  const query = new URLSearchParams({ query: search, limit: "1" });
  const deadline = Date.now() + timeout;

  while (Date.now() < deadline) {
    const { messages } = (await mailpit(`search?${query}`).then((response) => response.json())) as MailpitSearch;
    if (messages.length > 0) return await takeMessage(messages[0].ID);

    await delay(100);
  }

  throw new Error(`no email matched ${search}`);
}

/** Returns the target of each link of an HTML email, in order. */
export function mailLinks(html: string): string[] {
  return Array.from(html.matchAll(LINK), ([, href]) => href.replaceAll("&amp;", "&"));
}

/** Reads a MailPit message, then deletes it. */
async function takeMessage(id: string): Promise<MailpitMessage> {
  const message = (await mailpit(`message/${id}`).then((response) => response.json())) as MailpitMessage;

  const body = JSON.stringify({ IDs: [id] });
  await mailpit("messages", { method: "DELETE", headers: HTTP_HEADERS.JSON, body }).then((response) =>
    response.body?.cancel()
  );

  return message;
}

/** Calls MailPit's API, which listens on `MAIL_UI_URL`, and returns its successful response. */
async function mailpit(path: string, init?: RequestInit): Promise<Response> {
  const mailUrl = process.env.MAIL_UI_URL;
  if (!mailUrl) throw new Error("MAIL_UI_URL must be set");

  return await fetch(`${mailUrl}/api/v1/${path}`, init).then(handleHttpResponse);
}
