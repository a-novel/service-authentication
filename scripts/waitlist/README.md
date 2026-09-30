# Install the private invitation list

This directory contains the Google Apps Script that stores Agora invitation requests in a private
Google Sheet. Follow this guide to install it, connect the authentication service, and verify the
complete flow. No database migration is needed.

Joining the list does **not** create an account, send an email, or grant registration permission.
Account creation still requires an invitation from the existing admin flow. Once registration
completes, authentication attempts to remove the matching row.

Jump to [installation](#installation), [verification](#verify-the-installation),
[cleanup](#on-demand-cleanup), [updates and rollback](#update-or-roll-back-the-script),
[key rotation](#rotate-the-signing-key), or [troubleshooting](#troubleshooting).

## Before you start

You need:

- An organization-controlled Google account that can create a spreadsheet and deploy an Apps Script
  web app accessible without a Google sign-in. Confirm your Workspace policy allows this first.
- Permission to configure and redeploy the authentication REST service, and to manage its runtime
  secrets. Maintenance jobs that access the list need the same configuration.
- These files from the **same repository release** as the authentication service you will deploy:

  | File                                 | Use                                                               |
  | ------------------------------------ | ----------------------------------------------------------------- |
  | [Code.js](./Code.js)                 | Copy its entire contents into the Apps Script editor's `Code.gs`. |
  | [appsscript.json](./appsscript.json) | Copy its entire contents into the project's manifest.             |
  | [Code.test.ts](./Code.test.ts)       | Local automated tests only; do not upload this file to Google.    |

- A password manager to generate and retain a random signing key.
- A separate test environment and email addresses you control for the verification steps.

Use the repository's [releases](https://github.com/a-novel/service-authentication/releases) to select
the matching files. This browser-based installation needs no local build, npm package, `clasp`,
service-account key, trigger, or scheduled task. The existing authentication database remains the
source of truth for accounts; the sheet stores pending requests only.

There are two separate permission boundaries:

```text
Studio → authentication API → signed server request → Apps Script → private Google Sheet
                            WAITLIST_SECRET          executes as the Google owner
```

The browser never receives the signing key or accesses Google directly. The web app accepts incoming
requests without a Google login, but the script verifies a recent HMAC signature before accessing the
sheet. The sheet itself stays restricted. Use one sheet, one script project, and a separate signing key
per environment; two script projects cannot share the lock used for deduplication.

## Installation

### 1. Create the spreadsheet

1. Sign in with the organization-controlled account that will own the deployment.
2. Create a blank spreadsheet, for example **Agora invitation list — staging**. The spreadsheet title
   is for operators; the script does not depend on it.
3. Open **Share** and keep general access **Restricted**. Give access only to the operators who need
   it. Do not publish the sheet to the web or enable link sharing.
4. Rename the first sheet tab to exactly `Waitlist` (capital `W`).
5. Enter these exact lowercase headers, one per cell, with no surrounding spaces:

   | Cell | Value          |
   | ---- | -------------- |
   | A1   | `email`        |
   | B1   | `language`     |
   | C1   | `requested_at` |

6. Leave all data rows empty. Do not add a sample row, a formula, merged cells, or extra headings.
7. Save the spreadsheet ID privately. It is the part after `/d/` in the browser URL:

   ```text
   https://docs.google.com/spreadsheets/d/<SPREADSHEET_ID>/edit
   ```

Copy the ID only, not the whole URL or the `gid` identifying a tab. It is configuration, not a
credential, but should remain out of public issues alongside the private sheet contents.

The script owns the `Waitlist` tab. While it is active, do not sort, insert blank rows, append data
manually, or write through another script. A script lock cannot protect against human editors.
Use a separate copy for analysis; never connect that copy to another live writer for the same list.

### 2. Install both script files

1. From the spreadsheet, open **Extensions → Apps Script**. Name the project so its purpose and
   environment are clear, for example **Agora invitation list — staging**.
2. Open the default `Code.gs`, remove the sample function, and paste all of [Code.js](./Code.js).
   Keep the editor filename `Code.gs`: Apps Script uses this extension for JavaScript. Do not paste
   Markdown fences or leave a second copy of the functions in another file.
3. Open **Project Settings** and enable **Show "appsscript.json" manifest file in editor**.
4. Return to **Editor**, open `appsscript.json`, and replace its contents with the complete
   [repository manifest](./appsscript.json). Save both files. See Google's
   [manifest editing guide](https://developers.google.com/apps-script/concepts/manifests).
5. Confirm the editor's **Services** list includes **Sheets**, with identifier `Sheets` and version
   `v4`. The manifest enables it. If absent, check that you saved the manifest; the editor also lets
   you add **Google Sheets API** using **Services → +**.

With Apps Script's default Cloud project, enabling the advanced service also enables its API. If
your organization has associated a **standard Google Cloud project**, enable **Google Sheets API**
in that associated project's API Library as well—not merely in the project hosting authentication.
See [advanced service setup](https://developers.google.com/apps-script/guides/services/advanced).

The supplied manifest uses V8, UTC, and the spreadsheet OAuth scope only. That scope can access
spreadsheets available to the executing owner; it is **not restricted to this one spreadsheet**.
Limit access to the owner account and script project accordingly. Keep the supplied logging settings:
the script deliberately avoids logging request bodies, addresses, signing keys, or Google exceptions.

Do not click **Run** on `doPost` to install or test the integration. It expects an incoming HTTP event
with a signed request; running it in the editor does not provide one. No install function or trigger
is required.

### 3. Set Script Properties

Generate a cryptographically random key in your password manager; a **64-character random hexadecimal
value** is a suitable choice. The minimum is 32 bytes in authentication and 32 characters in the script,
so use ASCII to satisfy both. Do not use an account password, API access token, or an example value.

In **Project Settings → Script Properties**, choose **Add script property** (or **Edit script
properties** if properties already exist), enter these two names and values, and **Save script
properties**. They must be script properties, not user or document properties. Google's
[property management guide](https://developers.google.com/apps-script/guides/properties) shows the controls.

| Property                  | Value                                                                  |
| ------------------------- | ---------------------------------------------------------------------- |
| `WAITLIST_SPREADSHEET_ID` | The ID copied in step 1.                                               |
| `WAITLIST_SECRET`         | The generated signing key, exactly as stored in your password manager. |

Do not add quotation marks, line breaks, or spaces around either value. `LAST_JOIN_AT` is created and
maintained by the script; leave it alone. No deployment URL property is needed here.

Treat script editors as trusted secret holders. Keep the key only in the password manager, Script
Properties, and the authentication runtime's secret store. Never put it in `Code.gs`, the manifest,
a sheet cell, a URL, a shell argument, a screenshot, an issue, or chat.

### 4. Deploy the web app

1. Select **Deploy → New deployment**.
2. Under **Select type**, choose **Web app** (use the deployment-type gear if needed).
3. Add a description identifying the repository release being installed.
4. Set **Execute as** to **Me**, using the organization-controlled owner.
5. Set **Who has access** to **Anyone**, not an option that requires a Google account.
6. Select **Deploy** and complete the owner's authorization when prompted. Check that the project
   and spreadsheet access match what you installed. Stop if unexpected permissions are requested
   or an administrator blocks authorization; do not bypass organization policy.
7. Copy the **Web app URL**, which has this shape:

   ```text
   https://script.google.com/macros/s/<DEPLOYMENT_ID>/exec
   ```

Use the published `/exec` URL, not `/dev`, the script editor URL, the spreadsheet URL, or a redirected
`script.googleusercontent.com` URL. See Google's [web app deployment guide](https://developers.google.com/apps-script/guides/web).

**Anyone applies to the web app endpoint, not spreadsheet sharing.** The signature check protects
operations; an obscure URL is not the security boundary. If Workspace policy does not permit this
deployment, stop and resolve it with the administrator. Do not make the sheet public as a workaround.

Keep the owner, project link, deployment URL, environment, and installed repository release in a
private operations record. Opening `/exec` in a browser is not a functional test: this script has
`doPost`, not `doGet`. Proceed through authentication to test it.

### 5. Connect the authentication runtime

Configure the **REST service** and any **maintenance job that accesses the list**. Do not add these
values to Studio, browser configuration, Docker build arguments, or committed environment files.

| Runtime variable   | Value                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------- |
| `WAITLIST_URL`     | The complete HTTPS `/exec` URL from step 4, without query parameters or a fragment.   |
| `WAITLIST_SECRET`  | A secret-store reference injecting exactly the same signing key as Script Properties. |
| `WAITLIST_TIMEOUT` | Optional duration; defaults to `10s`. Must be greater than zero and at most `20s`.    |

Both URL and key absent disable the integration. Providing only one, an unsafe URL, a short key,
or an out-of-range timeout fails configuration validation. Deploy the script **before** enabling
the matching authentication release; an older script may not understand conflict codes or cleanup.

The existing `SERVICE_AUTHENTICATION_ENV_PREFIX` convention also applies to these runtime variables.
If your deployment uses it, apply the prefix exactly as for its other authentication variables.
Script Property names remain unprefixed. Authentication does not need the spreadsheet ID.

#### Google Cloud / Cloud Run example

1. In the authentication runtime's Cloud project, create a Secret Manager secret, for example
   `authentication-waitlist-signing-key`. Add the exact key from your password manager as its payload.
   This is a runtime secret, not a GitHub repository secret.
2. Grant **Secret Manager Secret Accessor** on this secret to the service identity executing
   authentication. If maintenance uses a different identity, grant it access too. Do not grant all
   project members access or confuse the deployer's account with the runtime identity.
3. In the Cloud Run service's configuration, use **Variables and Secrets → Reference a secret** to
   expose a specific secret version as `WAITLIST_SECRET`. Add `WAITLIST_URL` as an environment variable;
   omit `WAITLIST_TIMEOUT` unless the default needs changing.
4. Deploy the configuration, and configure any maintenance job the same way. Environment-based
   secrets are read when an instance starts; changing the stored value alone does not update existing
   instances. See [Cloud Run secret configuration](https://docs.cloud.google.com/run/docs/configuring/services/secrets).

The runtime identity needs access to **the secret**, not the spreadsheet. Google operations run as
the Apps Script owner. There is no service-account JSON key to download and no production Google
credential to add to CI: builds and automated tests do not call the live sheet.

For another container host, use its runtime secret injection mechanism and restart/redeploy the
service with the same variables. Preserve the existing database, JSON Keys, and SMTP configuration;
this guide only adds the optional waitlist integration.

#### What to share when asking for setup help

Share the environment, authentication release, `/exec` deployment URL through an appropriate private
channel, and confirmation that both secret stores are configured. For a failure, share the HTTP
status, conflict code if present, and time of the attempt. **Never share the key**, bearer token,
private sheet rows, raw request bodies, or an unredacted network capture.

## Verify the installation

Start with a separate test sheet, script project, key, and authentication environment. Use addresses
you control; do not run failure injection or seed test rows in production. Automated tests exercise
local fixtures and cannot validate Google's permissions or your deployed settings.

The easiest end-to-end check is the invitation-list form in Studio connected to that authentication
environment. API callers use `PUT /v2/waitlist`, an anonymous or authenticated bearer session, and
`{ "email": "person@example.com", "lang": "en" }`. Use a real address you control in the test.
The JS package exports `waitlistJoin` and `WaitlistJoinRequestSchema`; see the
[client setup](../../README.md#javascript--typescript). Do not send this public payload directly to
Apps Script: authentication adds the signature and timestamp on the server.

| Check                                                                                        | Expected result                                                                                                                         |
| -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Submit a new address with `lang: "en"` or `"fr"`.                                            | Empty HTTP `202`; exactly one row containing the address, language, and UTC timestamp. No email or account is created.                  |
| Submit the identical address again.                                                          | HTTP `409` with `{"code":"already_waitlisted"}`; still one row with its original language and timestamp.                                |
| Submit an existing account's address.                                                        | HTTP `409` with `{"code":"account_exists"}`; no row is added and no email is sent.                                                      |
| Invite the pending test address through the existing admin flow, then complete registration. | The account is created and its row disappears; other rows and the headers remain. Sending the invitation alone does not remove the row. |
| Try opening the sheet using an unrelated Google account.                                     | No access to the spreadsheet or its data.                                                                                               |
| In the test environment only, temporarily mismatch the runtime key and retry a new address.  | The request fails, no row appears, and the UI does not claim success. Restore the correct key, redeploy, and verify the request works.  |

Wait at least a second between distinct new addresses. On HTTP `429` or `503`, respect the API's
`Retry-After: 60` response and investigate repeated failures rather than submitting in a loop.
After a timeout, a retry may return `already_waitlisted`: the first write can succeed even if its
acknowledgement was lost. Inspect the test sheet before assuming a write failed.

Also verify maintenance in the test environment: pause writers, add a literal row for an existing
test account, then resume. Run [cleanup preview](#on-demand-cleanup) and confirm no rows change.
Run `--apply` and confirm only registered addresses disappear; a second run should remove nothing.
Keep an unregistered test address in the sheet to verify pending requests are preserved.

Before enabling production intake, repeat the successful join, duplicate, existing-account, and
private-sharing checks against the production configuration using controlled addresses. Arrange
any account creation with the normal invitation operator; do not create accounts just to test an
unapproved deployment. Record the result without copying private data into a public issue.

## Operations and limits

- **Exact email matching:** equality follows the account database. Do not lowercase addresses,
  strip dots, or merge aliases. New cells are written as RAW values, so formula-like addresses stay
  literal text.
- **Deduplication:** a shared script lock serializes reads and writes across authentication replicas.
  A second account check after joining handles an account created during the request. Distinct
  `account_exists` and `already_waitlisted` responses deliberately disclose membership; the typed
  client exposes them through `WaitlistJoinConflictError.code`.
- **Capacity:** at most 10,000 entries and one new row per second. Duplicate checks and removals do
  not consume the new-row rate limit, but can still encounter lock contention. The public API allows
  four concurrent waitlist requests per replica and an 8 KiB body. Apply deployment-level abuse
  protection; an anonymous token is not proof of a person's identity.
- **Availability:** HTTP `429` means writer contention, throttling, or capacity. HTTP `503` means
  unavailable/disabled storage or a full per-instance concurrency gate. Neither is success. A green
  readiness healthcheck does **not** prove Google is working; this optional dependency is not probed
  there, so monitor waitlist failures separately.
- **Account cleanup:** after committing the account, registration attempts removal before signing
  session tokens, with a three-second budget independent of client cancellation. A failure is recorded
  in tracing but does not undo the account. There is no durable retry queue; run maintenance after an
  outage and before inviting from the list.
- **Google transport:** signed requests accept a five-minute clock difference. Authentication follows
  only one HTTPS response redirect, as a GET to `script.googleusercontent.com`; it never forwards the
  signed POST to another host. Outbound access must allow both Google hosts.
- **Privacy:** rows are unverified requests, not proof of email ownership or marketing consent.
  Recheck account existence before sending invitations, restrict operator access, and agree a
  retention period. Keep exported copies private and remove them under the same policy.

### On-demand cleanup

Use the maintenance image from the same authentication release, with access to its database and the
waitlist runtime settings above. This operation needs neither SMTP nor JSON Keys. The commands below
are the maintenance executable's invocation; for a job whose entrypoint is already `maintenance`,
set its arguments to `waitlist-cleanup` or `waitlist-cleanup --apply`.

Preview first:

```text
maintenance waitlist-cleanup
```

Check the target environment and reported counts. To remove rows belonging to registered accounts:

```text
maintenance waitlist-cleanup --apply
```

Preview does not change the sheet. Apply checks the database again and removes only matching
addresses, including duplicate rows; it preserves pending requests and sends no invitations.
The command logs `scanned` unique addresses, `matched` accounts, and `removed` rows, never email values.

A run reads at most 100 addresses per request and 10,000 overall, with a ten-minute deadline.
Its email cursor tolerates deletions but is not a snapshot: new addresses before the current cursor
are checked on the next run. There is no automatic schedule.

A failure exits nonzero and reports partial counts. Deletions can commit before an acknowledgement
is lost, so `removed` may undercount. Fix the cause and rerun: already-removed rows are absent and
remaining addresses are checked again. Do not restore stale rows merely to reproduce the count.

### Update or roll back the script

Saving editor changes does **not** update the published `/exec` deployment. For an update:

1. Test the matching release's `Code.js` and manifest in the separate test environment first.
2. Record the currently deployed script version and authentication release for recovery.
3. Replace and save both production editor files with the tested files. Preserve Script Properties.
4. Use **Deploy → Manage deployments**, select the active web app, choose **Edit**, select
   **New version**, and deploy. Reuse the existing deployment rather than creating a replacement URL.
5. Verify the live flow, then enable the matching authentication release if it requires the new script.

To roll back, edit that same deployment and select the previous known-good version. Confirm it is
compatible with the running authentication release; otherwise pause intake and roll back the pair.
A code rollback does not restore deleted sheet rows, Script Properties, or runtime secrets.
See Google's [versioned deployment guide](https://developers.google.com/apps-script/concepts/deployments).

Keep the project and deployment owner stable. A deployment remains associated with its creator even
when project ownership changes; plan and test an ownership transition before retiring that account.

### Rotate the signing key

Only one key is accepted at a time, so plan a short maintenance window:

1. Pause invitation intake, account-completion traffic that could attempt cleanup, and cleanup jobs;
   drain in-flight operations. Alternatively, disable the integration by removing **both** runtime
   settings on all instances; accounts created while disabled will need maintenance cleanup later.
2. Generate a fresh key, update `WAITLIST_SECRET` in Script Properties, and add a runtime secret version.
3. Point all authentication instances and maintenance jobs at that version and redeploy. Re-enable
   both runtime settings if they were removed. Do not send traffic to mixed old/new configurations.
4. Verify a controlled request, resume traffic, and run cleanup if registration continued while disabled.
5. Revoke the old secret version once no running revision or job still needs it.

Changing Script Properties takes effect without a new code deployment. Rolling back only the service
revision after rotation may restore an old secret reference and break signing; coordinate both ends.

## Troubleshooting

The service intentionally hides private Google error details. Check configuration through the private
consoles rather than adding payload logging or pasting secrets into a diagnostic command.

| Symptom                                                                                      | Check / action                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/exec` in a browser reports a missing `doGet`, or **Run** in the editor does not add a row. | Expected: neither sends the signed POST event. Test through the authentication API or Studio instead.                                                                                                                           |
| Deployment asks callers to sign in to Google or returns an HTML page.                        | Confirm **Execute as: Me**, **Who has access: Anyone**, and the published `/exec` URL. Resolve Workspace restrictions with an administrator; leave the sheet restricted.                                                        |
| Authentication fails at startup after configuration.                                         | Check that URL and key are both present, the URL has the exact accepted shape, the key meets its minimum, and the timeout is a duration in range. Check environment prefixes and runtime secret access without printing values. |
| Every new join returns `503`.                                                                | Check both Script Properties, exact tab/header spelling, the owner's access to the sheet, matching keys, the advanced Sheets service/API, and the deployed version. A disabled integration also returns `503`.                  |
| Requests fail even though the keys match.                                                    | Check runtime clock synchronization (signatures expire after five minutes), network access to both Google hosts, and that no proxy replaces Google's JSON response with a login/error page.                                     |
| Duplicate checks work, but appending a new address fails.                                    | Recheck the `Sheets` v4 advanced service and Google Sheets API in the script's associated Cloud project; appends use that API.                                                                                                  |
| New joins return `429`.                                                                      | Wait for `Retry-After`, then inspect contention and row count. At 10,000 entries, run cleanup for registered accounts or stop intake; do not delete pending requests just to free space.                                        |
| An address is reported as already waiting after a timeout.                                   | The first write may have succeeded. This warning does not mean there are two rows; inspect the sheet privately.                                                                                                                 |
| A created account still has a row.                                                           | Run cleanup preview, then apply after reviewing counts. Registration deliberately survives Google failures; check waitlist tracing separately from account creation.                                                            |
| Cleanup fails on a sheet that contains data.                                                 | Check for blank/invalid email cells, extra headings, or a tab over the entry limit. Pause writers before any repair, retain a private recovery copy, and rerun preview after fixing the data.                                   |
| Code was changed but behavior is unchanged.                                                  | Saving does not publish. Check **Manage deployments** and create a new version for the existing deployment; also check that the server still points to it.                                                                      |
| Healthchecks are green but the form fails.                                                   | Expected possibility: general readiness excludes this optional integration. Verify the waitlist through a controlled API request.                                                                                               |
| Google reports quota, access, or authorization errors.                                       | Check the owner's authorization and associated Cloud project, then consult [Apps Script quotas](https://developers.google.com/apps-script/guides/services/quotas). Do not retry aggressively or broaden sharing.                |

## Retire the temporary flow

Restore public registration permissions and the Studio form in a coordinated release. Remove both
waitlist runtime settings, archive the web app deployment, remove its signing-key property, and
revoke runtime secret access. Retain or delete the sheet and any exports according to the agreed
retention policy. Keep invitation-completion routes available while previously issued links remain
valid.
