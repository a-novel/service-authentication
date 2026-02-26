package config

import (
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/muesli/termenv"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

const (
	OtelFlushTimeout = 2 * time.Second
)

var LoggerDev = &loggingpresets.LogLocal{
	Out:      os.Stdout,
	Renderer: lipgloss.NewRenderer(os.Stdout, termenv.WithTTY(true)),
}

var LoggerProd = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

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
		ServiceJsonKeysPort: env.ServiceJsonKeysPort,
		ServiceJsonKeysHost: env.ServiceJsonKeysHost,
		ServiceJsonKeysCredentials: lo.Ternary[grpcf.CredentialsProvider](
			env.GcloudProjectId == "",
			&grpcf.LocalCredentialsProvider{},
			&grpcf.GcloudCredentialsProvider{
				Host: env.ServiceJsonKeysHost,
			},
		),
	},
	Permissions:      PermissionsConfigDefault,
	ShortCodesConfig: ShortCodesPresetDefault,
	SmtpUrlsConfig: SmtpUrls{
		UpdateEmail:    env.PlatformAuthUpdateEmailUrl,
		UpdatePassword: env.PlatformAuthUpdatePasswordUrl,
		Register:       env.PlatformAuthRegisterUrl,
		Timeout:        env.SmtpTimeout,
	},

	Smtp: lo.Ternary[smtp.Sender](env.SmtpAddr == "", smtp.NewDebugSender(nil), &smtp.ProdSender{
		Addr:                env.SmtpAddr,
		Name:                env.SmtpSenderName,
		Email:               env.SmtpSenderEmail,
		Password:            env.SmtpSenderPassword,
		Domain:              env.SmtpSenderDomain,
		ForceUnencryptedTls: env.SmtpForceUnencrypted,
	}),
	Otel: lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
		ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
			FlushTimeout: OtelFlushTimeout,
		}).
		Else(&otelpresets.Gcloud{
			ProjectID:    env.GcloudProjectId,
			FlushTimeout: OtelFlushTimeout,
		}),
	Logger: lo.Ternary[logging.Log](env.GcloudProjectId == "", LoggerDev, LoggerProd),
	HttpLogger: lo.Ternary[logging.HttpConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HttpLocal{
			BaseLogger: LoggerDev,
		},
		&loggingpresets.HttpGcloud{
			BaseLogger: LoggerProd,
		},
	),
	Postgres: PostgresPresetDefault,
}
