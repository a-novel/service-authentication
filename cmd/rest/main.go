package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/samber/lo"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/services"
	"github.com/a-novel/service-authentication/v2/pkg"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	jsonKeysCredentials := lo.Must(cfg.DependenciesConfig.ServiceJsonKeysCredentials.Options(ctx))

	jsonKeysClient := lo.Must(jkpkg.NewClient(
		fmt.Sprintf("%s:%d", cfg.DependenciesConfig.ServiceJsonKeysHost, cfg.DependenciesConfig.ServiceJsonKeysPort),
		jsonKeysCredentials...,
	))

	serviceVerifyAccessToken := jkpkg.NewClaimsVerifier[services.AccessTokenClaims](jsonKeysClient)
	serviceVerifyRefreshToken := jkpkg.NewClaimsVerifier[services.RefreshTokenClaims](jsonKeysClient)

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	repositoryShortCodeDelete := dao.NewShortCodeDelete()
	repositoryShortCodeInsert := dao.NewShortCodeInsert()
	repositoryShortCodeSelect := dao.NewShortCodeSelect()

	repositoryCredentialsExist := dao.NewCredentialsExist()
	repositoryCredentialsInsert := dao.NewCredentialsInsert()
	repositoryCredentialsList := dao.NewCredentialsList()
	repositoryCredentialsSelect := dao.NewCredentialsSelect()
	repositoryCredentialsSelectByEmail := dao.NewCredentialsSelectByEmail()
	repositoryCredentialsUpdateEmail := dao.NewCredentialsUpdateEmail()
	repositoryCredentialsUpdatePassword := dao.NewCredentialsUpdatePassword()
	repositoryCredentialsUpdateRole := dao.NewCredentialsUpdateRole()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceShortCodeConsume := services.NewShortCodeConsume(repositoryShortCodeSelect, repositoryShortCodeDelete)
	serviceShortCodeCreate := services.NewShortCodeCreate(repositoryShortCodeInsert, cfg.ShortCodesConfig)
	serviceShortCodeCreateEmailUpdate := services.NewShortCodeCreateEmailUpdate(
		serviceShortCodeCreate,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreatePasswordReset := services.NewShortCodeCreatePasswordReset(
		serviceShortCodeCreate,
		repositoryCredentialsSelectByEmail,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	serviceShortCodeCreateRegister := services.NewShortCodeCreateRegister(
		serviceShortCodeCreate,
		cfg.Smtp,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)

	serviceCredentialsCreate := services.NewCredentialsCreate(
		repositoryCredentialsInsert,
		serviceShortCodeConsume,
		jsonKeysClient,
	)
	serviceCredentialsExist := services.NewCredentialsExist(repositoryCredentialsExist)
	serviceCredentialsGet := services.NewCredentialsGet(repositoryCredentialsSelect)
	serviceCredentialsList := services.NewCredentialsList(repositoryCredentialsList)
	serviceCredentialsUpdateEmail := services.NewCredentialsUpdateEmail(
		repositoryCredentialsUpdateEmail,
		serviceShortCodeConsume,
	)
	serviceCredentialsUpdatePassword := services.NewCredentialsUpdatePassword(
		repositoryCredentialsUpdatePassword,
		repositoryCredentialsSelect,
		serviceShortCodeConsume,
	)
	serviceCredentialsUpdateRole := services.NewCredentialsUpdateRole(
		repositoryCredentialsUpdateRole,
		repositoryCredentialsSelect,
	)

	serviceTokenCreate := services.NewTokenCreate(repositoryCredentialsSelectByEmail, jsonKeysClient)
	serviceTokenCreateAnon := services.NewTokenCreateAnon(jsonKeysClient)
	serviceTokenRefresh := services.NewTokenRefresh(
		repositoryCredentialsSelect,
		jsonKeysClient,
		serviceVerifyAccessToken,
		serviceVerifyRefreshToken,
	)

	// =================================================================================================================
	// MIDDLEWARES
	// =================================================================================================================

	withAuth := pkg.NewAuthHandler(serviceVerifyAccessToken, cfg.Permissions)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewHealth(jsonKeysClient, cfg.Smtp)

	handlerClaimsGet := handlers.NewClaimsGet()

	handlerCredentialsCreate := handlers.NewCredentialsCreate(serviceCredentialsCreate)
	handlerCredentialsExist := handlers.NewCredentialsExist(serviceCredentialsExist)
	handlerCredentialsGet := handlers.NewCredentialsGet(serviceCredentialsGet)
	handlerCredentialsList := handlers.NewCredentialsList(serviceCredentialsList)
	handlerCredentialsResetPassword := handlers.NewCredentialsResetPassword(serviceCredentialsUpdatePassword)
	handlerCredentialsUpdatePassword := handlers.NewCredentialsUpdatePassword(serviceCredentialsUpdatePassword)
	handlerCredentialsUpdateEmail := handlers.NewCredentialsUpdateEmail(serviceCredentialsUpdateEmail)
	handlerCredentialsUpdateRole := handlers.NewCredentialsUpdateRole(serviceCredentialsUpdateRole)

	handlerShortCodeCreateEmailUpdate := handlers.NewShortCodeCreateEmailUpdate(serviceShortCodeCreateEmailUpdate)
	handlerShortCodeCreatePasswordReset := handlers.NewShortCodeCreatePasswordReset(serviceShortCodeCreatePasswordReset)
	handlerShortCodeCreateRegister := handlers.NewShortCodeCreateRegister(serviceShortCodeCreateRegister)

	handlerTokenCreate := handlers.NewTokenCreate(serviceTokenCreate)
	handlerTokenCreateAnon := handlers.NewTokenCreateAnon(serviceTokenCreateAnon)
	handlerTokenRefresh := handlers.NewTokenRefresh(serviceTokenRefresh)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(cfg.Api.Timeouts.Request))
	router.Use(middleware.RequestSize(cfg.Api.MaxRequestSize))
	router.Use(cfg.Otel.HttpHandler())
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Api.Cors.AllowedOrigins,
		AllowedHeaders:   cfg.Api.Cors.AllowedHeaders,
		AllowCredentials: cfg.Api.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		MaxAge: cfg.Api.Cors.MaxAge,
	}))
	router.Use(cfg.Logger.Logger())

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
		withAuth(r, "credentials:email:patch").Patch("/email", handlerCredentialsUpdateEmail.ServeHTTP)
		withAuth(r, "credentials:password:patch").Patch("/password", handlerCredentialsUpdatePassword.ServeHTTP)
		withAuth(r, "credentials:password:reset").Put("/password", handlerCredentialsResetPassword.ServeHTTP)
		withAuth(r, "credentials:role:patch").Patch("/role", handlerCredentialsUpdateRole.ServeHTTP)
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
		Addr:              ":" + strconv.Itoa(cfg.Api.Port),
		Handler:           router,
		ReadTimeout:       cfg.Api.Timeouts.Read,
		ReadHeaderTimeout: cfg.Api.Timeouts.ReadHeader,
		WriteTimeout:      cfg.Api.Timeouts.Write,
		IdleTimeout:       cfg.Api.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	log.Println("Starting server on " + httpServer.Addr)

	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
}
