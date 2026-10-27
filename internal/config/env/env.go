package env

import (
	"os"
	"runtime"
	"time"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/sentryf"
)

// prefix is prepended to every configuration variable name that this package reads.
// Set SERVICE_AUTHENTICATION_ENV_PREFIX when embedding the service in another project,
// where unprefixed variable names could collide with the host project's own.
var prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

const (
	// SmtpTimeoutDefault bounds one email send.
	SmtpTimeoutDefault = 20 * time.Second
	// SmtpMaxConcurrentDefault bounds the SMTP connections a burst or a stalled server can hold open.
	SmtpMaxConcurrentDefault = 16
	// WaitlistTimeoutDefault bounds one request to the optional Google sheet writer.
	WaitlistTimeoutDefault = 10 * time.Second
	// WaitlistMaxConcurrentDefault keeps optional Google calls from exhausting REST capacity.
	WaitlistMaxConcurrentDefault = 4
	// WaitlistMaxRequestSizeDefault allows the bounded email/language payload with JSON escaping overhead.
	WaitlistMaxRequestSizeDefault = 8 << 10

	// PlatformEmailUpdateURLDefault is the email-validation page, under PlatformAuthURL.
	PlatformEmailUpdateURLDefault = "/ext/email/validate"
	// PlatformPasswordResetURLDefault is the password-reset page, under PlatformAuthURL.
	PlatformPasswordResetURLDefault = "/ext/password/reset"
	// PlatformAccountCreateURLDefault is the registration page, under PlatformAuthURL.
	PlatformAccountCreateURLDefault = "/ext/account/create"

	// AppNameDefault names the application in logs and traces when APP_NAME is unset.
	AppNameDefault = "service-authentication"

	// ServiceJsonKeysHostDefault reaches a JSON Keys service on the same host.
	ServiceJsonKeysHostDefault = "localhost"
	// ServiceJsonKeysPortDefault is the JSON Keys service's default gRPC port.
	ServiceJsonKeysPortDefault = 8080

	// TimeoutShutdownDefault ends a server's drain 6 seconds after SIGTERM, so its Sentry and
	// OpenTelemetry flushes, up to 2 seconds each, finish before Cloud Run kills it at 10 seconds.
	TimeoutShutdownDefault = 6 * time.Second

	// PostgresMaxOpenConnsDefault caps the connections one instance opens. Multiplied by the replica
	// count, the cap stays well under PostgreSQL's default max_connections of 100. That leaves room
	// for the migration job and a psql session. A spike waits inside the process for a free connection.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit, so a burst keeps the connections it is about
	// to reuse.
	PostgresMaxIdleConnsDefault = 20
	// PostgresPortDefault is PostgreSQL's standard port.
	PostgresPortDefault = 5432
	// PostgresTLSEnabledDefault encrypts the connection unless a deployment turns it off.
	PostgresTLSEnabledDefault = true
)

var (
	// Argon2MaxConcurrentDefault runs one hash per CPU the container may use, as Keycloak does.
	// Hashing is CPU-bound, so more concurrent hashes add memory, 64 MiB each, but no throughput.
	Argon2MaxConcurrentDefault = runtime.GOMAXPROCS(0)

	// RestRateLimitStrictDefault caps anonymous tokens and invitation requests per caller.
	RestRateLimitStrictDefault = config.Must(httpf.ParseRate("3/5m"))
	// RestRateLimitLoginBurstDefault leaves a forgotten password a few attempts within a minute.
	RestRateLimitLoginBurstDefault = config.Must(httpf.ParseRate("10/1m"))
	// RestRateLimitLoginDefault leaves a forgotten password a few attempts over ten minutes.
	RestRateLimitLoginDefault = config.Must(httpf.ParseRate("30/10m"))
	// RestRateLimitMailboxDefault caps the emails one address receives from a route.
	RestRateLimitMailboxDefault = config.Must(httpf.ParseRate("3/1h"))
)

var (
	// PostgresHost is the PostgreSQL server hostname or IP address.
	PostgresHost = getEnv("POSTGRES_HOST")
	// PostgresPort is the PostgreSQL server port.
	PostgresPort = config.LoadEnv(getEnv("POSTGRES_PORT"), PostgresPortDefault, config.IntParser)
	// PostgresUser is the PostgreSQL login role.
	PostgresUser = getEnv("POSTGRES_USER")
	// PostgresPassword is the PostgreSQL login credential.
	PostgresPassword = getEnv("POSTGRES_PASSWORD")
	// PostgresDatabase is the PostgreSQL database name.
	PostgresDatabase = getEnv("POSTGRES_DATABASE")
	// PostgresTLSEnabled controls transport encryption for the PostgreSQL connection.
	PostgresTLSEnabled = config.LoadEnv(getEnv("POSTGRES_TLS_ENABLED"), PostgresTLSEnabledDefault, config.BoolParser)

	// PostgresMaxOpenConns is the maximum number of open connections to the database.
	PostgresMaxOpenConns = config.LoadEnv(getEnv("POSTGRES_MAX_OPEN_CONNS"), PostgresMaxOpenConnsDefault, config.IntParser)
	// PostgresMaxIdleConns is the maximum number of connections kept open while idle.
	PostgresMaxIdleConns = config.LoadEnv(getEnv("POSTGRES_MAX_IDLE_CONNS"), PostgresMaxIdleConnsDefault, config.IntParser)

	// PlatformAuthURL is the base URL of the authentication web client, prefixed onto the
	// links inserted in emails.
	PlatformAuthURL = getEnv("PLATFORM_AUTH_URL")
	// PlatformAuthUpdateEmailURL is the web client page linked from update emails to
	// complete the email update.
	PlatformAuthUpdateEmailURL = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_UPDATE_EMAIL"), PlatformAuthURL+PlatformEmailUpdateURLDefault, config.StringParser,
	)
	// PlatformAuthUpdatePasswordURL is the web client page linked from reset emails to
	// complete the password reset.
	PlatformAuthUpdatePasswordURL = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_UPDATE_PASSWORD"), PlatformAuthURL+PlatformPasswordResetURLDefault, config.StringParser,
	)
	// PlatformAuthRegisterURL is the web client page linked from registration emails to
	// complete the account creation.
	PlatformAuthRegisterURL = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_REGISTER"), PlatformAuthURL+PlatformAccountCreateURLDefault, config.StringParser,
	)

	// ServiceJsonKeysHost is the hostname of the JSON Keys service, without scheme or port.
	ServiceJsonKeysHost = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_HOST"), ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort is the gRPC port of the JSON Keys service.
	ServiceJsonKeysPort = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_PORT"), ServiceJsonKeysPortDefault, config.IntParser)

	// SmtpAddr is the SMTP server address, as domain:port.
	SmtpAddr = getEnv("SMTP_ADDR")
	// SmtpSenderName is the display name that appears as the sender on outgoing emails.
	SmtpSenderName = getEnv("SMTP_SENDER_NAME")
	// SmtpSenderEmail is the address outgoing emails are sent from.
	SmtpSenderEmail = getEnv("SMTP_SENDER_EMAIL")
	// SmtpUsername is the identity used for SMTP authentication. An empty value lets the sender
	// reuse SmtpSenderEmail. It is unused without SmtpSenderPassword.
	SmtpUsername = getEnv("SMTP_USERNAME")
	// SmtpSenderPassword is the SMTP authentication password. An empty value sends without
	// authentication, to a relay that admits the service by its IP address. The connection must
	// then upgrade with STARTTLS.
	SmtpSenderPassword = getEnv("SMTP_SENDER_PASSWORD")
	// SmtpSenderDomain is the server name authentication is bound to. It should match the host of
	// SmtpAddr, and it is unused without SmtpSenderPassword.
	SmtpSenderDomain = getEnv("SMTP_SENDER_DOMAIN")
	// SmtpTimeout bounds how long a single email send may take.
	SmtpTimeout = config.LoadEnv(getEnv("SMTP_TIMEOUT"), SmtpTimeoutDefault, config.DurationParser)
	// SmtpMaxConcurrent caps concurrent email deliveries; excess sends wait for a slot.
	SmtpMaxConcurrent = config.LoadEnv(getEnv("SMTP_MAX_CONCURRENT"), SmtpMaxConcurrentDefault, config.IntParser)
	// SmtpForceUnencrypted lets the SMTP client talk over a non-TLS connection. With a password, it
	// sends the credentials in the clear, which Go otherwise permits only towards localhost. Without
	// one, it lifts the STARTTLS requirement. Local runs under Docker need it because the mail host
	// offers no TLS and answers to another name.
	//
	// It must never be set in production.
	SmtpForceUnencrypted = config.LoadEnv(getEnv("SMTP_FORCE_UNENCRYPTED"), false, config.BoolParser)

	// WaitlistURL is the Google Apps Script /exec deployment URL; empty disables the integration.
	WaitlistURL = getEnv("WAITLIST_URL")
	// WaitlistSecret is a server-only signing key shared with the Google script properties.
	WaitlistSecret = getEnv("WAITLIST_SECRET")
	// WaitlistTimeout bounds the Google request and its content redirect.
	WaitlistTimeout = config.LoadEnv(getEnv("WAITLIST_TIMEOUT"), WaitlistTimeoutDefault, config.DurationParser)
	// WaitlistMaxConcurrent caps the waitlist requests waiting on Google at once, per instance.
	WaitlistMaxConcurrent = config.LoadEnv(
		getEnv("WAITLIST_MAX_CONCURRENT"), WaitlistMaxConcurrentDefault, config.IntParser,
	)
	// WaitlistMaxRequestSize caps a waitlist request body, in bytes.
	WaitlistMaxRequestSize = config.LoadEnv(
		getEnv("WAITLIST_MAX_REQUEST_SIZE"), WaitlistMaxRequestSizeDefault, config.Int64Parser,
	)

	// Argon2MaxConcurrent caps the password hashes running at once, per instance. Each allocates
	// 64 MiB, so size it to the instance's memory.
	Argon2MaxConcurrent = config.LoadEnv(getEnv("ARGON2_MAX_CONCURRENT"), Argon2MaxConcurrentDefault, config.IntParser)

	// RestRateLimitStrict caps each caller of a route issuing anonymous tokens or invitation requests.
	RestRateLimitStrict = config.LoadEnv(getEnv("REST_RATE_LIMIT_STRICT"), RestRateLimitStrictDefault, httpf.ParseRate)
	// RestRateLimitLoginBurst caps each caller's requests checking a password or short code, logins
	// first, over a short window.
	RestRateLimitLoginBurst = config.LoadEnv(
		getEnv("REST_RATE_LIMIT_LOGIN_BURST"), RestRateLimitLoginBurstDefault, httpf.ParseRate,
	)
	// RestRateLimitLogin caps the same requests over a longer window, per caller and per email or
	// account named.
	RestRateLimitLogin = config.LoadEnv(getEnv("REST_RATE_LIMIT_LOGIN"), RestRateLimitLoginDefault, httpf.ParseRate)
	// RestRateLimitMailbox caps the emails each address receives from one route.
	RestRateLimitMailbox = config.LoadEnv(getEnv("REST_RATE_LIMIT_MAILBOX"), RestRateLimitMailboxDefault, httpf.ParseRate)

	// AppName names the application in logs and traces.
	AppName = config.LoadEnv(getEnv("APP_NAME"), AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry instrumentation.
	Otel = config.LoadEnv(getEnv("OTEL"), false, config.BoolParser)
	// Sentry configures the fault reporter from the SENTRY_* variables. The reporter sends nothing
	// without SENTRY_DSN.
	Sentry = sentryf.LoadConfig(getEnv)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// GcloudProjectID is the Google Cloud project the service runs in. When set, the JSON Keys
	// connection authenticates with Google ID tokens.
	GcloudProjectID = getEnv("GCLOUD_PROJECT_ID")

	// OTLPEndpoint is the standard OTEL_EXPORTER_OTLP_ENDPOINT. When set, traces and logs export
	// over OTLP/HTTP, which the OTEL_EXPORTER_OTLP_* variables configure. When empty, they go to
	// standard output. The OpenTelemetry SDK reads these variables without the prefix, so this one
	// is read without it too.
	OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the drain, which fits
// Cloud Run's kill window. A send still running when the drain ends is cut.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Timeouts.Shutdown = TimeoutShutdownDefault

	return defaults
}
