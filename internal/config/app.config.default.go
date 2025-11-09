package config

import (
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/otel"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/internal/config/env"
)

const (
	OtelFlushTimeout = 2 * time.Second
)

var SMTPProd = smtp.ProdSender{
	Addr:                env.SmtpAddr,
	Name:                env.SmtpSenderName,
	Email:               env.SmtpSenderEmail,
	Password:            env.SmtpSenderPassword,
	Domain:              env.SmtpSenderDomain,
	ForceUnencryptedTls: config.LoadEnv(env.SmtpForceUnencrypted, false, config.BoolParser),
}

var OtelProd = otelpresets.GCloudOtelConfig{
	ProjectID:    env.GcloudProjectId,
	FlushTimeout: OtelFlushTimeout,
}

var OtelDev = otelpresets.LocalOtelConfig{
	PrettyPrint:  config.LoadEnv(env.PrettyConsole, true, config.BoolParser),
	FlushTimeout: OtelFlushTimeout,
}

var AppPresetDefault = App[otel.Config, postgres.Config, smtp.Sender]{
	App: Main{
		Name: config.LoadEnv(env.AppName, env.AppNameDefault, config.StringParser),
	},
	Api: API{
		Port:           config.LoadEnv(env.ApiPort, env.ApiPortDefault, config.IntParser),
		MaxRequestSize: config.LoadEnv(env.ApiMaxRequestSize, env.ApiMaxRequestSizeDefault, config.Int64Parser),
		Timeouts: APITimeouts{
			Read: config.LoadEnv(env.ApiTimeoutRead, env.ApiTimeoutReadDefault, config.DurationParser),
			ReadHeader: config.LoadEnv(
				env.ApiTimeoutReadHeader, env.ApiTimeoutReadHeaderDefault, config.DurationParser,
			),
			Write:   config.LoadEnv(env.ApiTimeoutWrite, env.ApiTimeoutWriteDefault, config.DurationParser),
			Idle:    config.LoadEnv(env.ApiTimeoutIdle, env.ApiTimeoutIdleDefault, config.DurationParser),
			Request: config.LoadEnv(env.ApiTimeoutRequest, env.ApiTimeoutRequestDefault, config.DurationParser),
		},
		Cors: Cors{
			AllowedOrigins: config.LoadEnv(
				env.ApiCorsAllowedOrigins, env.ApiCorsAllowedOriginsDefault, config.SliceParser(config.StringParser),
			),
			AllowedHeaders: config.LoadEnv(
				env.ApiCorsAllowedHeaders, env.ApiCorsAllowedHeadersDefault, config.SliceParser(config.StringParser),
			),
			AllowCredentials: config.LoadEnv(
				env.ApiCorsAllowCredentials, env.ApiCorsAllowCredentialsDefault, config.BoolParser,
			),
			MaxAge: config.LoadEnv(env.ApiCorsMaxAge, env.ApiCorsMaxAgeDefault, config.IntParser),
		},
	},

	DependenciesConfig: Dependencies{
		JsonKeysServiceUrl: env.JsonKeysServiceUrl,
	},
	Permissions:      PermissionsConfigDefault,
	ShortCodesConfig: ShortCodesPresetDefault,
	SmtpUrlsConfig: SmtpUrls{
		UpdateEmail: config.LoadEnv(
			env.AuthPlatformUpdateEmailUrl,
			env.AuthPlatformUrl+"/ext/email/validate",
			config.StringParser,
		),
		UpdatePassword: config.LoadEnv(
			env.AuthPlatformUpdatePasswordUrl,
			env.AuthPlatformUrl+"/ext/password/reset",
			config.StringParser,
		),
		Register: config.LoadEnv(
			env.AuthPlatformRegisterUrl,
			env.AuthPlatformUrl+"/ext/account/create",
			config.StringParser,
		),
		Timeout: config.LoadEnv(env.SmtpTimeout, env.SmtpTimeoutDefault, config.DurationParser),
	},

	Smtp:     lo.Ternary[smtp.Sender](env.SmtpAddr == "", smtp.NewDebugSender(nil), &SMTPProd),
	Otel:     lo.Ternary[otel.Config](env.GcloudProjectId == "", &OtelDev, &OtelProd),
	Postgres: PostgresPresetDefault,
}
