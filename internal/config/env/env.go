package env

import (
	"os"
	"runtime"
	"time"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
)

// prefix is prepended to every configuration environment variable name, so a project
// embedding this package can namespace them against its own.
var prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	SmtpTimeoutDefault = 20 * time.Second
	// SmtpMaxConcurrentDefault bounds the SMTP connections a burst or a stalled server can hold open.
	SmtpMaxConcurrentDefault = 16
	// WaitlistTimeoutDefault bounds one request to the optional Google sheet writer.
	WaitlistTimeoutDefault = 10 * time.Second
	// WaitlistMaxConcurrentDefault keeps optional Google calls from exhausting REST capacity.
	WaitlistMaxConcurrentDefault = 4
	// WaitlistMaxRequestSizeDefault allows the bounded email/language payload with JSON escaping overhead.
	WaitlistMaxRequestSizeDefault = 8 << 10

	PlatformEmailUpdateUrlDefault   = "/ext/email/validate"
	PlatformPasswordResetUrlDefault = "/ext/password/reset"
	PlatformAccountCreateUrlDefault = "/ext/account/create"

	AppNameDefault = "service-authentication"

	ServiceJsonKeysHostDefault = "localhost"
	ServiceJsonKeysPortDefault = 8080

	// RestTimeoutShutdownDefault clears SmtpTimeoutDefault, leaving room for a send that starts
	// just before the signal to finish. The container runtime's stop grace period has to clear
	// this in turn; builds/podman-compose.yaml sets one.
	RestTimeoutShutdownDefault = 25 * time.Second

	// PostgresMaxOpenConnsDefault keeps the pool well under a stock PostgreSQL
	// max_connections of 100 once multiplied by a service's replica count, leaving
	// room for the migration job and a psql session. Go's own default is unlimited,
	// which turns a spike into connection refusals for everything on that database
	// rather than queueing inside this process.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit so a burst does not close
	// connections it is about to reopen.
	PostgresMaxIdleConnsDefault = 20
	PostgresPortDefault         = 5432
	PostgresTLSEnabledDefault   = true
)

// Default rate limits, written "<requests>/<window>" as in their variables.
var (
	// Argon2MaxConcurrentDefault runs one hash per CPU the container may use, as Keycloak does.
	// Hashing is CPU-bound, so more concurrent hashes add memory, 64 MiB each, but no throughput.
	Argon2MaxConcurrentDefault = runtime.GOMAXPROCS(0)

	RestRateLimitStrictDefault = config.Must(httpf.ParseRate("3/5m"))
	// RestRateLimitLoginBurstDefault and RestRateLimitLoginDefault leave a forgotten password a few
	// attempts.
	RestRateLimitLoginBurstDefault = config.Must(httpf.ParseRate("10/1m"))
	RestRateLimitLoginDefault      = config.Must(httpf.ParseRate("30/10m"))
	RestRateLimitMailboxDefault    = config.Must(httpf.ParseRate("3/1h"))
)

var (
	// PostgresDsn is the legacy URL used when PostgresHost is empty.
	//
	// Configure the discrete PostgreSQL fields for new deployments.
	PostgresDsn = getEnv("POSTGRES_DSN")
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

	// PlatformAuthUrl is the base URL of the authentication web client, prefixed onto the
	// links inserted in emails.
	PlatformAuthUrl = getEnv("PLATFORM_AUTH_URL")
	// PlatformAuthUpdateEmailUrl is the web client page linked from update emails to
	// complete the email update.
	PlatformAuthUpdateEmailUrl = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_UPDATE_EMAIL"), PlatformAuthUrl+PlatformEmailUpdateUrlDefault, config.StringParser,
	)
	// PlatformAuthUpdatePasswordUrl is the web client page linked from reset emails to
	// complete the password reset.
	PlatformAuthUpdatePasswordUrl = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_UPDATE_PASSWORD"), PlatformAuthUrl+PlatformPasswordResetUrlDefault, config.StringParser,
	)
	// PlatformAuthRegisterUrl is the web client page linked from registration emails to
	// complete the account creation.
	PlatformAuthRegisterUrl = config.LoadEnv(
		getEnv("PLATFORM_AUTH_URL_REGISTER"), PlatformAuthUrl+PlatformAccountCreateUrlDefault, config.StringParser,
	)

	// ServiceJsonKeysHost points to the host name (without protocol / port) on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysHost = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_HOST"), ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort points to the port on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysPort = config.LoadEnv(getEnv("SERVICE_JSON_KEYS_PORT"), ServiceJsonKeysPortDefault, config.IntParser)

	// SmtpAddr is the address of the SMTP server used to send emails.
	//
	// It should be in the form domain:port.
	SmtpAddr = getEnv("SMTP_ADDR")
	// SmtpSenderName is the display name that appears as the sender on outgoing emails.
	SmtpSenderName = getEnv("SMTP_SENDER_NAME")
	// SmtpSenderEmail is the address outgoing emails are sent from.
	SmtpSenderEmail = getEnv("SMTP_SENDER_EMAIL")
	// SmtpUsername is the identity used for SMTP authentication. An empty value lets the sender
	// reuse SmtpSenderEmail.
	SmtpUsername = getEnv("SMTP_USERNAME")
	// SmtpSenderPassword is the credential used for SMTP authentication. It is sensitive and
	// should be handled accordingly.
	SmtpSenderPassword = getEnv("SMTP_SENDER_PASSWORD")
	// SmtpSenderDomain is the domain used when sending emails. It should match the host of SmtpAddr.
	SmtpSenderDomain = getEnv("SMTP_SENDER_DOMAIN")
	// SmtpTimeout bounds how long a single email send may take.
	SmtpTimeout = config.LoadEnv(getEnv("SMTP_TIMEOUT"), SmtpTimeoutDefault, config.DurationParser)
	// SmtpMaxConcurrent caps concurrent email deliveries; excess sends wait for a slot.
	SmtpMaxConcurrent = config.LoadEnv(getEnv("SMTP_MAX_CONCURRENT"), SmtpMaxConcurrentDefault, config.IntParser)
	// SmtpForceUnencrypted lets the SMTP client send plain credentials over a non-TLS
	// connection, which Go otherwise permits only towards localhost. Local runs under
	// Docker need it because the mail host answers to another name.
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

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(getEnv("APP_NAME"), AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry instrumentation.
	//
	// See https://opentelemetry.io/
	Otel = config.LoadEnv(getEnv("OTEL"), false, config.BoolParser)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the shutdown, which
// waits for detached mail.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Timeouts.Shutdown = RestTimeoutShutdownDefault

	return defaults
}
