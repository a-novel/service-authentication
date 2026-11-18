# Upgrading to NEXT

The service no longer reads `SERVICE_TOKEN_PATH`: its JSON Keys token comes from `SERVICE_JSON_KEYS_TOKEN_PATH` alone, as v2.27.0 announced.

## Do I need to act?

**Required** for a deployment that still sets `SERVICE_TOKEN_PATH` on the `rest` or `standalone-rest` image: rename it before taking this release. **No** for every other consumer: the REST and gRPC contracts, the Go client and the JavaScript packages do not change.

## Set SERVICE_JSON_KEYS_TOKEN_PATH

v2.27.0 renamed the variable and still read the old name when the new one was unset. This release drops that fallback.

```diff
-SERVICE_TOKEN_PATH=/var/run/secrets/tokens/json-keys
+SERVICE_JSON_KEYS_TOKEN_PATH=/var/run/secrets/tokens/json-keys
```

A deployment that sets only `SERVICE_TOKEN_PATH` now calls JSON Keys without a service token, and a JSON Keys that verifies service tokens refuses those calls. Production already sets the new name. Check a deployment by reading its environment: `SERVICE_JSON_KEYS_TOKEN_PATH` names the projected token file.

`SERVICE_TOKEN_PATH` keeps its meaning for the callers of this service, such as the account platform, which read their own token for the authentication audience from it.
