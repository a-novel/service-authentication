import { z } from "zod";

export const PasswordSchema = z.string().max(1024);

export const ShortCodeSchema = z.string().max(1024);

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
