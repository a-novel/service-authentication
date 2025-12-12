import { MAX_EMAIL_LENGTH, MAX_PASSWORD_LENGTH, MAX_SHORT_CODE_LENGTH } from "./const";

import { z } from "zod";

export const EmailSchema = z.email().max(MAX_EMAIL_LENGTH);

export const PasswordSchema = z.string().max(MAX_PASSWORD_LENGTH);

export const ShortCodeSchema = z.string().max(MAX_SHORT_CODE_LENGTH);

export enum Role {
  Anon = "auth:anon",
  User = "auth:user",
  Admin = "auth:admin",
  SuperAdmin = "auth:superadmin",
}

export const RoleSchema = z.enum([Role.Anon, Role.User, Role.Admin, Role.SuperAdmin]);

export enum Lang {
  Fr = "fr",
  En = "en",
}

export const LangSchema = z.enum([Lang.Fr, Lang.En]);
