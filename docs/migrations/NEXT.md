# Upgrading to NEXT

`@a-novel/service-authentication-rest-test` now holds the sign-up helpers other services' test suites need: the super-admin sign-in, invitations, user and anonymous tokens, and one MailPit poller. It no longer depends on vitest, jsdom or mailparser.

## Do I need to act?

**Required** for a suite that imports `@a-novel/service-authentication-rest-test`: `checkEmail` and `preRegisterUser` changed shape, and two helpers are gone. No repository outside service-authentication imports it today. **Recommended** for a suite that copies the sign-up helpers (service-profile and service-narrative-engine): import them instead.

## Read MailPit's address from `MAIL_UI_URL`

The helpers find MailPit through `MAIL_UI_URL` and take no mail host argument. `MAIL_HOST` is no longer read.

```diff
-const mail = await checkEmail(mailHost, `to:"${email}"`);
-const html = mail.html;
+const mail = await checkEmail(`to:"${email}"`);
+const html = mail.HTML;
```

`checkEmail` returns MailPit's message (`ID`, `Subject`, `HTML`) instead of a parsed mail, always deletes the message it reads, and throws an `Error` when none arrives within its wait, 3 seconds by default.

## Drop the mail host from the registration helpers

```diff
-const invitation = await preRegisterUser(api, mailHost, email);
+const invitation = await preRegisterUser(api, email);
 const user = await registerUser(api, invitation);
```

`registerUser(api)` without an invitation invites a random address first. `preRegisterUser` throws when the invitation link carries no short code for the address, instead of failing a vitest assertion.

## Replace the removed helpers

- `getHtmlMail(raw, selector)` is gone: `mailLinks(html)` returns the links of an email, in order.
- `expectAddress` is gone: compare the addresses your test reads directly.

## Import the sign-up helpers instead of copying them

A suite that copies the sign-up helpers keeps a `token.ts` and a `setup.ts` in `pkg/js/test/rest/src`. Replace them with the package:

1. Add the package from the repository root, with a caret range on its latest release:

   ```bash
   pnpm add -D -w @a-novel/service-authentication-rest-test
   ```

   It goes in the root `package.json`, beside `@a-novel/service-authentication-rest`. Renovate never reads a manifest under a `test/` directory, so the test project declares no dependency of its own.

2. Delete `pkg/js/test/rest/src/token.ts`.
3. Replace `pkg/js/test/rest/src/setup.ts`, which `vitest.config.ts` keeps as the suite's `globalSetup`:

   ```ts
   import { AuthenticationApi } from "@a-novel/service-authentication-rest";
   import { registerSuperAdmin } from "@a-novel/service-authentication-rest-test";

   /** Completes the super-admin invitation the authentication service mails at startup. */
   export default async function setup() {
     await registerSuperAdmin(new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!));
   }
   ```

4. In every file that imports `./token`, import the helpers from the package and pass them a client of the authentication service:

   ```diff
   -import { anonymousAccessToken, superAdminAccessToken, userAccessToken } from "./token";
   +import { AuthenticationApi } from "@a-novel/service-authentication-rest";
   +import { anonymousAccessToken, superAdminAccessToken, userAccessToken } from "@a-novel/service-authentication-rest-test";
   +
   +const authApi = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);

   -const token = await userAccessToken();
   +const token = await userAccessToken(authApi);
   ```

   `superAdminAccessToken(authApi)` and `anonymousAccessToken(authApi)` change alike. `superAdminCredentials()` takes no client.

The package behaves differently from the copies in three ways:

- An invitation must arrive within 3 seconds in a test, and within 15 seconds in the global setup. The copies waited 15 seconds everywhere.
- The helper reads the newest mail sent to the address, deletes it, and takes its first link. The copies searched the ten newest mails for an account-creation link and kept them.
- A new user gets a random `@provider.com` address and a random password, instead of an `@agorastoryverse.com` address and `integration-user`.

The super-admin defaults stay `noreply@agorastoryverse.com` and `admin`, overridden by `SUPER_ADMIN_EMAIL` and `SUPER_ADMIN_PASSWORD`. MailPit's address comes from `MAIL_UI_URL`.
