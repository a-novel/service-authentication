package config

import (
	"time"

	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// SMTPPingTimeout bounds the health check's SMTP ping. A mail can take minutes on a slow relay, and
// the health check still answers within seconds.
const SMTPPingTimeout = 10 * time.Second

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
		OAuthToken:             env.RestRateLimitOAuthToken,
		WaitlistMaxConcurrent:  env.WaitlistMaxConcurrent,
		WaitlistMaxRequestSize: env.WaitlistMaxRequestSize,
		Argon2MaxConcurrent:    env.Argon2MaxConcurrent,
	},
	ServiceToken: env.ServiceToken,

	Dependencies: Dependencies{ServiceJSONKeys: env.ServiceJSONKeys},
	Permissions:  authconfig.PermissionsConfigDefault,
	Waitlist:     authconfig.Waitlist{URL: env.WaitlistURL, Secret: env.WaitlistSecret, Timeout: env.WaitlistTimeout},
	ShortCodes:   authconfig.ShortCodesPresetDefault,
	OAuth:        authconfig.OAuth{AccountPlatformCaller: env.AccountPlatformCaller, Clients: env.OAuthClients},
	SMTPURLs: authconfig.SMTPURLs{
		UpdateEmail:    env.PlatformAuthUpdateEmailURL,
		UpdatePassword: env.PlatformAuthUpdatePasswordURL,
		Register:       env.PlatformAuthRegisterURL,
	},

	SMTP:       newSMTPSender(env.SMTPTimeout),
	SMTPPing:   newSMTPSender(SMTPPingTimeout),
	Otel:       env.Telemetry.Otel,
	Logger:     env.Telemetry.Logger,
	RestLogger: env.Telemetry.HTTPLogger,
	Postgres:   PostgresPresetDefault,
}

// newSMTPSender sends through the relay SMTP_ADDR names, within timeout, or prints every mail when
// SMTP_ADDR is unset.
func newSMTPSender(timeout time.Duration) smtp.Sender {
	if env.SMTPAddr == "" {
		return smtp.NewDebugSender(nil)
	}

	return &smtp.ProdSender{
		Addr:                env.SMTPAddr,
		Name:                env.SMTPSenderName,
		Email:               env.SMTPSenderEmail,
		Username:            env.SMTPUsername,
		Password:            env.SMTPSenderPassword,
		Domain:              env.SMTPSenderDomain,
		ForceUnencryptedTLS: env.SMTPForceUnencrypted,
		Timeout:             timeout,
	}
}
