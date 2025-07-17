package cmdpkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"
	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel/service-authentication/internal/api"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type AppAppConfig struct {
	Name string `json:"name" yaml:"name"`
}

type DependencyConfig struct {
	JSONKeysURL string `json:"jsonKeysURL" yaml:"jsonKeysURL"`
}

type AppApiTimeoutsConfig struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
}

type AppCorsConfig struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

type AppAPIConfig struct {
	Port           int                  `json:"port"           yaml:"port"`
	Timeouts       AppApiTimeoutsConfig `json:"timeouts"       yaml:"timeouts"`
	MaxRequestSize int64                `json:"maxRequestSize" yaml:"maxRequestSize"`
	Cors           AppCorsConfig        `json:"cors"           yaml:"cors"`
}

type AppConfig[Otel otel.Config, Pg postgres.Config, SMTP smtp.Sender] struct {
	App AppAppConfig `json:"app" yaml:"app"`
	API AppAPIConfig `json:"api" yaml:"api"`

	DependencyConfig  DependencyConfig         `json:"dependencies" yaml:"dependencies"`
	PermissionsConfig models.PermissionsConfig `json:"permissions"  yaml:"permissions"`
	ShortCodesConfig  models.ShortCodesConfig  `json:"shortCodes"   yaml:"shortCodes"`
	SMTPURLsConfig    models.SMTPURLsConfig    `json:"smtpUrls"     yaml:"smtpUrls"`

	SMTP     SMTP `json:"smtp"     yaml:"smtp"`
	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}

func App[Otel otel.Config, Pg postgres.Config, SMTP smtp.Sender](
	ctx context.Context, config AppConfig[Otel, Pg, SMTP],
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

	jkClient, err := jkPkg.NewAPIClient(ctx, config.DependencyConfig.JSONKeysURL)
	if err != nil {
		return fmt.Errorf("create JSON keys client: %w", err)
	}

	signer := jkPkg.NewClaimsSigner(jkClient)

	accessTokenVerifier, err := jkPkg.NewClaimsVerifier[models.AccessTokenClaims](jkClient, jkModels.DefaultJWKSConfig)
	if err != nil {
		return fmt.Errorf("create access token verifier: %w", err)
	}

	refreshTokenVerifier, err := jkPkg.NewClaimsVerifier[models.RefreshTokenClaims](
		jkClient, jkModels.DefaultJWKSConfig,
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
