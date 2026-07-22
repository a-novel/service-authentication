// Command init reconciles the super-admin account with the credentials given in
// SUPER_ADMIN_EMAIL and SUPER_ADMIN_PASSWORD. It is idempotent and safe to run on every
// deploy: a missing account is created with the super-admin role, and an existing account
// whose password or role has drifted is brought back in line.
//
// When either variable is empty the command logs a warning and exits without touching the
// database, so the bootstrap step can be disabled by leaving the variables unset.
package main

import (
	"context"
	"log"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("init: ")

	start := time.Now()

	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	if env.SuperAdminEmail == "" {
		cfg.Logger.Warn(ctx, "SUPER_ADMIN_EMAIL not set — skipping super-admin bootstrap")
		log.Printf("done — skipped (no SUPER_ADMIN_EMAIL), completed in %s",
			time.Since(start).Round(time.Millisecond))

		return
	}

	if env.SuperAdminPassword == "" {
		cfg.Logger.Warn(ctx, "SUPER_ADMIN_PASSWORD not set — skipping super-admin bootstrap")
		log.Printf("done — skipped (no SUPER_ADMIN_PASSWORD), completed in %s",
			time.Since(start).Round(time.Millisecond))

		return
	}

	log.Println("connecting to database...")

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	daoCredentialsInsert := dao.NewCredentialsInsert()
	daoCredentialsSelectByEmail := dao.NewCredentialsSelectByEmail()
	daoCredentialsUpdatePassword := dao.NewCredentialsUpdatePassword()
	daoCredentialsUpdateRole := dao.NewCredentialsUpdateRole()

	service := core.NewCredentialsCreateSuperAdmin(
		daoCredentialsInsert,
		daoCredentialsSelectByEmail,
		daoCredentialsUpdatePassword,
		daoCredentialsUpdateRole,
		postgres.NewTransactor(nil),
	)

	log.Printf("ensuring super-admin credentials for %s", env.SuperAdminEmail)
	_ = lo.Must(service.Exec(ctx, &core.CredentialsCreateSuperAdminRequest{
		Email:    env.SuperAdminEmail,
		Password: env.SuperAdminPassword,
	}))

	log.Printf("done — super-admin %s is created or up-to-date, completed in %s",
		env.SuperAdminEmail, time.Since(start).Round(time.Millisecond))
}
