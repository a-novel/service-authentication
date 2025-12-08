import { expect, vi } from "vitest";

import { handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import { JSDOM } from "jsdom";
import { type AddressObject, type ParsedMail, simpleParser } from "mailparser";

export async function checkEmail(mailHost: string, search: string, clear: boolean = true): Promise<ParsedMail> {
  const messageID = await vi.waitFor(
    async () => {
      const res = await fetch(`${mailHost}/api/v1/search?query=${search}&limit=1`, {
        headers: { accept: "application/json" },
      }).then(handleHttpResponse);
      const searchRes = await res.json();
      expect(searchRes.messages).toHaveLength(1);
      expect(searchRes.messages[0].MessageID).toBeTruthy();

      //return await simpleParser(rawContent, { skipImageLinks: true });
      return searchRes.messages[0].ID as string;
    },
    { timeout: 1000 }
  );

  const rawMessage = await fetch(`${mailHost}/api/v1/message/${messageID}/raw`, {
    headers: { accept: "application/json" },
  }).then(handleHttpResponse);

  // Delete the message once consumed.
  if (clear) {
    await fetch(`${mailHost}/api/v1/messages`, {
      method: "DELETE",
      body: JSON.stringify({ IDs: [messageID] }),
      headers: {
        accept: "application/json",
        "content-type": "application/json",
      },
    }).then(handleHttpResponse);
  }

  return await simpleParser(await rawMessage.text(), { skipImageLinks: true });
}

export function expectAddress(expected: string | string[], value: AddressObject | AddressObject[] | undefined) {
  expect(value).toBeDefined();

  if (Array.isArray(expected)) {
    expect(Array.isArray(value)).toBeTruthy();

    for (const expectedItem of expected) {
      expect(
        (value as AddressObject[]).some((valueItem) => "text" in valueItem && valueItem.text === expectedItem)
      ).toBeTruthy();
    }

    return;
  }

  expect("text" in value!).toBeTruthy();
  expect((value as AddressObject).text).toBe(expected);
}

export function getHtmlMail(raw: string, querySelector: string) {
  const dom = new JSDOM(raw);
  return dom.window.document.querySelectorAll(querySelector);
}
