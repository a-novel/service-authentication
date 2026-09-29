# Private invitation list

The sheet stores requests for an invitation, not accounts. Joining sends no email and grants no
registration permission. Account creation still requires a short code issued by a super-admin.
The integration needs no database migration, service account, or production secret in GitHub Actions.

## 1. Create the sheet

Use an organization-controlled Google account to create a spreadsheet named **Agora invitation list**.
Keep sharing **Restricted**, give access only to invitation operators, and do not publish it to the web.
Rename its first tab to `Waitlist`. Enter these exact headers in cells A1–C1:

| A     | B        | C            |
| ----- | -------- | ------------ |
| email | language | requested_at |

Leave the remaining rows empty. The writer owns this tab: no formulas, sorting, blank rows, manual
appends, or second script project while the service is writing. Use a separate copy for analysis.
The script lock coordinates instances of this one project; it cannot lock other editors.

The spreadsheet ID is the part between `/d/` and `/edit` in its URL. It is configuration, not a
credential, but keep it out of public issue descriptions alongside the private sheet contents.

## 2. Install the script

From the sheet, open **Extensions → Apps Script**.

1. Replace `Code.gs` with the contents of [`scripts/waitlist/Code.js`](../scripts/waitlist/Code.js).
2. In **Project Settings**, enable **Show appsscript.json manifest file in editor**.
3. Replace the manifest with [`scripts/waitlist/appsscript.json`](../scripts/waitlist/appsscript.json).
   It enables the Sheets advanced service and requests spreadsheet access only. With a custom
   Google Cloud project, also enable the Google Sheets API in that project.
4. In **Project Settings → Script Properties**, add:

| Property                  | Value                                                           |
| ------------------------- | --------------------------------------------------------------- |
| `WAITLIST_SPREADSHEET_ID` | The private spreadsheet ID.                                     |
| `WAITLIST_SECRET`         | A cryptographically random signing key, at least 32 characters. |

Generate the key in your password manager, for example a 64-character random hexadecimal value.
Store it there and in the two required secret stores only. Do not put it in source, a URL, a shell
argument, an issue, a screenshot, or chat. `LAST_JOIN_AT` is maintained by the script; do not create it.

## 3. Deploy the web app

Choose **Deploy → New deployment → Web app**:

- **Execute as:** Me, using the organization-controlled owner.
- **Who has access:** Anyone.

Authorize the spreadsheet scope and copy the deployment URL ending in `/exec`, not the development
URL ending in `/dev`. Anonymous transport access is necessary for the auth server; it does not make
the sheet public. Every operation must have a valid, recent HMAC signature before any sheet access.
If your Workspace policy forbids this deployment setting, stop here: do not broaden sheet sharing.

Future script changes require **Deploy → Manage deployments → Edit → New version → Deploy**.
Keep the same project and deployment so every replica uses the same lock.

## 4. Configure authentication

Set these on the REST service, never Studio:

- `WAITLIST_URL`: the `/exec` deployment URL.
- `WAITLIST_SECRET`: the same key as the script property, injected from the runtime secret store.
- `WAITLIST_TIMEOUT`: optional; defaults to `10s`, with an allowed range greater than zero and at most `20s`.

For Google Cloud, create a Secret Manager secret such as `authentication-waitlist-signing-key` and
grant the auth runtime identity access to that secret. If using Cloud Run, reference a specific secret
version as the `WAITLIST_SECRET` environment variable. Do not add the key to GitHub repository secrets:
the build and tests do not deploy the script or contact the production sheet.

Both URL and secret absent means disabled. Partial or unsafe configuration prevents server startup.
The existing `SERVICE_AUTHENTICATION_ENV_PREFIX` convention applies to these environment names too.

What to share for setup help: the deployment URL and confirmation that the script properties and
runtime secret are installed. **Never share the signing key.** The auth server does not need the
spreadsheet ID; only the script does.

## Behavior and recovery

- The public endpoint accepts `{ "email": "person@example.com", "lang": "en" }` with a bearer token.
  `waitlistJoin` and `WaitlistJoinRequestSchema` are the typed client entry points.
- New rows, repeat requests, and already-registered accounts receive an empty HTTP 202. Existing
  accounts cause no sheet write or email. Repeated requests keep the original language and date.
- Email equality matches the account database exactly. No lowercasing, dot removal, or alias merging.
  The sheet uses RAW values so an address beginning with `=` cannot become a formula.
- One shared lock serializes deduplication and deletion. A second account lookup after adding a row
  closes the race with an account that completed registration during the request.
- Registration attempts cleanup after the database commits, before signing the session tokens.
  Cleanup has a three-second budget and survives a client disconnect. Failure is recorded in tracing
  but cannot roll back the account. There is no durable retry queue: after an outage, stale rows need
  reconciliation before inviting from the list. PostgreSQL remains authoritative for account existence.
- The temporary list is capped at 10,000 entries. The writer admits at most one new row per second,
  while duplicate checks and removal are not rate-limited. The API allows four concurrent waitlist
  requests per replica and an 8 KiB body. These bounds are not a replacement for deployment-level
  abuse protection; anonymous tokens do not identify a person.
- HTTP 429 means writer contention, throttling, or capacity; HTTP 503 means unavailable/disabled
  storage or a full per-instance concurrency gate. Do not show a success message for either.
  Retrying a timed-out join is safe: it checks the sheet before inserting.
- The private script returns JSON acknowledgements through Google's content redirect. The server
  accepts only a single HTTPS GET redirect to `script.googleusercontent.com`; it never forwards the
  signed POST to another host. The writer does not log payloads or private Google error details.

The waitlist is optional and is not probed by the general readiness healthcheck: a Google outage
must not take ordinary authentication out of service. Watch waitlist errors separately.

## Verify before opening the form

Use a separate test sheet and an email you control. Automated tests use local fixtures and cannot
verify the Google deployment's permissions.

1. Join twice through the deployed auth API. Both calls should return 202, with one unchanged row.
2. Request an invitation for an existing account. Expect 202 and no added row or email.
3. Invite the test address through the existing admin flow, then complete registration. Its row should
   disappear; other rows and the headers must remain.
4. With a test deployment only, use a mismatched signing key. A join must fail without modifying the
   sheet. Restore the matching key and verify a retry succeeds.
5. Confirm the sheet remains inaccessible to an unrelated Google account.

Before sending invitations, recheck account existence and treat the rows as unverified submissions.
Restrict the list to this purpose, agree a retention period with the operators, and delete it when
the temporary intake closes. It is not a newsletter subscription or proof of email ownership.

## Retire the temporary flow

Restore public registration permissions and the Studio form in a coordinated release. Disable the
waitlist by removing both runtime settings. Revoke the script deployment, remove its signing-key
property and runtime secret access, then remove the sheet according to the agreed retention policy.
Do not remove the existing invitation-completion routes while issued links are still valid.

## Provider references

- [Web app deployment](https://developers.google.com/apps-script/guides/web)
- [Script properties](https://developers.google.com/apps-script/guides/properties)
- [Shared locks and spreadsheet flush](https://developers.google.com/apps-script/reference/lock/lock)
- [Advanced Google services](https://developers.google.com/apps-script/guides/services/advanced)
- [Literal values and appending rows](https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/append)
- [Cloud Run secret configuration](https://docs.cloud.google.com/run/docs/configuring/services/secrets)
