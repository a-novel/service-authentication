package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/models"
)

const (
	SentryFlushTimeout = 2 * time.Second

	AppName = "service-authentication"

	APIPort                 = 8080
	APITimeoutRead          = 5 * time.Second
	APITimeoutReadHeader    = 3 * time.Second
	APITimeoutWrite         = 10 * time.Second
	APITimeoutIdle          = 30 * time.Second
	APITimeoutRequest       = 15 * time.Second
	APIMaxRequestSize       = 2 << 20 // 2 MiB
	APICorsAllowCredentials = false
	APICorsMaxAge           = 3600
)

var (
	APICorsAllowedOrigins = []string{"*"}
	APICorsAllowedHeaders = []string{"*"}
)

var isDebug = config.LoadEnv(
	lo.CoalesceOrEmpty(os.Getenv("SENTRY_DEBUG"), os.Getenv("DEBUG")), false, config.BoolParser,
)

var SMTPProd = smtp.ProdSender{
	Addr:     os.Getenv("SMTP_ADDR"),
	Name:     os.Getenv("SMTP_SENDER_NAME"),
	Email:    os.Getenv("SMTP_SENDER_EMAIL"),
	Password: os.Getenv("SMTP_SENDER_PASSWORD"),
	Domain:   os.Getenv("SMTP_SENDER_DOMAIN"),
}

var AppPresetDefault = App[*otelpresets.SentryOtelConfig, postgres.Config, smtp.Sender]{
	App: Main{
		Name: config.LoadEnv(os.Getenv("APP_NAME"), AppName, config.StringParser),
	},
	API: API{
		Port:           config.LoadEnv(os.Getenv("API_PORT"), APIPort, config.IntParser),
		MaxRequestSize: config.LoadEnv(os.Getenv("API_MAX_REQUEST_SIZE"), APIMaxRequestSize, config.Int64Parser),
		Timeouts: APITimeouts{
			Read: config.LoadEnv(os.Getenv("API_TIMEOUT_READ"), APITimeoutRead, config.DurationParser),
			ReadHeader: config.LoadEnv(
				os.Getenv("API_TIMEOUT_READ_HEADER"), APITimeoutReadHeader, config.DurationParser,
			),
			Write:   config.LoadEnv(os.Getenv("API_TIMEOUT_WRITE"), APITimeoutWrite, config.DurationParser),
			Idle:    config.LoadEnv(os.Getenv("API_TIMEOUT_IDLE"), APITimeoutIdle, config.DurationParser),
			Request: config.LoadEnv(os.Getenv("API_TIMEOUT_REQUEST"), APITimeoutRequest, config.DurationParser),
		},
		Cors: Cors{
			AllowedOrigins: config.LoadEnv(
				os.Getenv("API_CORS_ALLOWED_ORIGINS"), APICorsAllowedOrigins, config.SliceParser(config.StringParser),
			),
			AllowedHeaders: config.LoadEnv(
				os.Getenv("API_CORS_ALLOWED_HEADERS"), APICorsAllowedHeaders, config.SliceParser(config.StringParser),
			),
			AllowCredentials: config.LoadEnv(
				os.Getenv("API_CORS_ALLOW_CREDENTIALS"), APICorsAllowCredentials, config.BoolParser,
			),
			MaxAge: config.LoadEnv(os.Getenv("API_CORS_MAX_AGE"), APICorsMaxAge, config.IntParser),
		},
	},

	DependenciesConfig: Dependencies{
		JSONKeysURL: os.Getenv("JSON_KEYS_SERVICE_URL"),
	},
	PermissionsConfig: PermissionsConfigDefault,
	ShortCodesConfig:  ShortCodesPresetDefault,
	SMTPURLsConfig: models.SMTPURLsConfig{
		UpdateEmail: config.LoadEnv(
			os.Getenv("AUTH_PLATFORM_URL_UPDATE_EMAIL"),
			os.Getenv("AUTH_PLATFORM_URL")+"/ext/email/validate",
			config.StringParser,
		),
		UpdatePassword: config.LoadEnv(
			os.Getenv("AUTH_PLATFORM_URL_UPDATE_PASSWORD"),
			os.Getenv("AUTH_PLATFORM_URL")+"/ext/password/update",
			config.StringParser,
		),
		Register: config.LoadEnv(
			os.Getenv("AUTH_PLATFORM_URL_REGISTER"),
			os.Getenv("AUTH_PLATFORM_URL")+"/ext/account/create",
			config.StringParser,
		),
	},

	SMTP: lo.Ternary[smtp.Sender](os.Getenv("SMTP_ADDR") == "", smtp.NewDebugSender(nil), &SMTPProd),
	Otel: &otelpresets.SentryOtelConfig{
		DSN:          os.Getenv("SENTRY_DSN"),
		ServerName:   config.LoadEnv(os.Getenv("APP_NAME"), AppName, config.StringParser),
		Release:      os.Getenv("SENTRY_RELEASE"),
		Environment:  lo.CoalesceOrEmpty(os.Getenv("SENTRY_ENVIRONMENT"), os.Getenv("ENV")),
		FlushTimeout: config.LoadEnv(os.Getenv("SENTRY_FLUSH_TIMEOUT"), SentryFlushTimeout, config.DurationParser),
		Debug:        isDebug,
	},
	Postgres: PostgresPresetDefault,
}
