# Authentication REST test helpers

Helpers for integration tests that run against the [authentication service](https://github.com/a-novel/service-authentication). They read the emails the service sends through MailPit, register test accounts, and sign them in.

Install the package as a development dependency, from the registry the [REST client](https://github.com/a-novel/service-authentication#javascript--typescript) uses:

```bash
pnpm add -D @a-novel/service-authentication-rest-test
```

The helpers read MailPit's API address from `MAIL_UI_URL`, and the super-admin's sign-in from `SUPER_ADMIN_EMAIL` and `SUPER_ADMIN_PASSWORD`, which default to the test stacks' `noreply@agorastoryverse.com` and `admin`. The service mails the super-admin an invitation when it starts. A vitest `globalSetup` completes it once:

```ts
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { registerSuperAdmin } from "@a-novel/service-authentication-rest-test";

export default async function setup() {
  await registerSuperAdmin(new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!));
}
```

The tests then sign in with `superAdminAccessToken(api)`, `userAccessToken(api)`, which registers a new account, or `anonymousAccessToken(api)`.
