/* exported doPost */

/** Accepts signed invitation-list operations from the authentication service. */
function doPost(event) {
  try {
    const properties = PropertiesService.getScriptProperties();
    const secret = properties.getProperty("WAITLIST_SECRET");
    const contents = event?.postData?.contents;
    if (!secret || secret.length < 32 || typeof contents !== "string" || contents.length > 131072) {
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
    if (!Number.isInteger(request.timestamp) || Math.abs(Date.now() / 1000 - request.timestamp) > 300) {
      return reply({ status: "unauthorized" });
    }
    if (!validRequest(request)) return reply({ status: "invalid" });

    const lock = LockService.getScriptLock();
    if (!lock.tryLock(1000)) return reply({ status: "busy" });
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
  return typeof email === "string" && email.length <= 1024 && /^[^\s@]+@[^\s@]+$/u.test(email);
}

function validRequest(request) {
  switch (request.action) {
    case "join":
      return validEmail(request.email) && ["en", "fr"].includes(request.lang);
    case "remove":
      return validEmail(request.email);
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
  if (count > 10000) return { status: "unavailable" };

  const emails = count ? sheet.getRange(2, 1, count, 1).getValues().flat() : [];
  if (request.action === "remove") {
    // Work upward so removing one row cannot shift an unprocessed match.
    for (let index = emails.length - 1; index >= 0; index--) {
      if (emails[index] === request.email) sheet.deleteRow(index + 2);
    }
    return { status: "accepted" };
  }

  if (emails.includes(request.email)) return { status: "accepted" };
  if (count === 10000 || Date.now() - Number(properties.getProperty("LAST_JOIN_AT") ?? 0) < 1000) {
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
