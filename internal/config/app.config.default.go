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
	"github.com/a-novel-kit/golib/sentryf"
	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// OtelFlushTimeout bounds how long the OpenTelemetry exporter waits to flush pending spans on
// shutdown.
const OtelFlushTimeout = 2 * time.Second

// SentryFlushTimeout bounds how long a server waits on exit to send its pending Sentry events, inside
// the platform's shutdown grace period.
const SentryFlushTimeout = 2 * time.Second

// SmtpPingTimeout bounds the health check's SMTP ping. A mail can take minutes on a slow relay, and
// the health check still answers within seconds.
const SmtpPingTimeout = 10 * time.Second

// SentryJobFlushTimeout bounds a job's wait on exit for its pending Sentry events. It outlasts
// sentryf.SendTimeout, so an event sent before the network came up still leaves.
const SentryJobFlushTimeout = sentryf.SendTimeout + time.Minute

// OtelJobFlushTimeout bounds a job's wait on exit for its pending spans and logs. A job often ends
// before the network lets its first connection out. The exporter then sends the spans, then the
// logs, within 30 seconds each.
const OtelJobFlushTimeout = time.Minute

// OtelPresetDefault is the OpenTelemetry exporter of the servers. It exports nothing unless OTEL is
// set, then exports over OTLP/HTTP when OTEL_EXPORTER_OTLP_ENDPOINT is set, and to standard output
// otherwise. The liveness probe produces no span.
var OtelPresetDefault = newOtelPreset(OtelFlushTimeout)

// OtelJobPresetDefault is the OpenTelemetry exporter of the maintenance jobs. It exports as
// [OtelPresetDefault] does, and flushes within [OtelJobFlushTimeout].
var OtelJobPresetDefault = newOtelPreset(OtelJobFlushTimeout)

// exportsOTLP reports whether the servers export over OTLP: OTEL is set, and so is the OTLP
// endpoint. Their logs then go through the same backend, and their spans record each request.
var exportsOTLP = env.Otel && env.OTLPEndpoint != ""

// LoggerDev writes human-readable logs to stdout for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends application logs through the OpenTelemetry backend, linked to their trace.
var LoggerProd = &loggingpresets.LogOTel{}

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
	ServiceToken: servicetoken.Config{
		Issuer:   env.ServiceTokenIssuer,
		Audience: env.ServiceTokenAudience,
		Callers:  env.ServiceTokenCallers,
	},

	DependenciesConfig: Dependencies{
		ServiceJsonKeysPort:        env.ServiceJsonKeysPort,
		ServiceJsonKeysHost:        env.ServiceJsonKeysHost,
		ServiceJsonKeysCredentials: newJsonKeysCredentials(),
	},
	Permissions:      authconfig.PermissionsConfigDefault,
	Waitlist:         authconfig.Waitlist{URL: env.WaitlistURL, Secret: env.WaitlistSecret, Timeout: env.WaitlistTimeout},
	ShortCodesConfig: authconfig.ShortCodesPresetDefault,
	SmtpUrlsConfig: authconfig.SmtpUrls{
		UpdateEmail:    env.PlatformAuthUpdateEmailURL,
		UpdatePassword: env.PlatformAuthUpdatePasswordURL,
		Register:       env.PlatformAuthRegisterURL,
	},

	Smtp:     newSmtpSender(env.SmtpTimeout),
	SmtpPing: newSmtpSender(SmtpPingTimeout),
	Otel:     OtelPresetDefault,
	Sentry:   env.Sentry,
	Logger:   lo.Ternary[logging.Log](!exportsOTLP, LoggerDev, LoggerProd),
	// The server span records each request once telemetry exports over OTLP.
	HttpLogger: lo.Ternary[logging.HTTPConfig](
		!exportsOTLP, &loggingpresets.HTTPLocal{BaseLogger: LoggerDev}, nil,
	),
	Postgres: PostgresPresetDefault,
}

// newOtelPreset picks the exporter [OtelPresetDefault] describes, flushing within flushTimeout.
func newOtelPreset(flushTimeout time.Duration) otel.Config {
	switch {
	case !env.Otel:
		return &otelpresets.Disabled{}
	case env.OTLPEndpoint == "":
		return &otelpresets.Local{FlushTimeout: flushTimeout}
	default:
		return &otelpresets.OTLP{UntracedPaths: []string{env.Rest.Liveness}, FlushTimeout: flushTimeout}
	}
}

// newJsonKeysCredentials authenticates the JSON Keys connection with the service token at
// SERVICE_TOKEN_PATH. Without one, it sends a Google ID token when the service runs on Google Cloud,
// and nothing otherwise.
func newJsonKeysCredentials() grpcf.CredentialsProvider {
	switch {
	case env.ServiceTokenPath != "":
		return &grpcf.ServiceTokenCredentialsProvider{ServiceTokenPath: env.ServiceTokenPath}
	case env.GcloudProjectID != "":
		return &grpcf.GcloudCredentialsProvider{Host: env.ServiceJsonKeysHost}
	default:
		return &grpcf.LocalCredentialsProvider{}
	}
}

// newSmtpSender sends through the relay SMTP_ADDR names, within timeout, or prints every mail when
// SMTP_ADDR is unset.
func newSmtpSender(timeout time.Duration) smtp.Sender {
	if env.SmtpAddr == "" {
		return smtp.NewDebugSender(nil)
	}

	return &smtp.ProdSender{
		Addr:                env.SmtpAddr,
		Name:                env.SmtpSenderName,
		Email:               env.SmtpSenderEmail,
		Username:            env.SmtpUsername,
		Password:            env.SmtpSenderPassword,
		Domain:              env.SmtpSenderDomain,
		ForceUnencryptedTLS: env.SmtpForceUnencrypted,
		Timeout:             timeout,
	}
}
