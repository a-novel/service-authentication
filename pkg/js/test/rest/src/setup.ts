import { accountPlatform, callerApi } from "./caller";

import { registerSuperAdmin } from "@a-novel/service-authentication-rest-test";

/** Completes the super-admin invitation the maintenance command mails when the stack starts. */
export default async function setup() {
  await registerSuperAdmin(callerApi(accountPlatform));
}
