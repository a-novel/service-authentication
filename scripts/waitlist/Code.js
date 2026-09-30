/* exported doPost */

const LIMITS = {
  // Minimum signing-key length shared with the authentication configuration.
  secret: 32,
  // Allows one bounded maintenance batch with JSON escaping overhead.
  requestCharacters: 1048576,
  // Matches the authentication service email-length limit.
  email: 1024,
  // Accepts a short clock skew while rejecting old signed requests.
  signatureAgeSeconds: 300,
  // Limits contention on the single writer shared by every service replica.
  lockWaitMs: 1000,
  // Keeps each private maintenance request bounded; shared with auth.WaitlistBatchSize.
  batch: 100,
  // Caps temporary storage and each maintenance scan; shared with auth.WaitlistMaxEntries.
  entries: 10000,
  // Admits one new address per second across all service replicas.
  joinIntervalMs: 1000,
};

/** Accepts signed invitation-list operations from the authentication service. */
function doPost(event) {
  try {
    const properties = PropertiesService.getScriptProperties();
    const secret = properties.getProperty("WAITLIST_SECRET");
    const contents = event?.postData?.contents;
    if (
      !secret ||
      secret.length < LIMITS.secret ||
      typeof contents !== "string" ||
      contents.length > LIMITS.requestCharacters
    ) {
      return reply({ status: "unavailable" });
    }

    const envelope = JSON.parse(contents);
    if (typeof envelope.payload !== "string" || typeof envelope.signature !== "string") {
      return reply({ status: "unauthorized" });
    }
    const expected = Utilities.computeHmacSha256Signature(envelope.payload, secret);
    const actual = Utilities.base64DecodeWebSafe(envelope.signature);
    if (actual.length !== expected.length) return reply({ status: "unauthorized" });
    let difference = 0;
    for (let index = 0; index < expected.length; index++) difference |= actual[index] ^ expected[index];
    if (difference !== 0) return reply({ status: "unauthorized" });

    const request = JSON.parse(envelope.payload);
    if (
      !Number.isInteger(request.timestamp) ||
      Math.abs(Date.now() / 1000 - request.timestamp) > LIMITS.signatureAgeSeconds
    ) {
      return reply({ status: "unauthorized" });
    }
    if (!validRequest(request)) return reply({ status: "invalid" });

    const lock = LockService.getScriptLock();
    if (!lock.tryLock(LIMITS.lockWaitMs)) return reply({ status: "busy" });
    try {
      return reply(applyRequest(request, properties));
    } finally {
      // Pending row deletions must commit before another execution can inspect the sheet.
      try {
        SpreadsheetApp.flush();
      } finally {
        lock.releaseLock();
      }
    }
  } catch {
    // Google exceptions may include private sheet contents or credential-bearing URLs.
    return reply({ status: "unavailable" });
  }
}

function validEmail(email) {
  return typeof email === "string" && email.length <= LIMITS.email && /^[^\s@]+@[^\s@]+$/u.test(email);
}

function validRequest(request) {
  switch (request.action) {
    case "join":
      return validEmail(request.email) && ["en", "fr"].includes(request.lang);
    case "remove":
      if (request.emails === undefined) return validEmail(request.email);
      return (
        request.email === undefined &&
        Array.isArray(request.emails) &&
        request.emails.length > 0 &&
        request.emails.length <= LIMITS.batch &&
        request.emails.every(validEmail)
      );
    case "list":
      return request.after === undefined || validEmail(request.after);
    default:
      return false;
  }
}

function applyRequest(request, properties) {
  const spreadsheetId = properties.getProperty("WAITLIST_SPREADSHEET_ID");
  const sheet = SpreadsheetApp.openById(spreadsheetId).getSheetByName("Waitlist");
  if (!sheet || sheet.getRange(1, 1, 1, 3).getValues()[0].join(",") !== "email,language,requested_at") {
    return { status: "unavailable" };
  }
  const count = sheet.getLastRow() - 1;
  if (count > LIMITS.entries) return { status: "unavailable" };

  const emails = count ? sheet.getRange(2, 1, count, 1).getValues().flat() : [];
  if (request.action === "list") {
    if (!emails.every(validEmail)) return { status: "unavailable" };
    // Email cursors keep deletions from shifting unprocessed entries across page boundaries.
    const page = [...new Set(emails)]
      .sort()
      .filter((email) => email > (request.after ?? ""))
      .slice(0, LIMITS.batch);
    return { status: "accepted", emails: page };
  }
  if (request.action === "remove") {
    const targets = new Set(request.emails ?? [request.email]);
    let removed = 0;
    // Work upward so removing one row cannot shift an unprocessed match.
    for (let index = emails.length - 1; index >= 0; index--) {
      if (targets.has(emails[index])) {
        sheet.deleteRow(index + 2);
        removed++;
      }
    }
    return { status: "accepted", removed };
  }

  if (emails.includes(request.email)) return { status: "already_waitlisted" };
  if (
    count === LIMITS.entries ||
    Date.now() - Number(properties.getProperty("LAST_JOIN_AT") ?? 0) < LIMITS.joinIntervalMs
  ) {
    return { status: "busy" };
  }
  properties.setProperty("LAST_JOIN_AT", String(Date.now()));

  // RAW preserves addresses beginning with formula characters as literal text.
  Sheets.Spreadsheets.Values.append(
    { values: [[request.email, request.lang, new Date().toISOString()]] },
    spreadsheetId,
    "'Waitlist'!A:C",
    { valueInputOption: "RAW", insertDataOption: "INSERT_ROWS" }
  );
  return { status: "accepted" };
}

function reply(result) {
  return ContentService.createTextOutput(JSON.stringify(result)).setMimeType(ContentService.MimeType.JSON);
}
