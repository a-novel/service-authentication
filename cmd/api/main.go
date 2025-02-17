package main

import (
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/a-novel-kit/context"
	zeromiddleware "github.com/a-novel-kit/middlewares/zerolog"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	"github.com/a-novel/authentication/models"
)

const (
	MaxRequestSize = 2 << 20 // 2 MiB
)

func main() {
	// =================================================================================================================
	// LOAD DEPENDENCIES (EXTERNAL)
	// =================================================================================================================
	logger := zerolog.New(os.Stderr).With().
		Str("app", "authentication").
		Timestamp().
		Logger()

	if config.LoggerColor {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	logger.Info().Msg("starting application...")

	// Initialize all contexts at once.
	ctx, err := lib.NewAgoraContext(context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("initialize agora context")
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

	// SERVICES --------------------------------------------------------------------------------------------------------

	searchKeysService := services.NewSearchKeysService(searchKeysDAO)
	selectKeyService := services.NewSelectKeyService(selectKeyDAO)

	authSignSource := services.NewAuthPrivateKeysProvider(searchKeysService)
	authVerifySource := services.NewAuthPublicKeysProvider(searchKeysService)
	issueTokenService := services.NewIssueTokenService(authSignSource)
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

	requestRegisterService := services.NewRequestRegisterService(createShortCodeService)
	defer requestRegisterService.Wait()

	requestEmailUpdateService := services.NewRequestEmailUpdateService(createShortCodeService)
	defer requestEmailUpdateService.Wait()

	requestPasswordResetService := services.NewRequestPasswordResetService(
		services.NewRequestPasswordResetSource(selectCredentialsByEmailDAO, createShortCodeService),
	)
	defer requestPasswordResetService.Wait()

	// =================================================================================================================
	// SETUP ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	// MIDDLEWARES -----------------------------------------------------------------------------------------------------

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(zeromiddleware.ZeroLog(&logger))
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

	// RUN -------------------------------------------------------------------------------------------------------------

	handler := &api.API{
		LoginService:     loginService,
		LoginAnonService: loginAnonService,

		SelectKeyService:  selectKeyService,
		SearchKeysService: searchKeysService,

		RequestRegisterService:      requestRegisterService,
		RequestEmailUpdateService:   requestEmailUpdateService,
		RequestPasswordResetService: requestPasswordResetService,

		RegisterService:       registerService,
		EmailExistsService:    emailExistsService,
		UpdateEmailService:    updateEmailService,
		UpdatePasswordService: updatePasswordService,
	}

	apiPermissions := map[codegen.OperationName][]models.Permission{
		codegen.PingOperation: {},

		codegen.CheckSessionOperation:      {},
		codegen.CreateSessionOperation:     {},
		codegen.CreateAnonSessionOperation: {},

		codegen.GetPublicKeyOperation:   {"jwk:read"},
		codegen.ListPublicKeysOperation: {"jwk:read"},

		codegen.RequestRegistrationOperation:  {"register:request"},
		codegen.RequestEmailUpdateOperation:   {"email:update:request"},
		codegen.RequestPasswordResetOperation: {"password:reset:request"},

		codegen.RegisterOperation:       {"register"},
		codegen.EmailExistsOperation:    {"email:exists"},
		codegen.UpdateEmailOperation:    {"email:update"},
		codegen.UpdatePasswordOperation: {"password:update"},

		codegen.ResetPasswordOperation: {"password:reset"},
	}

	securityHandler, err := api.NewSecurity(apiPermissions, config.Permissions, authenticateService)
	if err != nil {
		logger.Fatal().Err(err).Msg("initialize security handler")
	}

	apiServer, err := codegen.NewServer(handler, securityHandler)
	if err != nil {
		logger.Fatal().Err(err).Msg("initialize server")
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

	logger.Info().
		Str("address", httpServer.Addr).
		Msg("application started!")

	if err = httpServer.ListenAndServe(); err != nil {
		logger.Fatal().Err(err).Msg("application stopped")
	}
}
