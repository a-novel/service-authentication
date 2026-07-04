import { MAX_EMAIL_LENGTH, MAX_PASSWORD_LENGTH, MAX_SHORT_CODE_LENGTH, MIN_PASSWORD_LENGTH } from "./const";

import { z } from "zod";

// Field validators shared across the request schemas in this package, so a form can be checked
// against the same bounds the service enforces before a request is sent.

export const EmailSchema = z.email().max(MAX_EMAIL_LENGTH);

export const PasswordSchema = z.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH);

export const ShortCodeSchema = z.string().max(MAX_SHORT_CODE_LENGTH);

/** Authorization level granted to a credential and carried by a session's claims. */
export enum Role {
  /** Unauthenticated session, issued without credentials. */
  Anon = "auth:anon",
  /** Standard authenticated user. */
  User = "auth:user",
  /** Elevated operator, able to manage other users' credentials. */
  Admin = "auth:admin",
  /** Highest privilege, able to manage admins. */
  SuperAdmin = "auth:superadmin",
}

export const RoleSchema = z.enum([Role.Anon, Role.User, Role.Admin, Role.SuperAdmin]);

/** Language used to localize the emails the service sends, such as short-code messages. */
export enum Lang {
  /** French. */
  Fr = "fr",
  /** English. */
  En = "en",
}

export const LangSchema = z.enum([Lang.Fr, Lang.En]);
