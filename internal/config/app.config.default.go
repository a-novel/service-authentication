package config

import (
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/logging"
	loggingpresets "github.com/a-novel/golib/logging/presets"
	"github.com/a-novel/golib/otel"
	otelpresets "github.com/a-novel/golib/otel/presets"
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
	ForceUnencryptedTls: env.SmtpForceUnencrypted,
}

var OtelProd = otelpresets.Gcloud{
	ProjectID:    env.GcloudProjectId,
	FlushTimeout: OtelFlushTimeout,
}

var OtelDev = otelpresets.Local{
	FlushTimeout: OtelFlushTimeout,
}

var NoOtel = otelpresets.Disabled{}

var LoggerProd = loggingpresets.HttpGcloud{
	ProjectId: env.GcloudProjectId,
}

var LoggerDev = loggingpresets.HttpLocal{}

var AppPresetDefault = App{
	App: Main{
		Name: env.AppName,
	},
	Api: API{
		Port:           env.ApiPort,
		MaxRequestSize: env.ApiMaxRequestSize,
		Timeouts: APITimeouts{
			Read:       env.ApiTimeoutRead,
			ReadHeader: env.ApiTimeoutReadHeader,
			Write:      env.ApiTimeoutWrite,
			Idle:       env.ApiTimeoutIdle,
			Request:    env.ApiTimeoutRequest,
		},
		Cors: Cors{
			AllowedOrigins:   env.CorsAllowedOrigins,
			AllowedHeaders:   env.CorsAllowedHeaders,
			AllowCredentials: env.CorsAllowCredentials,
			MaxAge:           env.CorsMaxAge,
		},
	},

	DependenciesConfig: Dependencies{
		ServiceJsonKeysUrl: env.ServiceJsonKeysUrl,
	},
	Permissions:      PermissionsConfigDefault,
	ShortCodesConfig: ShortCodesPresetDefault,
	SmtpUrlsConfig: SmtpUrls{
		UpdateEmail:    env.PlatformAuthUpdateEmailUrl,
		UpdatePassword: env.PlatformAuthUpdatePasswordUrl,
		Register:       env.PlatformAuthRegisterUrl,
		Timeout:        env.SmtpTimeout,
	},

	Smtp: lo.Ternary[smtp.Sender](env.SmtpAddr == "", smtp.NewDebugSender(nil), &SMTPProd),
	Otel: lo.If[otel.Config](!env.Otel, &NoOtel).
		ElseIf(env.GcloudProjectId == "", &OtelDev).
		Else(&OtelProd),
	Logger:   lo.Ternary[logging.HttpConfig](env.GcloudProjectId == "", &LoggerDev, &LoggerProd),
	Postgres: PostgresPresetDefault,
}
