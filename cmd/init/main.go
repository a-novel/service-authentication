// Command init bootstraps the super-admin credentials from environment variables. It is
// idempotent and safe to run on every deploy:
//
//   - If no account exists for SUPER_ADMIN_EMAIL, a new one is created with the super-admin
//     role and the configured password.
//   - If an account exists with a different password, the password is updated.
//   - If an account exists with a non-super-admin role, the role is upgraded.
//
// If either SUPER_ADMIN_EMAIL or SUPER_ADMIN_PASSWORD is empty, the command logs a warning
// and exits cleanly without attempting any database changes — useful for environments where
// the bootstrap step is intentionally disabled.
package main

import (
	"context"
	"log"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if env.SuperAdminEmail == "" {
		cfg.Logger.Warn(ctx, "admin email not set, aborting")

		return
	}

	if env.SuperAdminPassword == "" {
		cfg.Logger.Warn(ctx, "admin password not set, aborting")

		return
	}

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	repositoryCredentialsInsert := dao.NewCredentialsInsert()
	repositoryCredentialsSelectByEmail := dao.NewCredentialsSelectByEmail()
	repositoryCredentialsUpdatePassword := dao.NewCredentialsUpdatePassword()
	repositoryCredentialsUpdateRole := dao.NewCredentialsUpdateRole()

	service := services.NewCredentialsCreateSuperAdmin(
		repositoryCredentialsInsert,
		repositoryCredentialsSelectByEmail,
		repositoryCredentialsUpdatePassword,
		repositoryCredentialsUpdateRole,
	)

	_ = lo.Must(service.Exec(ctx, &services.CredentialsCreateSuperAdminRequest{
		Email:    env.SuperAdminEmail,
		Password: env.SuperAdminPassword,
	}))
}
