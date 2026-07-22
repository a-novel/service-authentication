// Command rest runs the authentication service's HTTP API. It wires the data-access, core,
// and handler layers together, mounts them on a chi router, and serves until an interrupt or
// termination signal arrives, then shuts the server down gracefully.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/pkg/go"
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

	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	jsonKeysCredentials := lo.Must(cfg.DependenciesConfig.ServiceJsonKeysCredentials.Options(ctx))

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(
		fmt.Sprintf("%s:%d", cfg.DependenciesConfig.ServiceJsonKeysHost, cfg.DependenciesConfig.ServiceJsonKeysPort),
		jsonKeysCredentials...,
	))

	serviceVerifyAccessToken := lo.Must(servicejsonkeys.NewClaimsVerifier[core.AccessTokenClaims](jsonKeysClient))
	serviceVerifyRefreshToken := lo.Must(servicejsonkeys.NewClaimsVerifier[core.RefreshTokenClaims](jsonKeysClient))

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoShortCodeDelete := dao.NewShortCodeDelete()
	daoShortCodeInsert := dao.NewShortCodeInsert()
	daoShortCodeSelect := dao.NewShortCodeSelect()

	daoCredentialsExist := dao.NewCredentialsExist()
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

	serviceShortCodeConsume := core.NewShortCodeConsume(daoShortCodeSelect, daoShortCodeDelete)
	serviceShortCodeCreate := core.NewShortCodeCreate(daoShortCodeInsert, cfg.ShortCodesConfig)
	serviceShortCodeCreateEmailUpdate := core.NewShortCodeCreateEmailUpdate(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreatePasswordReset := core.NewShortCodeCreatePasswordReset(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreateRegister := core.NewShortCodeCreateRegister(
		serviceShortCodeCreate,
		daoCredentialsSelectByEmail,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)

	serviceCredentialsCreate := core.NewCredentialsCreate(
		daoCredentialsInsert, serviceShortCodeConsume, jsonKeysClient, daoTransactor,
	)
	serviceCredentialsExist := core.NewCredentialsExist(daoCredentialsExist)
	serviceCredentialsGet := core.NewCredentialsGet(daoCredentialsSelect)
	serviceCredentialsList := core.NewCredentialsList(daoCredentialsList)
	serviceCredentialsUpdateEmail := core.NewCredentialsUpdateEmail(
		daoCredentialsUpdateEmail, serviceShortCodeConsume, daoTransactor,
	)
	serviceCredentialsUpdatePassword := core.NewCredentialsUpdatePassword(
		daoCredentialsUpdatePassword, daoCredentialsSelect, serviceShortCodeConsume, daoTransactor,
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

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewRestHealth(jsonKeysClient, cfg.Smtp)

	handlerClaimsGet := handlers.NewClaimsGet(cfg.Logger)

	handlerCredentialsCreate := handlers.NewCredentialsCreate(serviceCredentialsCreate, cfg.Logger)
	handlerCredentialsExist := handlers.NewCredentialsExist(serviceCredentialsExist, cfg.Logger)
	handlerCredentialsGet := handlers.NewCredentialsGet(serviceCredentialsGet, cfg.Logger)
	handlerCredentialsList := handlers.NewCredentialsList(serviceCredentialsList, cfg.Logger)
	handlerCredentialsResetPassword := handlers.NewCredentialsResetPassword(
		serviceCredentialsUpdatePassword,
		cfg.Logger,
	)
	handlerCredentialsUpdatePassword := handlers.NewCredentialsUpdatePassword(
		serviceCredentialsUpdatePassword,
		cfg.Logger,
	)
	handlerCredentialsUpdateEmail := handlers.NewCredentialsUpdateEmail(
		serviceCredentialsUpdateEmail,
		cfg.Logger,
	)
	handlerCredentialsUpdateRole := handlers.NewCredentialsUpdateRole(
		serviceCredentialsUpdateRole,
		cfg.Logger,
	)

	handlerShortCodeCreateEmailUpdate := handlers.NewShortCodeCreateEmailUpdate(
		serviceShortCodeCreateEmailUpdate,
		cfg.Logger,
	)
	handlerShortCodeCreatePasswordReset := handlers.NewShortCodeCreatePasswordReset(
		serviceShortCodeCreatePasswordReset,
		cfg.Logger,
	)
	handlerShortCodeCreateRegister := handlers.NewShortCodeCreateRegister(
		serviceShortCodeCreateRegister,
		cfg.Logger,
	)

	handlerTokenCreate := handlers.NewTokenCreate(serviceTokenCreate, cfg.Logger)
	handlerTokenCreateAnon := handlers.NewTokenCreateAnon(serviceTokenCreateAnon, cfg.Logger)
	handlerTokenRefresh := handlers.NewTokenRefresh(serviceTokenRefresh, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Timeout(cfg.Rest.Timeouts.Request))
	router.Use(middleware.RequestSize(cfg.Rest.MaxRequestSize))
	router.Use(cfg.Otel.HttpHandler())
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Rest.Cors.AllowedOrigins,
		AllowedHeaders:   cfg.Rest.Cors.AllowedHeaders,
		AllowCredentials: cfg.Rest.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		MaxAge: cfg.Rest.Cors.MaxAge,
	}))
	router.Use(cfg.HttpLogger.Logger())

	router.Get("/ping", handlerPing.ServeHTTP)
	router.Get("/healthcheck", handlerHealth.ServeHTTP)

	router.Route("/session", func(r chi.Router) {
		r.Put("/", handlerTokenCreate.ServeHTTP)
		r.Put("/anon", handlerTokenCreateAnon.ServeHTTP)

		withAuth(r).Get("/", handlerClaimsGet.ServeHTTP)
		r.Patch("/", handlerTokenRefresh.ServeHTTP)
	})

	router.Route("/credentials", func(r chi.Router) {
		withAuth(r, "credentials:get").Get("/", handlerCredentialsGet.ServeHTTP)
		withAuth(r, "credentials:exist").Head("/", handlerCredentialsExist.ServeHTTP)
		withAuth(r, "credentials:list").Get("/all", handlerCredentialsList.ServeHTTP)

		withAuth(r, "credentials:create").Put("/", handlerCredentialsCreate.ServeHTTP)
		withAuth(r, "credentials:email:patch").
			Patch("/email", handlerCredentialsUpdateEmail.ServeHTTP)
		withAuth(r, "credentials:password:patch").
			Patch("/password", handlerCredentialsUpdatePassword.ServeHTTP)
		withAuth(r, "credentials:password:reset").
			Put("/password", handlerCredentialsResetPassword.ServeHTTP)
		withAuth(r, "credentials:role:patch").
			Patch("/role", handlerCredentialsUpdateRole.ServeHTTP)
	})

	router.Route("/short-code", func(r chi.Router) {
		withAuth(r, "shortCode:register").Put("/register", handlerShortCodeCreateRegister.ServeHTTP)
		withAuth(r, "shortCode:email:update").Put("/update-email", handlerShortCodeCreateEmailUpdate.ServeHTTP)
		withAuth(r, "shortCode:password:reset").Put("/update-password", handlerShortCodeCreatePasswordReset.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Rest.Port),
		Handler:           router,
		ReadTimeout:       cfg.Rest.Timeouts.Read,
		ReadHeaderTimeout: cfg.Rest.Timeouts.ReadHeader,
		WriteTimeout:      cfg.Rest.Timeouts.Write,
		IdleTimeout:       cfg.Rest.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	serve(
		httpServer,
		cfg.Rest.Timeouts.Shutdown,
		serviceShortCodeCreateRegister,
		serviceShortCodeCreateEmailUpdate,
		serviceShortCodeCreatePasswordReset,
	)
}

// serve runs the server until an interrupt or termination signal arrives, then stops it and drains
// whatever detached work is still in flight.
//
// shutdownTimeout bounds the whole stop. The HTTP shutdown and the drain share it, so a deploy waits
// no longer than the operator configured.
func serve(httpServer *http.Server, shutdownTimeout time.Duration, drains ...lib.Waiter) {
	log.Println("Starting REST server on " + httpServer.Addr)

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down REST server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := httpServer.Shutdown(shutdownCtx)
	if err != nil {
		panic(err)
	}

	// Drained after the HTTP shutdown, once the server has stopped accepting: the set being waited
	// on is closed by then.
	log.Println("Draining in-flight emails...")

	err = lib.Drain(shutdownCtx, drains...)
	if err != nil {
		// Logged while the process is already stopping. Mail that missed the budget is worth
		// reporting and the shutdown still completes.
		log.Println("Some emails were still in flight at shutdown: " + err.Error())
	}
}
