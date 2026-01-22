import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  AuthenticationApi,
  Lang,
  Role,
  type Token,
  claimsGet,
  credentialsExists,
  credentialsGet,
  credentialsList,
  credentialsResetPassword,
  credentialsUpdateEmail,
  credentialsUpdatePassword,
  credentialsUpdateRole,
  shortCodeCreateEmailUpdate,
  shortCodeCreatePasswordReset,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";
import {
  checkEmail,
  generateRandomMail,
  generateRandomPassword,
  getHtmlMail,
  preRegisterUser,
  registerUser,
} from "@a-novel/service-authentication-rest-test";

describe("credentialsCreate", () => {
  it("registers the user", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    await registerUser(api, preRegister);
  });

  it("does not register with wrong link", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    await expectStatus(registerUser(api, { ...preRegister, shortCode: "invalid" }), 403);
  });

  it("only registers once", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    await registerUser(api, preRegister);
    await expectStatus(registerUser(api, preRegister), 403);
  });

  it("only takes into account the latest attempt", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const email = generateRandomMail();

    const preRegister1 = await preRegisterUser(api, process.env.MAIL_TEST_HOST!, email);
    const preRegister2 = await preRegisterUser(api, process.env.MAIL_TEST_HOST!, email);
    await expectStatus(registerUser(api, preRegister1), 403);
    await registerUser(api, preRegister2);
  });
});

async function requestEmailUpdate(api: AuthenticationApi, token: Token, newEmail: string = generateRandomMail()) {
  await shortCodeCreateEmailUpdate(api, token.accessToken, {
    email: newEmail,
    lang: Lang.En,
  });

  const mailData = await checkEmail(process.env.MAIL_TEST_HOST!, `to:"${newEmail}" subject:"Email Update Request."`);

  expect(mailData.html).toBeTruthy();
  const links = getHtmlMail(mailData.html as string, "a");
  expect(links).toHaveLength(1);

  const updateUrl = (links[0] as HTMLAnchorElement).href;

  const parsedUrl = new URL(updateUrl);
  const shortCode = parsedUrl.searchParams.get("shortCode");
  const target = parsedUrl.searchParams.get("target");

  expect(shortCode).toBeTruthy();
  expect(target).toBeTruthy();

  return {
    shortCode: shortCode!,
    target: target!,
    newEmail,
  };
}

describe("credentialsUpdateEmail", () => {
  it("changes email address", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const anonToken = await tokenCreateAnon(api);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const { shortCode, target, newEmail } = await requestEmailUpdate(api, userToken);

    const newCredentials = await credentialsUpdateEmail(api, anonToken.accessToken, {
      userID: target,
      shortCode,
    });

    expect(newCredentials.email).toBe(newEmail);

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

  it("only takes into account the latest attempt", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const anonToken = await tokenCreateAnon(api);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const updateRequest1 = await requestEmailUpdate(api, userToken);
    const updateRequest2 = await requestEmailUpdate(api, userToken);

    await expectStatus(
      credentialsUpdateEmail(api, anonToken.accessToken, {
        userID: updateRequest1.target,
        shortCode: updateRequest1.shortCode,
      }),
      403
    );

    const newCredentials = await credentialsUpdateEmail(api, anonToken.accessToken, {
      userID: updateRequest2.target,
      shortCode: updateRequest2.shortCode,
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
    const api = new AuthenticationApi(process.env.API_URL!);

    const anonToken = await tokenCreateAnon(api);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const { shortCode, target, newEmail } = await requestEmailUpdate(api, userToken);

    // Register a new email before updating it.
    const preRegister2 = await preRegisterUser(api, process.env.MAIL_TEST_HOST!, newEmail);
    await registerUser(api, preRegister2);

    await expectStatus(
      credentialsUpdateEmail(api, anonToken.accessToken, {
        userID: target,
        shortCode,
      }),
      409
    );
  });
});

describe("credentialsUpdatePassword", () => {
  it("changes the user password", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const userToken = await tokenCreate(api, {
      email: user.email,
      password: user.password,
    });

    const newPassword = generateRandomPassword();

    await credentialsUpdatePassword(api, userToken.accessToken, {
      password: newPassword,
      currentPassword: user.password,
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

  it("refuses to update if current password is incorrect", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

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

  const mailData = await checkEmail(process.env.MAIL_TEST_HOST!, `to:"${email}" subject:"Password Reset Request."`);

  expect(mailData.html).toBeTruthy();
  const links = getHtmlMail(mailData.html as string, "a");
  expect(links).toHaveLength(1);

  const updateUrl = (links[0] as HTMLAnchorElement).href;

  const parsedUrl = new URL(updateUrl);
  const shortCode = parsedUrl.searchParams.get("shortCode");
  const target = parsedUrl.searchParams.get("target");

  expect(shortCode).toBeTruthy();
  expect(target).toBeTruthy();

  return {
    shortCode: shortCode!,
    target: target!,
  };
}

describe("credentialsResetPassword", () => {
  it("changes password after reset", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const anonToken = await tokenCreateAnon(api);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

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
    const api = new AuthenticationApi(process.env.API_URL!);

    const anonToken = await tokenCreateAnon(api);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

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
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    expect(user.claims.roles).toStrictEqual([Role.User]);

    await credentialsUpdateRole(api, superAdminToken.accessToken, {
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
    const api = new AuthenticationApi(process.env.API_URL!);

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

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
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const credentials = await credentialsGet(api, superAdminToken.accessToken, {
      id: user.claims.userID!,
    });

    expect(credentials.email).toBe(user.email);
    expect(credentials.role).toBe(Role.User);
    expect(credentials.id).toBe(user.claims.userID);
  });

  it("does not get non-existing credentials", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    await expectStatus(
      credentialsGet(api, superAdminToken.accessToken, {
        id: crypto.randomUUID(),
      }),
      404
    );
  });
});

describe("credentialsExists", () => {
  it("gets existing credentials", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const exists = await credentialsExists(api, superAdminToken.accessToken, {
      email: user.email,
    });

    expect(exists).toBeTruthy();
  });

  it("does not get non-existing credentials", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const exists = await credentialsExists(api, superAdminToken.accessToken, {
      email: "fake@provider.com",
    });

    expect(exists).toBeFalsy();
  });
});

describe("credentialsList", () => {
  it("returns users", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    const superAdminToken = await tokenCreate(api, {
      email: process.env.SUPER_ADMIN_EMAIL!,
      password: process.env.SUPER_ADMIN_PASSWORD!,
    });

    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
    const user = await registerUser(api, preRegister);

    const credentials = await credentialsList(api, superAdminToken.accessToken, {});

    expect(credentials.length).toBeGreaterThan(0);
    expect(credentials.some((item) => item.id === user.claims.userID)).toBeTruthy();
  });
});
