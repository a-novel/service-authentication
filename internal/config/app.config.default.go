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

// OtelFlushTimeout bounds how long the OpenTelemetry exporter waits to flush pending spans on
// shutdown.
const OtelFlushTimeout = 2 * time.Second

// SentryFlushTimeout bounds how long a binary waits on exit to send its pending Sentry events.
const SentryFlushTimeout = 2 * time.Second

// OtelPresetDefault is the OpenTelemetry exporter of the servers. It exports nothing unless OTEL is
// set, then exports to Google Cloud when GCLOUD_PROJECT_ID is set, and to standard output otherwise.
var OtelPresetDefault = newOtelPreset()

// LoggerDev writes human-readable logs to stdout for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd emits structured logs for the Google Cloud Logging backend.
var LoggerProd = &loggingpresets.LogGcloud{
	ProjectID: env.GcloudProjectID,
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
			env.GcloudProjectID == "",
			&grpcf.LocalCredentialsProvider{},
			&grpcf.GcloudCredentialsProvider{
				Host: env.ServiceJsonKeysHost,
			},
		),
	},
	Permissions:      authconfig.PermissionsConfigDefault,
	Waitlist:         authconfig.Waitlist{URL: env.WaitlistURL, Secret: env.WaitlistSecret, Timeout: env.WaitlistTimeout},
	ShortCodesConfig: authconfig.ShortCodesPresetDefault,
	SmtpUrlsConfig: authconfig.SmtpUrls{
		UpdateEmail:    env.PlatformAuthUpdateEmailURL,
		UpdatePassword: env.PlatformAuthUpdatePasswordURL,
		Register:       env.PlatformAuthRegisterURL,
	},

	Smtp: lo.Ternary[smtp.Sender](env.SmtpAddr == "", smtp.NewDebugSender(nil), &smtp.ProdSender{
		Addr:                env.SmtpAddr,
		Name:                env.SmtpSenderName,
		Email:               env.SmtpSenderEmail,
		Username:            env.SmtpUsername,
		Password:            env.SmtpSenderPassword,
		Domain:              env.SmtpSenderDomain,
		ForceUnencryptedTLS: env.SmtpForceUnencrypted,
		Timeout:             env.SmtpTimeout,
	}),
	Otel:   OtelPresetDefault,
	Sentry: env.Sentry,
	Logger: lo.Ternary[logging.Log](env.GcloudProjectID == "", LoggerDev, LoggerProd),
	HttpLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectID == "",
		&loggingpresets.HTTPLocal{
			BaseLogger: LoggerDev,
		},
		&loggingpresets.HTTPGcloud{
			BaseLogger: LoggerProd,
		},
	),
	Postgres: PostgresPresetDefault,
}

// newOtelPreset picks the exporter [OtelPresetDefault] describes.
func newOtelPreset() otel.Config {
	switch {
	case !env.Otel:
		return &otelpresets.Disabled{}
	case env.GcloudProjectID == "":
		return &otelpresets.Local{FlushTimeout: OtelFlushTimeout}
	default:
		return &otelpresets.Gcloud{ProjectID: env.GcloudProjectID, FlushTimeout: OtelFlushTimeout}
	}
}
