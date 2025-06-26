package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	sentryhttp "github.com/getsentry/sentry-go/http"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

const (
	MaxRequestSize     = 2 << 20 // 2 MiB
	SentryFlushTimeout = 2 * time.Second
)

func main() {
	ctx := context.Background()
	// =================================================================================================================
	// LOAD DEPENDENCIES (EXTERNAL)
	// =================================================================================================================
	if err := sentry.Init(config.SentryClient); err != nil {
		log.Fatalf("initialize sentry: %v", err)
	}
	defer sentry.Flush(SentryFlushTimeout)

	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())

	logger := sentry.NewLogger(ctx)
	logger.SetAttributes(
		attribute.String("app", "agora"),
		attribute.String("service", "authentication"),
	)

	logger.Info(ctx, "starting application")

	ctx, err := lib.NewAgoraContext(ctx, config.DSN)
	if err != nil {
		logger.Fatalf(ctx, "initialize agora context: %v", err)
	}

	// =================================================================================================================
	// LOAD REPOSITORIES (INTERNAL)
	// =================================================================================================================

	// REPOSITORIES ----------------------------------------------------------------------------------------------------

	searchKeysDAO := dao.NewSearchKeysRepository()
	selectKeyDAO := dao.NewSelectKeyRepository()

	selectShortCodeDAO := dao.NewSelectShortCodeByParamsRepository()
	deleteShortCodeDAO := dao.NewDeleteShortCodeRepository()
	insertShortCodeDAO := dao.NewInsertShortCodeRepository()

	emailExistsDAO := dao.NewExistsCredentialsEmailRepository()
	selectCredentialsDAO := dao.NewSelectCredentialsRepository()
	selectCredentialsByEmailDAO := dao.NewSelectCredentialsByEmailRepository()
	insertCredentialsDAO := dao.NewInsertCredentialsRepository()
	updateEmailDAO := dao.NewUpdateCredentialsEmailRepository()
	updatePasswordDAO := dao.NewUpdateCredentialsPasswordRepository()
	updateRoleDAO := dao.NewUpdateCredentialsRoleRepository()

	listUsersDAO := dao.NewListUsersRepository()

	// SERVICES --------------------------------------------------------------------------------------------------------

	searchKeysService := services.NewSearchKeysService(searchKeysDAO)
	selectKeyService := services.NewSelectKeyService(selectKeyDAO)

	authSignSource := services.NewAuthPrivateKeysProvider(searchKeysService)
	authVerifySource := services.NewAuthPublicKeysProvider(searchKeysService)
	refreshSignSource := services.NewRefreshPrivateKeysProvider(searchKeysService)
	refreshVerifySource := services.NewRefreshPublicKeysProvider(searchKeysService)

	issueTokenService := services.NewIssueTokenService(authSignSource)
	issueRefreshTokenService := services.NewIssueRefreshTokenService(refreshSignSource)
	consumeRefreshTokenService := services.NewConsumeRefreshTokenService(
		services.NewNewConsumeRefreshTokenServiceSource(
			selectCredentialsDAO,
			issueTokenService,
		),
		authVerifySource,
		refreshVerifySource,
	)
	createShortCodeService := services.NewCreateShortCodeService(insertShortCodeDAO)
	consumeShortCodeService := services.NewConsumeShortCodeService(
		services.NewConsumeShortCodeSource(selectShortCodeDAO, deleteShortCodeDAO),
	)

	loginService := services.NewLoginService(services.NewLoginServiceSource(
		selectCredentialsByEmailDAO,
		issueTokenService,
	))
	loginAnonService := services.NewLoginAnonService(issueTokenService)
	authenticateService := services.NewAuthenticateService(authVerifySource)

	emailExistsService := services.NewEmailExistsService(emailExistsDAO)
	registerService := services.NewRegisterService(services.NewRegisterSource(
		insertCredentialsDAO, issueTokenService, consumeShortCodeService,
	))
	updateEmailService := services.NewUpdateEmailService(services.NewUpdateEmailSource(
		updateEmailDAO, consumeShortCodeService,
	))
	updatePasswordService := services.NewUpdatePasswordService(services.NewUpdatePasswordSource(
		selectCredentialsDAO, updatePasswordDAO, consumeShortCodeService,
	))
	updateRoleService := services.NewUpdateRoleService(services.NewUpdateRoleServiceSource(
		updateRoleDAO, selectCredentialsDAO,
	))

	listUsersService := services.NewListUsersService(listUsersDAO)

	smtpService := services.NewSMTPService()

	requestRegisterService := services.NewRequestRegisterService(
		services.NewRequestRegisterServiceSource(createShortCodeService, smtpService),
	)
	defer requestRegisterService.Wait()

	requestEmailUpdateService := services.NewRequestEmailUpdateService(
		services.NewRequestEmailUpdateServiceSource(createShortCodeService, smtpService),
	)
	defer requestEmailUpdateService.Wait()

	requestPasswordResetService := services.NewRequestPasswordResetService(
		services.NewRequestPasswordResetSource(selectCredentialsByEmailDAO, createShortCodeService, smtpService),
	)
	defer requestPasswordResetService.Wait()

	// =================================================================================================================
	// SETUP ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	// MIDDLEWARES -----------------------------------------------------------------------------------------------------

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(config.API.Timeouts.Request))
	router.Use(middleware.RequestSize(MaxRequestSize))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.API.Cors.AllowedOrigins,
		AllowedHeaders:   config.API.Cors.AllowedHeaders,
		AllowCredentials: config.API.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		MaxAge: config.API.Cors.MaxAge,
	}))

	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	router.Use(sentryHandler.Handle)

	// RUN -------------------------------------------------------------------------------------------------------------

	handler := &api.API{
		LoginService:               loginService,
		LoginAnonService:           loginAnonService,
		ConsumeRefreshTokenService: consumeRefreshTokenService,
		IssueRefreshTokenService:   issueRefreshTokenService,

		SelectKeyService:  selectKeyService,
		SearchKeysService: searchKeysService,

		RequestRegisterService:      requestRegisterService,
		RequestEmailUpdateService:   requestEmailUpdateService,
		RequestPasswordResetService: requestPasswordResetService,

		RegisterService:       registerService,
		EmailExistsService:    emailExistsService,
		UpdateEmailService:    updateEmailService,
		UpdatePasswordService: updatePasswordService,
		UpdateRoleService:     updateRoleService,

		ListUsersService: listUsersService,
	}

	securityHandler, err := api.NewSecurity(config.Permissions, authenticateService)
	if err != nil {
		logger.Fatalf(ctx, "start security handler: %v", err)
	}

	apiServer, err := codegen.NewServer(handler, securityHandler)
	if err != nil {
		logger.Fatalf(ctx, "start server: %v", err)
	}

	router.Mount("/v1/", http.StripPrefix("/v1", apiServer))

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(config.API.Port),
		Handler:           router,
		ReadTimeout:       config.API.Timeouts.Read,
		ReadHeaderTimeout: config.API.Timeouts.ReadHeader,
		WriteTimeout:      config.API.Timeouts.Write,
		IdleTimeout:       config.API.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	logger.SetAttributes(attribute.Int("server.port", config.API.Port))
	logger.Infof(ctx, "start http server on port %v", config.API.Port)

	if err = httpServer.ListenAndServe(); err != nil {
		logger.Fatalf(ctx, "start http server: %v", err)
	}
}
