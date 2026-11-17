# Upgrading to NEXT

The Go client names its claims helpers after what they do and reads the request's user for you, and the service reads its JSON Keys token from `SERVICE_JSON_KEYS_TOKEN_PATH`. `@a-novel/service-authentication-rest-test` now holds the sign-up helpers other services' test suites need: the super-admin sign-in, invitations, user and anonymous tokens, and one MailPit poller. It no longer depends on vitest, jsdom or mailparser.

## Do I need to act?

**Recommended** for a Go consumer and a deployment: nothing breaks, since the old function and the old variable keep working in this release. Move them to the new helpers and the new variable before a later release removes the old names.

**Required** for a suite that imports `@a-novel/service-authentication-rest-test`: `checkEmail` and `preRegisterUser` changed shape, and two helpers are gone. No repository outside service-authentication imports it today. **Recommended** for a suite that copies the sign-up helpers (service-profile and service-narrative-engine): import them instead.

**Required** for a JavaScript consumer of `@a-novel/service-authentication-rest`: it needs `@a-novel-kit/nodelib-browser` 2.13.0 or later.

## Call RequireClaimsContext instead of MustGetClaimsContext

`MustGetClaimsContext` never panicked, as its `Must` prefix suggests: it returns an error when the context carries no claims. `RequireClaimsContext` does the same under a name that says so. `MustGetClaimsContext` stays as an alias, without a deprecation marker so a consumer's staticcheck stays green; prefer the new name, and a later minor removes the alias once no consumer calls it.

```diff
-claims, err := serviceauthentication.MustGetClaimsContext(r.Context())
+claims, err := serviceauthentication.RequireClaimsContext(r.Context())
```

## Read the request's user with RequestUser

`RequestUser(ctx)` returns the ID of the user an authenticated request acts for. It fails as `RequireClaimsContext` does without claims, and with `ErrAnonymousCaller` when the claims name no user, as an anonymous session's do. It replaces a local helper such as this one:

```diff
-func requestOwner(r *http.Request) (uuid.UUID, error) {
-	claims, err := serviceauthentication.MustGetClaimsContext(r.Context())
-	if err != nil {
-		return uuid.Nil, fmt.Errorf("get claims: %w", err)
-	}
-
-	if claims.UserID == nil {
-		return uuid.Nil, errAnonymousOwner
-	}
-
-	return *claims.UserID, nil
-}
-
-ownerID, err := requestOwner(r)
+ownerID, err := serviceauthentication.RequestUser(r.Context())
```

Code that checked its own anonymous sentinel with `errors.Is` checks `serviceauthentication.ErrAnonymousCaller` instead, and the local sentinel goes. An error map keyed on `nil`, such as `httpf.ErrMap{nil: http.StatusForbidden}`, keeps answering 403 for both refusals: a nil key sets the status of every error the map does not name.

`GetClaimsContext` keeps its signature, but its error is now always nil: only `SetClaimsContext` stores claims, so the context never holds a value of another type. A check of that error can go.

## Rename SERVICE_TOKEN_PATH to SERVICE_JSON_KEYS_TOKEN_PATH

The variable names the token file for the JSON Keys audience, so it takes the name the other services use. The service reads `SERVICE_JSON_KEYS_TOKEN_PATH`, and `SERVICE_TOKEN_PATH` when the new one is unset. A later release stops reading `SERVICE_TOKEN_PATH`.

```diff
-SERVICE_TOKEN_PATH=/var/run/secrets/tokens/json-keys
+SERVICE_JSON_KEYS_TOKEN_PATH=/var/run/secrets/tokens/json-keys
```

The images `rest` and `standalone-rest` read it. When both are set, `SERVICE_JSON_KEYS_TOKEN_PATH` wins, so a deployment can set the new name before it removes the old one.

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

   It goes in the root `package.json`. Move `@a-novel/service-authentication-rest` there too if the test project declares it, with every other entry of the test project's `package.json`: Renovate never reads a manifest under a `test/` directory, so the test project declares no dependency of its own.

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

## Install nodelib-browser 2.13 with the JavaScript client

`AuthenticationApi` now extends `RestClient`, which `@a-novel-kit/nodelib-browser` 2.13.0 introduced, so a consumer on an older nodelib-browser fails to build. Update nodelib-browser before the client, or in the same change:

```bash
pnpm add @a-novel-kit/nodelib-browser@^2.13.0
```

Every name and signature of the client stays. Its `fetch`, `fetchVoid` and `fetchResponse` take a `RestInit`, a `RequestInit` that also accepts `accessToken` and `json`. `fetchVoid` now releases the response body, and a request without a body no longer sends `Content-Type`.

## Notes

- `NewAuthHandler` reports a role that inherits an undefined role as an unknown role at startup. It reported a circular dependency before; both stop the process.
- The `jobs/maintenance` image records each run as a root span named after its operation, such as `job.AccountRole`, which carries the failure and its `error.type`. `account-role` now ends after 5 minutes, like the purges.
- `/v2/healthcheck` runs its probes at once, so it answers within the slowest probe instead of their sum. Its body and status stay the same. Each probe's span takes the probe's name, such as `rest.Health(client:smtp)` instead of `rest.Health(reportSMTP)`.
