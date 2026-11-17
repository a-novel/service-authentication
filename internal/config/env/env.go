package env

import (
	"cmp"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"
	"github.com/a-novel-kit/golib/servicetoken"

	"github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// prefix is prepended to every configuration variable name that this package reads.
// Set SERVICE_AUTHENTICATION_ENV_PREFIX when embedding the service in another project,
// where unprefixed variable names could collide with the host project's own.
var prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

const (
	// SMTPTimeoutDefault bounds one email send.
	SMTPTimeoutDefault = 20 * time.Second
	// SMTPMaxConcurrentDefault bounds the SMTP connections a burst or a stalled server can hold open.
	SMTPMaxConcurrentDefault = 16
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

	// ServiceJSONKeysHostDefault reaches a JSON Keys service on the same host.
	ServiceJSONKeysHostDefault = "localhost"
	// ServiceJSONKeysPortDefault is the JSON Keys service's default gRPC port.
	ServiceJSONKeysPortDefault = 8080

	// AccountPlatformCallerDefault is the caller of the account platform in the cluster.
	AccountPlatformCallerDefault = "system:serviceaccount:account:web"

	// TimeoutShutdownDefault ends a server's drain 25 seconds after SIGTERM. That outlasts
	// SMTPTimeoutDefault, so a send that starts just before the signal finishes, and leaves the
	// OpenTelemetry flush, up to 2 seconds, inside the 30 seconds before Kubernetes kills the pod.
	TimeoutShutdownDefault = 25 * time.Second
)

var (
	// Argon2MaxConcurrentDefault runs one hash per CPU the container may use, as Keycloak does.
	// Hashing is CPU-bound, so more concurrent hashes add memory, 64 MiB each, but no throughput.
	Argon2MaxConcurrentDefault = runtime.GOMAXPROCS(0)

	// RestRateLimitStrictDefault caps anonymous tokens and invitation requests per caller.
	RestRateLimitStrictDefault = lo.Must(httpf.ParseRate("3/5m"))
	// RestRateLimitLoginBurstDefault leaves a forgotten password a few attempts within a minute.
	RestRateLimitLoginBurstDefault = lo.Must(httpf.ParseRate("10/1m"))
	// RestRateLimitLoginDefault leaves a forgotten password a few attempts over ten minutes.
	RestRateLimitLoginDefault = lo.Must(httpf.ParseRate("30/10m"))
	// RestRateLimitMailboxDefault caps the emails one address receives from a route.
	RestRateLimitMailboxDefault = lo.Must(httpf.ParseRate("3/1h"))
	// RestRateLimitOAuthTokenDefault lets a client sign a visitor in or refresh their session once a
	// second, sustained.
	RestRateLimitOAuthTokenDefault = lo.Must(httpf.ParseRate("60/1m"))
)

var (
	// Postgres reaches PostgreSQL through the POSTGRES_* variables.
	Postgres = postgrespresets.LoadConnection(getEnv, postgrespresets.ConnectionDefault)

	// PlatformAuthURL is the base URL of the authentication web client, prefixed onto the
	// links inserted in emails.
	PlatformAuthURL = getEnv("PLATFORM_AUTH_URL")
	// PlatformAuthUpdateEmailURL is the web client page linked from update emails to
	// complete the email update.
	PlatformAuthUpdateEmailURL = cmp.Or(
		getEnv("PLATFORM_AUTH_URL_UPDATE_EMAIL"), PlatformAuthURL+PlatformEmailUpdateURLDefault,
	)
	// PlatformAuthUpdatePasswordURL is the web client page linked from reset emails to
	// complete the password reset.
	PlatformAuthUpdatePasswordURL = cmp.Or(
		getEnv("PLATFORM_AUTH_URL_UPDATE_PASSWORD"), PlatformAuthURL+PlatformPasswordResetURLDefault,
	)
	// PlatformAuthRegisterURL is the web client page linked from registration emails to
	// complete the account creation.
	PlatformAuthRegisterURL = cmp.Or(getEnv("PLATFORM_AUTH_URL_REGISTER"), PlatformAuthURL+PlatformAccountCreateURLDefault)

	// ServiceJSONKeys locates the JSON Keys service through the SERVICE_JSON_KEYS_* variables.
	// SERVICE_TOKEN_PATH, the deprecated name of SERVICE_JSON_KEYS_TOKEN_PATH, applies while the new
	// one is unset.
	ServiceJSONKeys = grpcf.LoadClientConfig(getEnv, "SERVICE_JSON_KEYS", grpcf.ClientConfig{
		Host:      ServiceJSONKeysHostDefault,
		Port:      ServiceJSONKeysPortDefault,
		TokenPath: getEnv("SERVICE_TOKEN_PATH"),
	})

	// SMTPAddr is the SMTP server address, as domain:port.
	SMTPAddr = getEnv("SMTP_ADDR")
	// SMTPSenderName is the display name that appears as the sender on outgoing emails.
	SMTPSenderName = getEnv("SMTP_SENDER_NAME")
	// SMTPSenderEmail is the address outgoing emails are sent from.
	SMTPSenderEmail = getEnv("SMTP_SENDER_EMAIL")
	// SMTPUsername is the identity used for SMTP authentication. An empty value lets the sender
	// reuse SMTPSenderEmail. It is unused without SMTPSenderPassword.
	SMTPUsername = getEnv("SMTP_USERNAME")
	// SMTPSenderPassword is the SMTP authentication password. An empty value sends without
	// authentication, to a relay that admits the service by its IP address. The connection must
	// then upgrade with STARTTLS.
	SMTPSenderPassword = getEnv("SMTP_SENDER_PASSWORD")
	// SMTPSenderDomain is the server name authentication is bound to. It should match the host of
	// SMTPAddr, and it is unused without SMTPSenderPassword.
	SMTPSenderDomain = getEnv("SMTP_SENDER_DOMAIN")
	// SMTPTimeout bounds how long a single email send may take.
	SMTPTimeout = config.LoadEnv(getEnv("SMTP_TIMEOUT"), SMTPTimeoutDefault, time.ParseDuration)
	// SMTPMaxConcurrent caps concurrent email deliveries; excess sends wait for a slot.
	SMTPMaxConcurrent = config.LoadEnv(getEnv("SMTP_MAX_CONCURRENT"), SMTPMaxConcurrentDefault, strconv.Atoi)
	// SMTPForceUnencrypted lets the SMTP client talk over a non-TLS connection. With a password, it
	// sends the credentials in the clear, which Go otherwise permits only towards localhost. Without
	// one, it lifts the STARTTLS requirement. Local runs under Docker need it because the mail host
	// offers no TLS and answers to another name.
	//
	// It must never be set in production.
	SMTPForceUnencrypted = config.LoadEnv(getEnv("SMTP_FORCE_UNENCRYPTED"), false, strconv.ParseBool)

	// WaitlistURL is the Google Apps Script /exec deployment URL; empty disables the integration.
	WaitlistURL = getEnv("WAITLIST_URL")
	// WaitlistSecret is a server-only signing key shared with the Google script properties.
	WaitlistSecret = getEnv("WAITLIST_SECRET")
	// WaitlistTimeout bounds the Google request and its content redirect.
	WaitlistTimeout = config.LoadEnv(getEnv("WAITLIST_TIMEOUT"), WaitlistTimeoutDefault, time.ParseDuration)
	// WaitlistMaxConcurrent caps the waitlist requests waiting on Google at once, per instance.
	WaitlistMaxConcurrent = config.LoadEnv(
		getEnv("WAITLIST_MAX_CONCURRENT"), WaitlistMaxConcurrentDefault, strconv.Atoi,
	)
	// WaitlistMaxRequestSize caps a waitlist request body, in bytes.
	WaitlistMaxRequestSize = config.LoadEnv(
		getEnv("WAITLIST_MAX_REQUEST_SIZE"), WaitlistMaxRequestSizeDefault, config.Int64Parser,
	)

	// Argon2MaxConcurrent caps the password hashes running at once, per instance. Each allocates
	// 64 MiB, so size it to the instance's memory.
	Argon2MaxConcurrent = config.LoadEnv(getEnv("ARGON2_MAX_CONCURRENT"), Argon2MaxConcurrentDefault, strconv.Atoi)

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
	// RestRateLimitOAuthToken caps the token requests each client sends for each visitor.
	RestRateLimitOAuthToken = config.LoadEnv(
		getEnv("REST_RATE_LIMIT_OAUTH_TOKEN"), RestRateLimitOAuthTokenDefault, httpf.ParseRate,
	)

	// AppName names the application in logs and traces.
	AppName = cmp.Or(getEnv("APP_NAME"), AppNameDefault)
	// Telemetry is the OpenTelemetry backend of the server, and the loggers that match it.
	Telemetry = otelpresets.LoadTelemetry(getEnv, otelpresets.ServerFlushTimeout)
	// JobTelemetry is the OpenTelemetry backend of the maintenance jobs, which wait longer on exit
	// for their spans and logs to leave.
	JobTelemetry = otelpresets.LoadTelemetry(getEnv, otelpresets.JobFlushTimeout)

	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// ServiceToken names the service tokens the REST server accepts, through the SERVICE_TOKEN_*
	// variables. Without an issuer, the server refuses to start unless SERVICE_ISSUER_OPTIONAL is set.
	ServiceToken = servicetoken.LoadConfig(getEnv)

	// OAuthClients is the client registry, the platforms that sign people in through the account
	// platform, in the JSON form [auth.ParseClients] reads. Empty, no platform is a client.
	OAuthClients = config.LoadEnv(getEnv("OAUTH_CLIENTS"), nil, auth.ParseClients)
	// AccountPlatformCaller is the caller of the account platform, the only one that validates
	// authorization requests and issues codes.
	AccountPlatformCaller = cmp.Or(getEnv("ACCOUNT_PLATFORM_CALLER"), AccountPlatformCallerDefault)

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the drain, which
// outlasts a send and fits the pod's grace period. A send still running when the drain ends is cut.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Timeouts.Shutdown = TimeoutShutdownDefault

	return defaults
}
