package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

const (
	// OtelFlushTimeout bounds how long the process waits to flush buffered telemetry on shutdown.
	OtelFlushTimeout = 2 * time.Second
)

// LoggerDev writes human-readable logs to stdout for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd emits structured logs for the Google Cloud Logging backend.
var LoggerProd = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

// AppPresetDefault is the application configuration read from the environment, with each field falling
// back to its documented default when the variable is unset.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		DowntimeStart: env.DowntimeStart,
	},
	Rest: env.Rest,
	Limits: authconfig.Limits{
		Strict:                 env.RestRateLimitStrict,
		LoginBurst:             env.RestRateLimitLoginBurst,
		Login:                  env.RestRateLimitLogin,
		Mailbox:                env.RestRateLimitMailbox,
		WaitlistMaxConcurrent:  env.WaitlistMaxConcurrent,
		WaitlistMaxRequestSize: env.WaitlistMaxRequestSize,
		Argon2MaxConcurrent:    env.Argon2MaxConcurrent,
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
	Waitlist:         authconfig.Waitlist{URL: env.WaitlistURL, Secret: env.WaitlistSecret, Timeout: env.WaitlistTimeout},
	ShortCodesConfig: ShortCodesPresetDefault,
	SmtpUrlsConfig: SmtpUrls{
		UpdateEmail:    env.PlatformAuthUpdateEmailUrl,
		UpdatePassword: env.PlatformAuthUpdatePasswordUrl,
		Register:       env.PlatformAuthRegisterUrl,
	},

	Smtp: lo.Ternary[smtp.Sender](env.SmtpAddr == "", smtp.NewDebugSender(nil), &smtp.ProdSender{
		Addr:                env.SmtpAddr,
		Name:                env.SmtpSenderName,
		Email:               env.SmtpSenderEmail,
		Username:            env.SmtpUsername,
		Password:            env.SmtpSenderPassword,
		Domain:              env.SmtpSenderDomain,
		ForceUnencryptedTls: env.SmtpForceUnencrypted,
		Timeout:             env.SmtpTimeout,
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
	HttpLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HTTPLocal{
			BaseLogger: LoggerDev,
		},
		&loggingpresets.HTTPGcloud{
			BaseLogger: LoggerProd,
		},
	),
	Postgres: PostgresPresetDefault,
}
