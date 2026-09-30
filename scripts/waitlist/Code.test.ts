import { createHmac } from "node:crypto";
import { readFileSync } from "node:fs";
import { Script, createContext } from "node:vm";

import { describe, expect, it, vi } from "vitest";

const source = new Script(readFileSync(new URL("./Code.js", import.meta.url), "utf8") + "\ndoPost;");
const testSecret = "test-only-key-not-a-deployed-secret";
type Reply = { status: string; emails?: string[]; removed?: number };

// Only Google's service boundary is replaced. The deployed script verifies real HMACs and operates on row data.
function writer(emails: string[] = []) {
  const rows = [["email", "language", "requested_at"], ...emails.map((email) => [email, "en", "original date"])];
  const properties = new Map([
    ["WAITLIST_SECRET", testSecret],
    ["WAITLIST_SPREADSHEET_ID", "private-test-sheet"],
  ]);
  const lock = { tryLock: vi.fn(() => true), releaseLock: vi.fn() };
  const sheet = {
    getLastRow: () => rows.length,
    getRange: (row: number, column: number, length: number, width: number) => ({
      getValues: () =>
        rows.slice(row - 1, row - 1 + length).map((cells) => cells.slice(column - 1, column - 1 + width)),
    }),
    deleteRow: vi.fn((row: number) => rows.splice(row - 1, 1)),
  };
  const append = vi.fn((value: { values: string[][] }) => rows.push(...value.values));
  const flush = vi.fn();
  const open = vi.fn(() => ({ getSheetByName: () => sheet }));
  const scope = createContext({
    PropertiesService: {
      getScriptProperties: () => ({
        getProperty: (name: string) => properties.get(name),
        setProperty: (name: string, value: string) => properties.set(name, value),
      }),
    },
    Utilities: {
      computeHmacSha256Signature: (value: string, secret: string) =>
        Array.from(new Int8Array(createHmac("sha256", secret).update(value).digest())),
      base64DecodeWebSafe: (value: string) => Array.from(new Int8Array(Buffer.from(value, "base64url"))),
    },
    LockService: { getScriptLock: () => lock },
    SpreadsheetApp: { openById: open, flush },
    Sheets: { Spreadsheets: { Values: { append } } },
    ContentService: { MimeType: { JSON: "json" }, createTextOutput: (value: string) => ({ setMimeType: () => value }) },
  });
  const post = source.runInContext(scope) as (event: { postData: { contents: string } }) => string;
  function call(request: Record<string, unknown>, secret = testSecret): Reply {
    const payload = JSON.stringify({ timestamp: Math.floor(Date.now() / 1000), ...request });
    const signature = createHmac("sha256", secret).update(payload).digest("base64url");
    return JSON.parse(post({ postData: { contents: JSON.stringify({ payload, signature }) } })) as Reply;
  }
  return { call, post, rows, properties, lock, sheet, append, flush, open };
}

describe("private invitation writer", () => {
  it("stores a literal address and reports duplicates without changing the original row", () => {
    const script = writer();
    const request = { action: "join", email: "=member@example.com", lang: "fr" };
    expect(script.call(request)).toEqual({ status: "accepted" });
    expect(script.call({ ...request, lang: "en" })).toEqual({ status: "already_waitlisted" });
    expect(script.rows).toHaveLength(2);
    expect(script.rows[1].slice(0, 2)).toEqual([request.email, "fr"]);
    expect(script.append).toHaveBeenCalledExactlyOnceWith(
      { values: [script.rows[1]] },
      "private-test-sheet",
      "'Waitlist'!A:C",
      { valueInputOption: "RAW", insertDataOption: "INSERT_ROWS" }
    );
    expect(script.flush).toHaveBeenCalledTimes(2);
    expect(script.lock.releaseLock).toHaveBeenCalledTimes(2);
  });

  it("preserves the auth service's case-sensitive identity rule", () => {
    const script = writer(["Member@example.com"]);
    expect(script.call({ action: "join", email: "member@example.com", lang: "en" }).status).toBe("accepted");
    expect(script.rows).toHaveLength(3);
  });

  it("deletes every matching row, preserves other rows and safely repeats removal", () => {
    const script = writer(["member@example.com", "other@example.com", "member@example.com"]);
    const request = { action: "remove", email: "member@example.com" };
    expect(script.call(request)).toEqual({ status: "accepted", removed: 2 });
    expect(script.call(request)).toEqual({ status: "accepted", removed: 0 });
    expect(script.rows.slice(1).map(([email]) => email)).toEqual(["other@example.com"]);
    expect(script.sheet.deleteRow.mock.calls).toEqual([[4], [2]]);
    expect(script.flush.mock.invocationCallOrder[0]).toBeLessThan(script.lock.releaseLock.mock.invocationCallOrder[0]);
  });

  it("pages unique addresses without skipping entries after a batch is removed", () => {
    const emails = Array.from({ length: 103 }, (_, index) => `member${String(index).padStart(3, "0")}@example.com`);
    const script = writer([...emails].reverse().concat(emails[0]));
    const first = script.call({ action: "list" });
    expect(first).toEqual({ status: "accepted", emails: emails.slice(0, 100) });
    expect(script.sheet.deleteRow).not.toHaveBeenCalled();
    expect(script.call({ action: "remove", emails: first.emails })).toEqual({ status: "accepted", removed: 101 });
    expect(script.call({ action: "list", after: emails[99] })).toEqual({
      status: "accepted",
      emails: emails.slice(100),
    });
    expect(script.call({ action: "list", after: emails[102] })).toEqual({ status: "accepted", emails: [] });
    expect(
      script.rows
        .slice(1)
        .map(([email]) => email)
        .sort()
    ).toEqual(emails.slice(100));
  });

  it("safely reruns a partially failed removal and preserves case-distinct addresses", () => {
    const script = writer(["one@example.com", "pending@example.com", "One@example.com", "two@example.com"]);
    const request = { action: "remove", emails: ["one@example.com", "two@example.com"] };
    script.sheet.deleteRow
      .mockImplementationOnce((row) => script.rows.splice(row - 1, 1))
      .mockImplementationOnce(() => {
        throw new Error("private Google data");
      });
    expect(script.call(request)).toEqual({ status: "unavailable" });
    expect(script.lock.releaseLock).toHaveBeenCalledOnce();
    expect(script.call(request)).toEqual({ status: "accepted", removed: 1 });
    expect(script.call(request)).toEqual({ status: "accepted", removed: 0 });
    expect(script.rows.slice(1).map(([email]) => email)).toEqual(["pending@example.com", "One@example.com"]);
  });

  it("rejects tampered, stale and malformed operations before accessing the sheet", () => {
    const script = writer();
    const request = { action: "join", email: "member@example.com", lang: "en" };
    expect(script.call(request, "wrong-signing-key").status).toBe("unauthorized");
    expect(script.call({ ...request, timestamp: 1 }).status).toBe("unauthorized");
    expect(script.call({ ...request, lang: "xx" }).status).toBe("invalid");
    expect(script.call({ ...request, email: "invalid" }).status).toBe("invalid");
    expect(script.call({ action: "remove", email: "invalid" }).status).toBe("invalid");
    expect(script.call({ action: "remove", emails: [] }).status).toBe("invalid");
    expect(script.call({ action: "remove", emails: ["invalid"] }).status).toBe("invalid");
    expect(script.call({ action: "remove", emails: Array(101).fill("member@example.com") }).status).toBe("invalid");
    expect(script.call({ action: "remove", email: request.email, emails: [request.email] }).status).toBe("invalid");
    expect(script.call({ action: "list", after: "invalid" }).status).toBe("invalid");
    expect(script.call({ action: "unknown" }).status).toBe("invalid");
    expect(script.post({ postData: { contents: "not json" } })).toBe('{"status":"unavailable"}');
    expect(script.open).not.toHaveBeenCalled();
  });

  it("bounds contention without reading or writing outside the shared lock", () => {
    const script = writer();
    script.lock.tryLock.mockReturnValue(false);
    expect(script.call({ action: "join", email: "member@example.com", lang: "en" }).status).toBe("busy");
    expect(script.open).not.toHaveBeenCalled();
    expect(script.lock.releaseLock).not.toHaveBeenCalled();
  });

  it("allows cleanup and duplicates while rate-limiting new rows", () => {
    const script = writer(["member@example.com"]);
    script.properties.set("LAST_JOIN_AT", String(Date.now()));
    expect(script.call({ action: "join", email: "other@example.com", lang: "en" }).status).toBe("busy");
    expect(script.call({ action: "join", email: "member@example.com", lang: "en" }).status).toBe("already_waitlisted");
    expect(script.call({ action: "remove", email: "member@example.com" }).status).toBe("accepted");
    expect(script.append).not.toHaveBeenCalled();
  });

  it("bounds the sheet without preventing cleanup or duplicate warnings", () => {
    const emails = Array.from({ length: 10000 }, (_, index) => `member${index}@example.com`);
    const script = writer(emails);
    expect(script.call({ action: "join", email: "extra@example.com", lang: "en" }).status).toBe("busy");
    expect(script.call({ action: "join", email: emails[0], lang: "fr" }).status).toBe("already_waitlisted");
    expect(script.call({ action: "remove", email: emails[0] }).status).toBe("accepted");
    expect(script.rows).toHaveLength(10000);
  });

  it("refuses to list invalid sheet data", () => {
    const script = writer(["invalid"]);
    expect(script.call({ action: "list" })).toEqual({ status: "unavailable" });
    expect(script.sheet.deleteRow).not.toHaveBeenCalled();
  });

  it("does not mutate a sheet with unexpected headers", () => {
    const script = writer();
    script.rows[0][0] = "unexpected";
    expect(script.call({ action: "join", email: "member@example.com", lang: "en" }).status).toBe("unavailable");
    expect(script.append).not.toHaveBeenCalled();
    expect(script.lock.releaseLock).toHaveBeenCalledOnce();
  });

  it.each(["append", "flush"] as const)("releases the lock and hides private details when %s fails", (method) => {
    const script = writer();
    script[method].mockImplementation(() => {
      throw new Error("private Google data");
    });
    expect(script.call({ action: "join", email: "member@example.com", lang: "en" })).toEqual({ status: "unavailable" });
    expect(script.lock.releaseLock).toHaveBeenCalledOnce();
  });
});
