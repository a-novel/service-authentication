# Install the private invitation list

This guide is about the private invitation list. The Google Apps Script in this directory stores
invitation requests in a private Google Sheet, and the authentication service writes to it over
signed requests. The list needs no database migration.

Joining the list creates no account, sends no email and grants no permission to register. Accounts
still come from an admin invitation. Once a registration completes, authentication removes the
matching row.

## Before you start

You need:

- An organization-controlled Google account that can create a spreadsheet and deploy an Apps Script
  web app reachable without a Google sign-in. Check that your Workspace policy allows it.
- Permission to configure and redeploy the authentication REST service, and to manage its runtime
  secrets. Maintenance jobs that read the list need the same configuration.
- These files, taken from the [release](https://github.com/a-novel/service-authentication/releases)
  of the authentication service you deploy:

  | File                                 | Use                                                             |
  | ------------------------------------ | --------------------------------------------------------------- |
  | [Code.js](./Code.js)                 | Its whole content goes into the Apps Script editor's `Code.gs`. |
  | [appsscript.json](./appsscript.json) | Its whole content goes into the project's manifest.             |
  | [Code.test.ts](./Code.test.ts)       | Local automated tests. It never goes to Google.                 |

- A password manager, to generate and keep a random signing key.
- A separate test environment, and email addresses you control, for the verification steps.

The installation runs in the browser. It needs no local build, npm package, `clasp`, service-account
key, trigger or scheduled task. The authentication database stays the source of truth for accounts,
and the sheet holds pending requests only.

Two permission boundaries protect the list:

```text
Studio → authentication API → signed server request → Apps Script → private Google Sheet
                            WAITLIST_SECRET          executes as the Google owner
```

The browser never receives the signing key and never reaches Google. The web app accepts requests
without a Google sign-in, and the script checks a recent HMAC signature before it touches the sheet.
The sheet itself stays restricted. Use one sheet, one script project and one signing key per
environment. Two script projects cannot share the lock that deduplicates requests.

## Installation

### 1. Create the spreadsheet

1. Sign in with the organization-controlled account that will own the deployment.
2. Create a blank spreadsheet, for example **Invitation list — staging**. The script ignores the
   title.
3. Open **Share** and keep general access **Restricted**. Give access only to the operators who need
   it. Keep link sharing and publishing to the web off.
4. Rename the first sheet tab to exactly `Waitlist`, with a capital `W`.
5. Enter these exact lowercase headers, one per cell, with no surrounding spaces:

   | Cell | Value          |
   | ---- | -------------- |
   | A1   | `email`        |
   | B1   | `language`     |
   | C1   | `requested_at` |

6. Leave every data row empty: no sample row, formula, merged cell or extra heading.
7. Save the spreadsheet ID privately. It is the part after `/d/` in the browser URL:

   ```text
   https://docs.google.com/spreadsheets/d/<SPREADSHEET_ID>/edit
   ```

Copy the ID alone, without the rest of the URL or the `gid` of a tab. The ID is configuration rather
than a credential, but keep it out of public issues, like the sheet's contents.

The script owns the `Waitlist` tab. While it is active, nobody sorts it, inserts blank rows, appends
data by hand or writes through another script: a script lock cannot hold back human editors. Analyze
a separate copy, and never connect that copy to another live writer for the same list.

### 2. Install both script files

1. From the spreadsheet, open **Extensions → Apps Script**. Name the project after its purpose and
   environment, for example **Invitation list — staging**.
2. Open the default `Code.gs`, remove the sample function, and paste all of [Code.js](./Code.js). Keep
   the editor filename `Code.gs`, the extension Apps Script uses for JavaScript. Paste the code alone,
   without Markdown fences, and keep no second copy of the functions in another file.
3. Open **Project Settings** and enable **Show "appsscript.json" manifest file in editor**.
4. Return to **Editor**, open `appsscript.json`, and replace its contents with the complete
   [repository manifest](./appsscript.json). Save both files. Google's
   [manifest editing guide](https://developers.google.com/apps-script/concepts/manifests) shows the
   controls.
5. Check that the editor's **Services** list includes **Sheets**, with identifier `Sheets` and version
   `v4`. The manifest enables it. If it is missing, check that you saved the manifest, or add
   **Google Sheets API** with **Services → +**.

With Apps Script's default Cloud project, enabling the advanced service also enables its API. If
your organization associated a **standard Google Cloud project**, enable **Google Sheets API** in that
project's API Library too. The project hosting authentication does not count. Google's
[advanced service setup](https://developers.google.com/apps-script/guides/services/advanced) covers
both cases.

The supplied manifest uses V8, UTC and the spreadsheet OAuth scope only. That scope reaches every
spreadsheet the executing owner can open, **not only this one**, so limit access to the owner account
and the script project. Keep the supplied logging settings: the script logs no request body,
address, signing key or Google exception.

The integration needs no install function and no trigger. **Run** on `doPost` tests nothing: the
function expects an incoming HTTP event carrying a signed request, which the editor does not send.

### 3. Set Script Properties

Generate a cryptographically random key in your password manager. A **64-character random
hexadecimal value** suits both ends: authentication wants at least 32 bytes and the script at least
32 characters, which an ASCII key satisfies at once. Never reuse an account password, an API access
token or an example value.

In **Project Settings → Script Properties**, choose **Add script property**, or **Edit script
properties** when some exist. Enter these two names and values, then **Save script properties**.
They must be script properties, not user or document properties. Google's
[property management guide](https://developers.google.com/apps-script/guides/properties) shows the
controls.

| Property                  | Value                                                                  |
| ------------------------- | ---------------------------------------------------------------------- |
| `WAITLIST_SPREADSHEET_ID` | The ID copied in step 1.                                               |
| `WAITLIST_SECRET`         | The generated signing key, exactly as stored in your password manager. |

Enter both values bare, with no quotation marks, line breaks or surrounding spaces. The script creates
and maintains `LAST_JOIN_AT` itself, so leave it alone. Script Properties hold no deployment URL.

Script editors can read the key, so treat them as trusted secret holders. The key lives in three
places only: the password manager, Script Properties and the authentication runtime's secret store.
Never put it in `Code.gs`, the manifest, a sheet cell, a URL, a shell argument, a screenshot, an
issue or a chat.

### 4. Deploy the web app

1. Select **Deploy → New deployment**.
2. Under **Select type**, choose **Web app**, through the deployment-type gear if needed.
3. Add a description naming the repository release you install.
4. Set **Execute as** to **Me**, the organization-controlled owner.
5. Set **Who has access** to **Anyone**. The options that require a Google account do not work.
6. Select **Deploy**, and complete the owner's authorization when prompted. Check that the project
   and spreadsheet access match what you installed. Stop when unexpected permissions are requested
   or an administrator blocks authorization, and leave organization policy in place.
7. Copy the **Web app URL**, which has this shape:

   ```text
   https://script.google.com/macros/s/<DEPLOYMENT_ID>/exec
   ```

Use the published `/exec` URL. The `/dev` URL, the script editor URL, the spreadsheet URL and a
redirected `script.googleusercontent.com` URL all fail. Google's
[web app deployment guide](https://developers.google.com/apps-script/guides/web) covers the
deployment.

**Anyone applies to the web app endpoint, not to the spreadsheet's sharing.** The signature check
protects every operation, and keeping the URL obscure adds no protection. If Workspace policy refuses
the deployment, stop and resolve it with the administrator, and keep the sheet private.

Record the owner, the project link, the deployment URL, the environment and the installed repository
release in a private operations record. Opening `/exec` in a browser tests nothing, since the script
handles `doPost` alone. Test it through authentication.

### 5. Connect the authentication runtime

Configure the **REST service** and every **maintenance job that reads the list**. Keep these values
in the runtime: out of Studio, browser configuration, Docker build arguments and committed
environment files.

| Runtime variable   | Value                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------- |
| `WAITLIST_URL`     | The complete HTTPS `/exec` URL from step 4, without query parameters or a fragment.   |
| `WAITLIST_SECRET`  | A secret-store reference injecting exactly the same signing key as Script Properties. |
| `WAITLIST_TIMEOUT` | Optional duration, `10s` by default. It must be greater than zero and at most `20s`.  |

With both the URL and the key absent, the integration is off. A lone URL or key, an unsafe URL, a
short key or an out-of-range timeout fails configuration validation. Deploy the script **before** the
matching authentication release, since an older script may not understand conflict codes or cleanup.

A deployment that sets `SERVICE_AUTHENTICATION_ENV_PREFIX` prefixes these variables like the others.
Script Property names take no prefix. Authentication does not need the spreadsheet ID.

#### Google Cloud / Cloud Run example

1. In the authentication runtime's Cloud project, create a Secret Manager secret, for example
   `authentication-waitlist-signing-key`. Add the exact key from your password manager as its payload.
   It is a runtime secret, not a GitHub repository secret.
2. Grant **Secret Manager Secret Accessor** on this secret to the service identity that runs
   authentication, and to the maintenance identity when it differs. Grant it to those identities
   alone: neither every project member nor the deployer's own account needs it.
3. In the Cloud Run service's configuration, use **Variables and Secrets → Reference a secret** to
   expose a specific secret version as `WAITLIST_SECRET`. Add `WAITLIST_URL` as an environment
   variable, and set `WAITLIST_TIMEOUT` only to change its default.
4. Deploy the configuration, and configure every maintenance job the same way. An instance reads its
   environment secrets when it starts, so changing the stored value alone leaves running instances on
   the old one. Google's
   [Cloud Run secret configuration](https://docs.cloud.google.com/run/docs/configuring/services/secrets)
   covers the details.

The runtime identity needs access to **the secret**, not to the spreadsheet: Google operations run as
the Apps Script owner. No service-account JSON key exists to download, and CI needs no production
Google credential, since builds and automated tests never call the live sheet.

On another container host, inject the secret through its runtime secret mechanism and redeploy the
service with the same variables. Keep the existing database, JSON Keys and SMTP configuration.

#### What to share when asking for setup help

Share the environment, the authentication release, the `/exec` deployment URL over a private
channel, and confirmation that both secret stores are configured. For a failure, add the HTTP
status, the conflict code if any, and the time of the attempt. **Never share the key**, a bearer
token, private sheet rows, raw request bodies or an unredacted network capture.

## Verify the installation

Start with a separate test sheet, script project, key and authentication environment, and use
addresses you control. Failure injection and test rows stay out of production. Automated tests run
against local fixtures, so they cannot check Google's permissions or your deployed settings.

The invitation-list form in Studio, connected to that authentication environment, gives the easiest
end-to-end check. An API caller sends `PUT /v2/waitlist` with an anonymous or authenticated bearer
session and `{ "email": "person@example.com", "lang": "en" }`, using a real address it controls. The
JS package exports `waitlistJoin` and `WaitlistJoinRequestSchema`, set up as the
[client setup](../../README.md#javascript--typescript) shows. Send this public payload to
authentication alone, which adds the signature and timestamp on the server.

| Check                                                                                        | Expected result                                                                                                                         |
| -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Submit a new address with `lang: "en"` or `"fr"`.                                            | Empty HTTP `202`; exactly one row containing the address, language, and UTC timestamp. No email or account is created.                  |
| Submit the identical address again.                                                          | HTTP `409` with `{"code":"already_waitlisted"}`; still one row with its original language and timestamp.                                |
| Submit an existing account's address.                                                        | HTTP `409` with `{"code":"account_exists"}`; no row is added and no email is sent.                                                      |
| Invite the pending test address through the existing admin flow, then complete registration. | The account is created and its row disappears; other rows and the headers remain. Sending the invitation alone does not remove the row. |
| Try opening the sheet using an unrelated Google account.                                     | No access to the spreadsheet or its data.                                                                                               |
| In the test environment only, temporarily mismatch the runtime key and retry a new address.  | The request fails, no row appears, and the UI does not claim success. Restore the correct key, redeploy, and verify the request works.  |

Wait at least a second between distinct new addresses, and send at most 3 requests per 5 minutes
from one address by default. On HTTP `429` or `503`, respect the API's `Retry-After` response, and
investigate repeated failures instead of submitting in a loop. A retry after a timeout may return
`already_waitlisted`, because the first write can succeed even when its acknowledgement was lost.
Inspect the test sheet before assuming a write failed.

Verify maintenance in the test environment too. Pause the writers, add a literal row for an existing
test account, then resume. Run the [cleanup preview](#on-demand-cleanup) and confirm no row changes.
Run `--apply` and confirm only registered addresses disappear, then run it again to see it remove
nothing. Keep an unregistered test address in the sheet, to verify that pending requests stay.

Before enabling production intake, repeat the successful join, duplicate, existing-account and
private-sharing checks against the production configuration with controlled addresses. Arrange any
account creation with the usual invitation operator, and create no account just to test a deployment
nobody approved. Record the result without copying private data into a public issue.

## Operations and limits

- **Exact email matching:** equality follows the account database, so addresses keep their case,
  dots and aliases. New cells are written as RAW values, so formula-like addresses stay literal text.
- **Deduplication:** a shared script lock serializes reads and writes across authentication replicas.
  A second account check after joining catches an account created during the request. The distinct
  `account_exists` and `already_waitlisted` responses disclose membership, which the UI's warnings
  need, and the typed client exposes them as `WaitlistJoinConflictError.code`.
- **Capacity:** the list holds at most 10,000 entries and takes one new row per second. Duplicate
  checks and removals spend no new-row budget, but can still meet lock contention. The public API
  allows four concurrent waitlist requests per replica, 3 requests per 5 minutes per caller, and an
  8 KiB body by default (`WAITLIST_MAX_CONCURRENT`, `REST_RATE_LIMIT_STRICT`,
  `WAITLIST_MAX_REQUEST_SIZE`). An anonymous token proves nobody's identity.
- **Availability:** HTTP `429` means too many requests, writer contention or a full list. HTTP `503`
  means unavailable or disabled storage. Neither is a success. The readiness healthcheck does not
  probe this optional dependency, so a green healthcheck says nothing about Google: monitor waitlist
  failures separately.
- **Account cleanup:** once the account is committed, registration removes its row before signing the
  session tokens, within three seconds and whatever the client does. A failed removal is traced and
  keeps the account. No durable queue retries it, so run maintenance after an outage and before
  inviting from the list.
- **Google transport:** a signed request stays valid for five minutes of clock difference.
  Authentication follows a single HTTPS response redirect, as a GET to `script.googleusercontent.com`,
  and never forwards the signed POST to another host. Outbound access must reach both Google hosts.
- **Privacy:** rows are unverified requests, neither proof of email ownership nor marketing consent.
  Recheck account existence before sending invitations, restrict operator access, and agree on a
  retention period. Keep exported copies private, and remove them under the same policy.

### On-demand cleanup

Use the maintenance image of the same authentication release, with access to its database and the
waitlist runtime settings above. This operation needs neither SMTP nor JSON Keys. The commands below
invoke the maintenance executable. A job whose entrypoint is already `maintenance` takes
`waitlist-cleanup` or `waitlist-cleanup --apply` as its arguments.

Preview first:

```text
maintenance waitlist-cleanup
```

Check the target environment and the reported counts, then remove the rows of registered accounts:

```text
maintenance waitlist-cleanup --apply
```

The preview leaves the sheet unchanged. Apply checks the database again and removes only matching
addresses, duplicate rows included. It keeps pending requests and sends no invitation. The command
logs the `scanned` unique addresses, the `matched` accounts and the `removed` rows, never an email.

A run reads at most 100 addresses per request and 10,000 overall, within ten minutes. Its email
cursor tolerates deletions but takes no snapshot: an address added behind the cursor waits for the
next run. No schedule runs the cleanup.

A failure exits nonzero and reports partial counts. A deletion can commit before its acknowledgement
is lost, so `removed` may undercount. Fix the cause and rerun: removed rows are gone, and the
remaining addresses are checked again. Leave stale rows deleted, even to reproduce a count.

### Update or roll back the script

Saving editor changes leaves the published `/exec` deployment unchanged. To update it:

1. Test the matching release's `Code.js` and manifest in the separate test environment first.
2. Record the deployed script version and the authentication release, for recovery.
3. Replace and save both production editor files with the tested ones. Keep Script Properties.
4. Use **Deploy → Manage deployments**, select the active web app, choose **Edit**, select **New
   version** and deploy. Reuse the existing deployment, which keeps its URL.
5. Verify the live flow, then enable the matching authentication release if it needs the new script.

To roll back, edit that same deployment and select the previous known-good version. Check that it
works with the running authentication release, or pause intake and roll back the pair. A code
rollback restores no deleted sheet row, Script Property or runtime secret. Google's
[versioned deployment guide](https://developers.google.com/apps-script/concepts/deployments) covers
versions.

Keep the project and deployment owner stable. A deployment stays tied to its creator even when the
project changes owner, so plan and test an ownership transfer before retiring that account.

### Rotate the signing key

The script accepts one key at a time, so plan a short maintenance window:

1. Pause invitation intake, account-completion traffic that could attempt cleanup, and cleanup jobs,
   then drain in-flight operations. Alternatively, disable the integration by removing **both**
   runtime settings on every instance. Accounts created while it is disabled need a maintenance
   cleanup later.
2. Generate a fresh key, update `WAITLIST_SECRET` in Script Properties, and add a runtime secret
   version.
3. Point every authentication instance and maintenance job at that version, and redeploy. Restore
   both runtime settings if you removed them. Send no traffic to a mix of old and new configurations.
4. Verify a controlled request, resume traffic, and run a cleanup if registration continued while the
   integration was disabled.
5. Revoke the old secret version once no running revision or job needs it.

A Script Properties change takes effect without a new code deployment. Rolling back the service
revision alone after a rotation may restore the old secret reference and break signing, so coordinate
both ends.

## Troubleshooting

The service hides Google's private error details. Check the configuration through the private
consoles, and add no payload logging or secret to a diagnostic command.

| Symptom                                                                                      | Check / action                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/exec` in a browser reports a missing `doGet`, or **Run** in the editor does not add a row. | Expected: neither sends the signed POST event. Test through the authentication API or Studio instead.                                                                                                                           |
| Deployment asks callers to sign in to Google or returns an HTML page.                        | Confirm **Execute as: Me**, **Who has access: Anyone**, and the published `/exec` URL. Resolve Workspace restrictions with an administrator; leave the sheet restricted.                                                        |
| Authentication fails at startup after configuration.                                         | Check that URL and key are both present, the URL has the exact accepted shape, the key meets its minimum, and the timeout is a duration in range. Check environment prefixes and runtime secret access without printing values. |
| Every new join returns `503`.                                                                | Check both Script Properties, exact tab/header spelling, the owner's access to the sheet, matching keys, the advanced Sheets service/API, and the deployed version. A disabled integration also returns `503`.                  |
| Requests fail even though the keys match.                                                    | Check runtime clock synchronization (signatures expire after five minutes), network access to both Google hosts, and that no proxy replaces Google's JSON response with a login/error page.                                     |
| Duplicate checks work, but appending or removing an address fails.                           | Recheck the `Sheets` v4 advanced service and Google Sheets API in the script's associated Cloud project; appends and removals use that API.                                                                                     |
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
revoke runtime secret access. Keep or delete the sheet and its exports under the agreed retention
policy. Keep the invitation-completion routes available while previously issued links stay valid.
