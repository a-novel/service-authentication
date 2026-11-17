import { accountPlatform, callerApi } from "./caller";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type AuthenticationApi,
  Lang,
  Role,
  type Token,
  claimsGet,
  credentialsExists,
  credentialsGet,
  credentialsGetSelf,
  credentialsList,
  credentialsResetPassword,
  credentialsUpdateEmail,
  credentialsUpdatePassword,
  credentialsUpdateRole,
  shortCodeCancelEmailUpdate,
  shortCodeCreateEmailUpdate,
  shortCodeCreatePasswordReset,
  shortCodeCreateRegister,
  tokenCreate,
  tokenCreateAnon,
  tokenRefresh,
} from "@a-novel/service-authentication-rest";
import {
  type MailpitMessage,
  checkEmail,
  generateRandomMail,
  generateRandomPassword,
  mailLinks,
  preRegisterUser,
  registerUser,
  superAdminAccessToken,
} from "@a-novel/service-authentication-rest-test";

/**
 * Returns the short code and target of the one link an email carries, as its button and again as fallback text, after
 * checking that it opens the platform page `page`.
 */
function mailedLink(mail: MailpitMessage, page: string) {
  const links = mailLinks(mail.HTML);
  expect(links).toHaveLength(2);
  expect(links[1]).toBe(links[0]);

  const { pathname, searchParams } = new URL(links[0]);
  expect(pathname).toBe(page);

  const shortCode = searchParams.get("shortCode");
  const target = searchParams.get("target");
  expect(shortCode).toBeTruthy();
  expect(target).toBeTruthy();

  return { shortCode: shortCode!, target: target! };
}

describe("shortCodeCreateRegister", () => {
  it("mails an invitation that links to account creation for the address", async () => {
    const api = callerApi(accountPlatform);
    const email = generateRandomMail();

    await shortCodeCreateRegister(api, await superAdminAccessToken(api), { email, lang: Lang.En });

    const mail = await checkEmail(`to:"${email}" subject:"Create your Agora account"`);
    const { target } = mailedLink(mail, "/ext/account/create");
    expect(Buffer.from(target, "base64url").toString()).toBe(email);
  });

  it("refuses anonymous callers", async () => {
    const api = callerApi(accountPlatform);
    const anonymousToken = await tokenCreateAnon(api);

    await expectStatus(
      shortCodeCreateRegister(api, anonymousToken.accessToken, {
        email: generateRandomMail(),
        lang: Lang.En,
      }),
      403
    );
  });
});

describe("credentialsCreate", () => {
  it("registers the user", async () => {
    const api = callerApi(accountPlatform);

    await registerUser(api);
  });

  it("registers an address with a plus sign", async () => {
    const api = callerApi(accountPlatform);

    await registerUser(api, await preRegisterUser(api, `tag+${generateRandomMail()}`));
  });

  it("does not register with wrong link", async () => {
    const api = callerApi(accountPlatform);

    const preRegister = await preRegisterUser(api);
    await expectStatus(registerUser(api, { ...preRegister, shortCode: "invalid" }), 403);
  });

  it("only registers once", async () => {
    const api = callerApi(accountPlatform);

    const preRegister = await preRegisterUser(api);
    await registerUser(api, preRegister);
    await expectStatus(registerUser(api, preRegister), 403);
  });

  it("only takes into account the latest attempt", async () => {
    const api = callerApi(accountPlatform);

    const email = generateRandomMail();

    const preRegister1 = await preRegisterUser(api, email);
    const preRegister2 = await preRegisterUser(api, email);
    await expectStatus(registerUser(api, preRegister1), 403);
    await registerUser(api, preRegister2);
  });
});

async function requestEmailUpdate(
  api: AuthenticationApi,
  token: Token,
  currentPassword: string,
  newEmail: string = generateRandomMail()
) {
  await shortCodeCreateEmailUpdate(api, token.accessToken, {
    currentPassword,
    email: newEmail,
    lang: Lang.En,
  });

  const mail = await checkEmail(`to:"${newEmail}" subject:"Confirm your new Agora email address"`);

  return { ...mailedLink(mail, "/ext/email/validate"), newEmail };
}

describe("credentialsUpdateEmail", () => {
  it("changes email address", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const { shortCode, target, newEmail } = await requestEmailUpdate(api, userToken, user.password);

    const newCredentials = await credentialsUpdateEmail(api, anonToken.accessToken, {
      userID: target,
      shortCode,
      lang: Lang.En,
    });

    expect(newCredentials.email).toBe(newEmail);

    // The former address learns of the change, and the sessions opened before it continue.
    await checkEmail(`to:"${user.email}" subject:"Your Agora email address changed"`);
    await tokenRefresh(api, userToken);

    await expectStatus(
      tokenCreate(api, {
        email: user.email,
        password: user.password,
      }),
      401
    );

    await tokenCreate(api, {
      email: newEmail,
      password: user.password,
    });
  });

  it("refuses to start a change without the current password", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    await expectStatus(
      shortCodeCreateEmailUpdate(api, user.token.accessToken, {
        currentPassword: "incorrect",
        email: generateRandomMail(),
        lang: Lang.En,
      }),
      403
    );

    const self = await credentialsGetSelf(api, user.token.accessToken);
    expect(self.pendingEmail).toBeUndefined();
  });

  it("only takes into account the latest attempt", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const updateRequest1 = await requestEmailUpdate(api, userToken, user.password);
    const updateRequest2 = await requestEmailUpdate(api, userToken, user.password);

    await expectStatus(
      credentialsUpdateEmail(api, anonToken.accessToken, {
        userID: updateRequest1.target,
        shortCode: updateRequest1.shortCode,
        lang: Lang.En,
      }),
      403
    );

    const newCredentials = await credentialsUpdateEmail(api, anonToken.accessToken, {
      userID: updateRequest2.target,
      shortCode: updateRequest2.shortCode,
      lang: Lang.En,
    });

    expect(newCredentials.email).toBe(updateRequest2.newEmail);

    await expectStatus(
      tokenCreate(api, {
        email: user.email,
        password: user.password,
      }),
      401
    );

    await tokenCreate(api, {
      email: updateRequest2.newEmail,
      password: user.password,
    });
  });

  it("accepts update if email has been taken", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const { shortCode, target, newEmail } = await requestEmailUpdate(api, userToken, user.password);

    // Register a new email before updating it.
    await registerUser(api, await preRegisterUser(api, newEmail));

    await expectStatus(
      credentialsUpdateEmail(api, anonToken.accessToken, {
        userID: target,
        shortCode,
        lang: Lang.En,
      }),
      409
    );
  });
});

describe("credentialsUpdatePassword", () => {
  it("changes the user password", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const newPassword = generateRandomPassword();

    await credentialsUpdatePassword(api, userToken.accessToken, {
      password: newPassword,
      currentPassword: user.password,
    });

    // The sessions opened before the change continue.
    await tokenRefresh(api, userToken);

    await expectStatus(
      tokenCreate(api, {
        email: user.email,
        password: user.password,
      }),
      401
    );

    await tokenCreate(api, {
      email: user.email,
      password: newPassword,
    });
  });

  it("refuses to update if current password is incorrect", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const newPassword = generateRandomPassword();

    await expectStatus(
      credentialsUpdatePassword(api, userToken.accessToken, {
        password: newPassword,
        currentPassword: "incorrect",
      }),
      403
    );

    await expectStatus(
      tokenCreate(api, {
        email: user.email,
        password: newPassword,
      }),
      401
    );

    await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });
  });
});

async function requestPasswordReset(api: AuthenticationApi, token: Token, email: string) {
  await shortCodeCreatePasswordReset(api, token.accessToken, {
    email: email,
    lang: Lang.En,
  });

  const mail = await checkEmail(`to:"${email}" subject:"Reset your Agora password"`);

  return mailedLink(mail, "/ext/password/reset");
}

describe("credentialsResetPassword", () => {
  it("changes password after reset", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const { shortCode, target } = await requestPasswordReset(api, anonToken, user.email);

    const newPassword = generateRandomPassword();

    await credentialsResetPassword(api, anonToken.accessToken, {
      userID: target,
      password: newPassword,
      shortCode,
    });

    await expectStatus(
      tokenCreate(api, {
        email: user.email,
        password: user.password,
      }),
      401
    );

    await tokenCreate(api, {
      email: user.email,
      password: newPassword,
    });
  });

  it("only takes into account the latest attempt", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const requestPasswordReset1 = await requestPasswordReset(api, anonToken, user.email);
    const requestPasswordReset2 = await requestPasswordReset(api, anonToken, user.email);

    const newPassword = generateRandomPassword();

    await expectStatus(
      credentialsResetPassword(api, anonToken.accessToken, {
        userID: requestPasswordReset1.target,
        password: newPassword,
        shortCode: requestPasswordReset1.shortCode,
      }),
      403
    );

    await credentialsResetPassword(api, anonToken.accessToken, {
      userID: requestPasswordReset2.target,
      password: newPassword,
      shortCode: requestPasswordReset2.shortCode,
    });
  });
});

describe("credentialsUpdateRole", () => {
  it("changes the role of a user", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    const user = await registerUser(api);

    expect(user.claims.roles).toStrictEqual([Role.User]);

    await credentialsUpdateRole(api, superAdminToken, {
      userID: user.claims.userID!,
      role: Role.Admin,
    });

    // Relogging is necessary for new role to be effective.
    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const newUserClaims = await claimsGet(api, userToken.accessToken);

    expect(newUserClaims.roles).toStrictEqual([Role.Admin]);
  });

  it("refuses user to change roles themselves", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    await expectStatus(
      credentialsUpdateRole(api, userToken.accessToken, {
        userID: user.claims.userID!,
        role: Role.Admin,
      }),
      403
    );
  });
});

describe("credentialsGet", () => {
  it("gets existing credentials", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    const user = await registerUser(api);

    const credentials = await credentialsGet(api, superAdminToken, {
      id: user.claims.userID!,
    });

    expect(credentials.email).toBe(user.email);
    expect(credentials.role).toBe(Role.User);
    expect(credentials.id).toBe(user.claims.userID);
  });

  it("does not get non-existing credentials", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    await expectStatus(
      credentialsGet(api, superAdminToken, {
        id: crypto.randomUUID(),
      }),
      404
    );
  });
});

describe("credentialsGetSelf", () => {
  it("gets the caller's credentials", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const credentials = await credentialsGetSelf(api, user.token.accessToken);

    expect(credentials.id).toBe(user.claims.userID);
    expect(credentials.email).toBe(user.email);
    expect(credentials.role).toBe(Role.User);
    expect(credentials.pendingEmail).toBeUndefined();
  });

  it("returns the pending email until it is confirmed", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const { shortCode, target, newEmail } = await requestEmailUpdate(api, user.token, user.password);

    const pending = await credentialsGetSelf(api, user.token.accessToken);
    expect(pending.email).toBe(user.email);
    expect(pending.pendingEmail?.email).toBe(newEmail);
    expect(pending.pendingEmail?.expiresAt.getTime()).toBeGreaterThan(Date.now());

    await credentialsUpdateEmail(api, anonToken.accessToken, { userID: target, shortCode, lang: Lang.En });

    const confirmed = await credentialsGetSelf(api, user.token.accessToken);
    expect(confirmed.email).toBe(newEmail);
    expect(confirmed.pendingEmail).toBeUndefined();
  });

  it("does not reveal that the pending email is registered", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const owner = await registerUser(api);

    await shortCodeCreateEmailUpdate(api, user.token.accessToken, {
      currentPassword: user.password,
      email: owner.email,
      lang: Lang.En,
    });

    const pending = await credentialsGetSelf(api, user.token.accessToken);
    expect(pending.pendingEmail?.email).toBe(owner.email);
    expect(pending.pendingEmail?.expiresAt.getTime()).toBeGreaterThan(Date.now());
  });

  it("refuses anonymous callers", async () => {
    const api = callerApi(accountPlatform);
    const anonToken = await tokenCreateAnon(api);

    await expectStatus(credentialsGetSelf(api, anonToken.accessToken), 403);
  });
});

describe("shortCodeCancelEmailUpdate", () => {
  it("cancels the pending email change", async () => {
    const api = callerApi(accountPlatform);

    const anonToken = await tokenCreateAnon(api);

    const user = await registerUser(api);

    const { shortCode, target } = await requestEmailUpdate(api, user.token, user.password);

    await shortCodeCancelEmailUpdate(api, user.token.accessToken);

    const self = await credentialsGetSelf(api, user.token.accessToken);
    expect(self.email).toBe(user.email);
    expect(self.pendingEmail).toBeUndefined();

    // The emailed code no longer confirms the change.
    await expectStatus(
      credentialsUpdateEmail(api, anonToken.accessToken, { userID: target, shortCode, lang: Lang.En }),
      403
    );

    await expectStatus(shortCodeCancelEmailUpdate(api, user.token.accessToken), 404);
  });

  it("cancels a change to a registered address alike", async () => {
    const api = callerApi(accountPlatform);

    const user = await registerUser(api);

    const owner = await registerUser(api);

    await shortCodeCreateEmailUpdate(api, user.token.accessToken, {
      currentPassword: user.password,
      email: owner.email,
      lang: Lang.En,
    });

    await shortCodeCancelEmailUpdate(api, user.token.accessToken);

    const self = await credentialsGetSelf(api, user.token.accessToken);
    expect(self.pendingEmail).toBeUndefined();
  });

  it("refuses anonymous callers", async () => {
    const api = callerApi(accountPlatform);
    const anonToken = await tokenCreateAnon(api);

    await expectStatus(shortCodeCancelEmailUpdate(api, anonToken.accessToken), 403);
  });
});

describe("credentialsExists", () => {
  it("gets existing credentials", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    const user = await registerUser(api);

    const exists = await credentialsExists(api, superAdminToken, {
      email: user.email,
    });

    expect(exists).toBeTruthy();
  });

  it("does not get non-existing credentials", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    const exists = await credentialsExists(api, superAdminToken, {
      email: "fake@provider.com",
    });

    expect(exists).toBeFalsy();
  });
});

describe("credentialsList", () => {
  it("returns users", async () => {
    const api = callerApi(accountPlatform);

    const superAdminToken = await superAdminAccessToken(api);

    const user = await registerUser(api);

    const credentials = await credentialsList(api, superAdminToken, {});

    expect(credentials.length).toBeGreaterThan(0);
    expect(credentials.some((item) => item.id === user.claims.userID)).toBeTruthy();
  });
});
