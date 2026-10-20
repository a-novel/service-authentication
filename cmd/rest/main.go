// Command rest runs the authentication service's HTTP API. It wires the data-access, core,
// and handler layers together, mounts them on a chi router, and serves until an interrupt or
// termination signal arrives, then shuts the server down gracefully.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/pkg/go"
)

func main() {
	cfg := config.AppPresetDefault

	processCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := processCtx

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

		database := lo.Must(cfg.Postgres.DB(ctx))
		defer closeDatabase(database)
	}

	jsonKeysCredentials := lo.Must(cfg.DependenciesConfig.ServiceJsonKeysCredentials.Options(ctx))

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(
		fmt.Sprintf("%s:%d", cfg.DependenciesConfig.ServiceJsonKeysHost, cfg.DependenciesConfig.ServiceJsonKeysPort),
		jsonKeysCredentials...,
	))
	defer jsonKeysClient.Close()

	serviceVerifyAccessToken := lo.Must(servicejsonkeys.NewClaimsVerifier[core.AccessTokenClaims](jsonKeysClient))
	serviceVerifyRefreshToken := lo.Must(servicejsonkeys.NewClaimsVerifier[core.RefreshTokenClaims](jsonKeysClient))

	mailDelivery := core.NewMailDelivery(cfg.Smtp, env.SmtpMaxConcurrent)
	lib.LimitArgon2(cfg.Limits.Argon2MaxConcurrent)

	// Missing SMTP_ADDR looks like healthy mail loss: 202 answered, mail rendered to stdout.
	if _, debug := cfg.Smtp.(*smtp.DebugSender); debug {
		log.Println("WARNING: SMTP_ADDR is not set; emails are printed to stdout by the debug sender and none are delivered")
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoShortCodeDelete := dao.NewShortCodeDelete()
	daoShortCodeInsert := dao.NewShortCodeInsert()
	daoShortCodeSelect := dao.NewShortCodeSelect()

	daoCredentialsExist := dao.NewCredentialsExist()
	daoWaitlist := lo.Must(dao.NewGoogleWaitlist(cfg.Waitlist, nil))
	daoTransactor := postgres.NewTransactor(nil)

	daoCredentialsInsert := dao.NewCredentialsInsert()
	daoCredentialsList := dao.NewCredentialsList()
	daoCredentialsSelect := dao.NewCredentialsSelect()
	daoCredentialsSelectByEmail := dao.NewCredentialsSelectByEmail()
	daoCredentialsUpdateEmail := dao.NewCredentialsUpdateEmail()
	daoCredentialsUpdatePassword := dao.NewCredentialsUpdatePassword()
	daoCredentialsUpdateRole := dao.NewCredentialsUpdateRole()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceShortCodeVerify := core.NewShortCodeVerify(daoShortCodeSelect)
	serviceShortCodeConsume := core.NewShortCodeConsume(daoShortCodeDelete)
	serviceShortCodeCreate := core.NewShortCodeCreate(daoShortCodeInsert, cfg.ShortCodesConfig)
	serviceShortCodeCreateEmailUpdate := core.NewShortCodeCreateEmailUpdate(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		mailDelivery,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreatePasswordReset := core.NewShortCodeCreatePasswordReset(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		mailDelivery,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreateRegister := core.NewShortCodeCreateRegister(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		mailDelivery,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)

	serviceCredentialsCreate := core.NewCredentialsCreate(
		daoCredentialsInsert, serviceShortCodeVerify, serviceShortCodeConsume, jsonKeysClient, daoTransactor, daoWaitlist,
	)
	serviceCredentialsExist := core.NewCredentialsExist(daoCredentialsExist)
	serviceWaitlistJoin := core.NewWaitlistJoin(daoCredentialsExist, daoWaitlist)
	serviceCredentialsGet := core.NewCredentialsGet(daoCredentialsSelect)
	serviceCredentialsList := core.NewCredentialsList(daoCredentialsList)
	serviceCredentialsUpdateEmail := core.NewCredentialsUpdateEmail(
		daoCredentialsUpdateEmail, serviceShortCodeVerify, serviceShortCodeConsume, daoTransactor,
	)
	serviceCredentialsUpdatePassword := core.NewCredentialsUpdatePassword(
		daoCredentialsUpdatePassword, daoCredentialsSelect, serviceShortCodeVerify, serviceShortCodeConsume, daoTransactor,
	)
	serviceCredentialsUpdateRole := core.NewCredentialsUpdateRole(
		daoCredentialsUpdateRole,
		daoCredentialsSelect,
	)

	serviceTokenCreate := core.NewTokenCreate(daoCredentialsSelectByEmail, jsonKeysClient)
	serviceTokenCreateAnon := core.NewTokenCreateAnon(jsonKeysClient)
	serviceTokenRefresh := core.NewTokenRefresh(
		daoCredentialsSelect,
		jsonKeysClient,
		serviceVerifyAccessToken,
		serviceVerifyRefreshToken,
	)

	// =================================================================================================================
	// MIDDLEWARES
	// =================================================================================================================

	withAuth := serviceauthentication.NewAuthHandler(serviceVerifyAccessToken, cfg.Permissions, cfg.Logger)
	limit := middlewares.NewRestLimits(cfg.Rest, cfg.Limits)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewRestPing()
	handlerHealth := handlers.NewRestHealth(jsonKeysClient, cfg.Smtp, cfg.Logger)

	handlerClaimsGet := handlers.NewRestClaimsGet(cfg.Logger)

	handlerCredentialsCreate := handlers.NewRestCredentialsCreate(serviceCredentialsCreate, cfg.Logger)
	handlerCredentialsExist := handlers.NewRestCredentialsExist(serviceCredentialsExist, cfg.Logger)
	handlerWaitlistJoin := handlers.NewRestWaitlistJoin(serviceWaitlistJoin, cfg.Logger)
	handlerCredentialsGet := handlers.NewRestCredentialsGet(serviceCredentialsGet, cfg.Logger)
	handlerCredentialsList := handlers.NewRestCredentialsList(serviceCredentialsList, cfg.Logger)
	handlerCredentialsResetPassword := handlers.NewRestCredentialsResetPassword(
		serviceCredentialsUpdatePassword,
		cfg.Logger,
	)
	handlerCredentialsUpdatePassword := handlers.NewRestCredentialsUpdatePassword(
		serviceCredentialsUpdatePassword,
		cfg.Logger,
	)
	handlerCredentialsUpdateEmail := handlers.NewRestCredentialsUpdateEmail(
		serviceCredentialsUpdateEmail,
		cfg.Logger,
	)
	handlerCredentialsUpdateRole := handlers.NewRestCredentialsUpdateRole(
		serviceCredentialsUpdateRole,
		cfg.Logger,
	)

	handlerShortCodeCreateEmailUpdate := handlers.NewRestShortCodeCreateEmailUpdate(
		serviceShortCodeCreateEmailUpdate,
		cfg.Logger,
	)
	handlerShortCodeCreatePasswordReset := handlers.NewRestShortCodeCreatePasswordReset(
		serviceShortCodeCreatePasswordReset,
		cfg.Logger,
	)
	handlerShortCodeCreateRegister := handlers.NewRestShortCodeCreateRegister(
		serviceShortCodeCreateRegister,
		cfg.Logger,
	)

	handlerTokenCreate := handlers.NewRestTokenCreate(serviceTokenCreate, cfg.Logger)
	handlerTokenCreateAnon := handlers.NewRestTokenCreateAnon(serviceTokenCreateAnon, cfg.Logger)
	handlerTokenRefresh := handlers.NewRestTokenRefresh(serviceTokenRefresh, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, cfg.Otel, cfg.HttpLogger, cfg.App.DowntimeStart)

	router.Route("/v2", func(api chi.Router) {
		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)

		withAuth(api, "waitlist:join").With(limit.Waitlist).Put("/waitlist", handlerWaitlistJoin.ServeHTTP)

		api.Route("/session", func(r chi.Router) {
			r.With(limit.Login).Put("/", handlerTokenCreate.ServeHTTP)
			r.With(limit.Anonymous).Put("/anon", handlerTokenCreateAnon.ServeHTTP)

			withAuth(r).Get("/", handlerClaimsGet.ServeHTTP)
			r.Patch("/", handlerTokenRefresh.ServeHTTP)
		})

		api.Route("/credentials", func(r chi.Router) {
			withAuth(r, "credentials:get").Get("/", handlerCredentialsGet.ServeHTTP)
			withAuth(r, "credentials:exist").Head("/", handlerCredentialsExist.ServeHTTP)
			withAuth(r, "credentials:list").Get("/all", handlerCredentialsList.ServeHTTP)

			withAuth(r, "credentials:create").With(limit.SignUp).Put("/", handlerCredentialsCreate.ServeHTTP)
			withAuth(r, "credentials:email:patch").With(limit.ShortCode).
				Patch("/email", handlerCredentialsUpdateEmail.ServeHTTP)
			withAuth(r, "credentials:password:patch").With(limit.Password).
				Patch("/password", handlerCredentialsUpdatePassword.ServeHTTP)
			withAuth(r, "credentials:password:reset").With(limit.ShortCode).
				Put("/password", handlerCredentialsResetPassword.ServeHTTP)
			withAuth(r, "credentials:role:patch").
				Patch("/role", handlerCredentialsUpdateRole.ServeHTTP)
		})

		api.Route("/short-code", func(r chi.Router) {
			withAuth(r, "shortCode:register").With(limit.Mailbox).
				Put("/register", handlerShortCodeCreateRegister.ServeHTTP)
			withAuth(r, "shortCode:email:update").With(limit.Mail).
				Put("/update-email", handlerShortCodeCreateEmailUpdate.ServeHTTP)
			withAuth(r, "shortCode:password:reset").With(limit.Mail).
				Put("/update-password", handlerShortCodeCreatePasswordReset.ServeHTTP)
		})
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	httpServer := httpf.NewServer(ctx, cfg.Rest, router)

	lo.Must0(serve(
		processCtx,
		stop,
		httpServer,
		cfg.Rest.Timeouts.Shutdown,
		mailDelivery,
	))
}

// serve runs the shared HTTP lifecycle, then drains accepted mail inside the same shutdown budget.
func serve(
	ctx context.Context,
	stop context.CancelFunc,
	httpServer *http.Server,
	shutdownTimeout time.Duration,
	mailDelivery *core.MailDelivery,
) error {
	shutdownStarted := make(chan time.Time, 1)

	go func() {
		<-ctx.Done()
		mailDelivery.Close()

		shutdownStarted <- time.Now()
	}()

	log.Println("Starting REST server on " + httpServer.Addr)

	serveErr := httpf.Serve(ctx, httpServer, shutdownTimeout)

	// Stop also cancels the process context when serving ends because of a bind or runtime error.
	// The recorded deadline lets the HTTP drain and accepted mail share one operator budget.
	stop()

	shutdownCtx, cancel := context.WithDeadline(
		context.WithoutCancel(ctx),
		(<-shutdownStarted).Add(shutdownTimeout),
	)

	log.Println("Draining in-flight emails...")

	drainErr := mailDelivery.Wait(shutdownCtx)

	cancel()

	if drainErr != nil {
		// Logged while the process is already stopping. Failed deliveries or mail that missed
		// the budget are worth reporting and the shutdown still completes.
		log.Println("Some emails failed or were still in flight at shutdown: " + drainErr.Error())
	}

	return serveErr
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
