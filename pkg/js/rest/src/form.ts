import { MAX_EMAIL_LENGTH, MAX_PASSWORD_LENGTH, MAX_SHORT_CODE_LENGTH, MIN_PASSWORD_LENGTH } from "./const";

import { z } from "zod";

/** An email address within the length the service accepts, to check a form before sending it. */
export const EmailSchema = z.email().max(MAX_EMAIL_LENGTH);

/** A password within the lengths the service accepts, to check a form before sending it. */
export const PasswordSchema = z.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH);

/** A short code within the length the service accepts, to check a form before sending it. */
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

/** Validates a {@link Role}. */
export const RoleSchema = z.enum([Role.Anon, Role.User, Role.Admin, Role.SuperAdmin]);

/** Language used to localize the emails the service sends, such as short-code messages. */
export enum Lang {
  /** French. */
  Fr = "fr",
  /** English. */
  En = "en",
}

/** Validates a {@link Lang}. */
export const LangSchema = z.enum([Lang.Fr, Lang.En]);
