package cmdpkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"
	jkconfig "github.com/a-novel/service-json-keys/models/config"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel/service-authentication/internal/api"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/models/config"
)

func App[Otel otel.Config, Pg postgres.Config, SMTP smtp.Sender](
	ctx context.Context, config config.App[Otel, Pg, SMTP],
) error {
	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================
	otel.SetAppName(config.App.Name)

	err := otel.InitOtel(config.Otel)
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}
	defer config.Otel.Flush()

	// Don't override the context if it already has a bun.IDB
	_, err = postgres.GetContext(ctx)
	if err != nil {
		ctx, err = postgres.NewContext(ctx, config.Postgres)
		if err != nil {
			return fmt.Errorf("init postgres: %w", err)
		}
	}

	jkClient, err := jkpkg.NewAPIClient(ctx, config.DependenciesConfig.JSONKeysURL)
	if err != nil {
		return fmt.Errorf("create JSON keys client: %w", err)
	}

	signer := jkpkg.NewClaimsSigner(jkClient)

	accessTokenVerifier, err := jkpkg.NewClaimsVerifier[models.AccessTokenClaims](jkClient, jkconfig.JWKSPresetDefault)
	if err != nil {
		return fmt.Errorf("create access token verifier: %w", err)
	}

	refreshTokenVerifier, err := jkpkg.NewClaimsVerifier[models.RefreshTokenClaims](
		jkClient, jkconfig.JWKSPresetDefault,
	)
	if err != nil {
		return fmt.Errorf("create refresh token verifier: %w", err)
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

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

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	consumeRefreshTokenService := services.NewConsumeRefreshTokenService(
		services.NewConsumeRefreshTokenServiceSource(
			selectCredentialsDAO,
			signer,
			accessTokenVerifier,
			refreshTokenVerifier,
		),
	)
	createShortCodeService := services.NewCreateShortCodeService(insertShortCodeDAO, config.ShortCodesConfig)
	consumeShortCodeService := services.NewConsumeShortCodeService(
		services.NewConsumeShortCodeSource(selectShortCodeDAO, deleteShortCodeDAO),
	)

	loginService := services.NewLoginService(services.NewLoginServiceSource(
		selectCredentialsByEmailDAO,
		signer,
	))
	loginAnonService := services.NewLoginAnonService(signer)

	emailExistsService := services.NewEmailExistsService(emailExistsDAO)
	registerService := services.NewRegisterService(services.NewRegisterSource(
		insertCredentialsDAO, signer, consumeShortCodeService,
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
	getUserService := services.NewSelectUserService(selectCredentialsDAO)

	requestRegisterService := services.NewRequestRegisterService(
		services.NewRequestRegisterServiceSource(createShortCodeService, config.SMTP),
		config.ShortCodesConfig,
		config.SMTPURLsConfig,
	)
	defer requestRegisterService.Wait()

	requestEmailUpdateService := services.NewRequestEmailUpdateService(
		services.NewRequestEmailUpdateServiceSource(createShortCodeService, config.SMTP),
		config.ShortCodesConfig,
		config.SMTPURLsConfig,
	)
	defer requestEmailUpdateService.Wait()

	requestPasswordResetService := services.NewRequestPasswordResetService(
		services.NewRequestPasswordResetSource(selectCredentialsByEmailDAO, createShortCodeService, config.SMTP),
		config.ShortCodesConfig,
		config.SMTPURLsConfig,
	)
	defer requestPasswordResetService.Wait()

	// =================================================================================================================
	// SETUP ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(config.API.Timeouts.Request))
	router.Use(middleware.RequestSize(config.API.MaxRequestSize))
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
	router.Use(config.Otel.HTTPHandler())

	handler := &api.API{
		LoginService:               loginService,
		LoginAnonService:           loginAnonService,
		ConsumeRefreshTokenService: consumeRefreshTokenService,

		RequestRegisterService:      requestRegisterService,
		RequestEmailUpdateService:   requestEmailUpdateService,
		RequestPasswordResetService: requestPasswordResetService,

		RegisterService:       registerService,
		EmailExistsService:    emailExistsService,
		UpdateEmailService:    updateEmailService,
		UpdatePasswordService: updatePasswordService,
		UpdateRoleService:     updateRoleService,

		ListUsersService: listUsersService,
		GetUserService:   getUserService,

		JKClient: jkClient,
	}

	securityHandler, err := api.NewSecurity(accessTokenVerifier, config.PermissionsConfig)
	if err != nil {
		return fmt.Errorf("create security handler: %w", err)
	}

	apiServer, err := apimodels.NewServer(handler, securityHandler)
	if err != nil {
		return fmt.Errorf("new api server: %w", err)
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

	err = httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
