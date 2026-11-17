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

`superAdminCredentials()`, `superAdminAccessToken(api)`, `anonymousAccessToken(api)`, `userAccessToken(api)` and `registerSuperAdmin(api)` take an `AuthenticationApi`. Delete a local copy of them and wire the super-admin bootstrap as the suite's global setup:

```ts
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { registerSuperAdmin } from "@a-novel/service-authentication-rest-test";

/** Completes the super-admin invitation the authentication service mails at startup. */
export default async function setup() {
  await registerSuperAdmin(new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!));
}
```

The super-admin defaults stay `noreply@agorastoryverse.com` and `admin`, overridden by `SUPER_ADMIN_EMAIL` and `SUPER_ADMIN_PASSWORD`.
