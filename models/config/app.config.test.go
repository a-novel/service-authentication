package config

import (
	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/models"
)

func AppPresetTest(port int) App[*otelpresets.LocalOtelConfig, postgres.Config, *smtp.TestSender] {
	return App[*otelpresets.LocalOtelConfig, postgres.Config, *smtp.TestSender]{
		App: Main{
			Name: config.LoadEnv(getEnv("APP_NAME"), AppName, config.StringParser),
		},
		API: API{
			Port:           port,
			MaxRequestSize: config.LoadEnv(getEnv("API_MAX_REQUEST_SIZE"), APIMaxRequestSize, config.Int64Parser),
			Timeouts: APITimeouts{
				Read: config.LoadEnv(getEnv("API_TIMEOUT_READ"), APITimeoutRead, config.DurationParser),
				ReadHeader: config.LoadEnv(
					getEnv("API_TIMEOUT_READ_HEADER"), APITimeoutReadHeader, config.DurationParser,
				),
				Write:   config.LoadEnv(getEnv("API_TIMEOUT_WRITE"), APITimeoutWrite, config.DurationParser),
				Idle:    config.LoadEnv(getEnv("API_TIMEOUT_IDLE"), APITimeoutIdle, config.DurationParser),
				Request: config.LoadEnv(getEnv("API_TIMEOUT_REQUEST"), APITimeoutRequest, config.DurationParser),
			},
			Cors: Cors{
				AllowedOrigins: config.LoadEnv(
					getEnv("API_CORS_ALLOWED_ORIGINS"), APICorsAllowedOrigins, config.SliceParser(config.StringParser),
				),
				AllowedHeaders: config.LoadEnv(
					getEnv("API_CORS_ALLOWED_HEADERS"), APICorsAllowedHeaders, config.SliceParser(config.StringParser),
				),
				AllowCredentials: config.LoadEnv(
					getEnv("API_CORS_ALLOW_CREDENTIALS"), APICorsAllowCredentials, config.BoolParser,
				),
				MaxAge: config.LoadEnv(getEnv("API_CORS_MAX_AGE"), APICorsMaxAge, config.IntParser),
			},
		},

		DependenciesConfig: Dependencies{
			JSONKeysURL: getEnv("JSON_KEYS_SERVICE_TEST_URL"),
		},
		PermissionsConfig: PermissionsConfigDefault,
		ShortCodesConfig:  ShortCodesPresetDefault,
		SMTPURLsConfig: models.SMTPURLsConfig{
			UpdateEmail: config.LoadEnv(
				getEnv("AUTH_PLATFORM_URL_UPDATE_EMAIL"),
				getEnv("AUTH_PLATFORM_URL")+"/ext/email/validate",
				config.StringParser,
			),
			UpdatePassword: config.LoadEnv(
				getEnv("AUTH_PLATFORM_URL_UPDATE_PASSWORD"),
				getEnv("AUTH_PLATFORM_URL")+"/ext/password/reset",
				config.StringParser,
			),
			Register: config.LoadEnv(
				getEnv("AUTH_PLATFORM_URL_REGISTER"),
				getEnv("AUTH_PLATFORM_URL")+"/ext/account/create",
				config.StringParser,
			),
		},

		SMTP:     smtp.NewTestSender(),
		Otel:     &OtelDev,
		Postgres: PostgresPresetTest,
	}
}
