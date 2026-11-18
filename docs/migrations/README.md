# Migration guides

- [v2.28.0](./v2.28.0.md): The service no longer reads `SERVICE_TOKEN_PATH`: its JSON Keys token comes from `SERVICE_JSON_KEYS_TOKEN_PATH` alone, as v2.27.0 announced.
- [v2.27.0](./v2.27.0.md): The Go client names its claims helpers after what they do and reads the request's user for you, and the service reads its JSON Keys token from `SERVICE_JSON_KEYS_TOKEN_PATH`. `@a-novel/service-authentication-rest-test` now holds the sign-up helpers other services' test suites need: the super-admin sign-in, invitations, user and anonymous tokens, and one MailPit poller. It no longer depends on vitest, jsdom or mailparser. The JavaScript client, `@a-novel/service-authentication-rest`, now builds on nodelib-browser's `RestClient`, so it requires `@a-novel-kit/nodelib-browser` 2.13.0 or later.
- [v2.24.0](v2.24.0.md): Send the current password and notice language on email changes.
- [v2.16.0](v2.16.0.md): Configure PostgreSQL through discrete settings, not a URL.
- [v2.10.0](v2.10.0.md): Replace the init bootstrap with explicit maintenance.
- [v2.6.0](v2.6.0.md): Handle unhealthy readiness checks as HTTP errors.
