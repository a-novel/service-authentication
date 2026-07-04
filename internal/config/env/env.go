package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
)

// prefix is prepended to every configuration environment variable name. It lets a project that
// embeds this package namespace the variables so they do not clash with its own.
var prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	SmtpTimeoutDefault = 20 * time.Second

	PlatformEmailUpdateUrlDefault   = "/ext/email/validate"
	PlatformPasswordResetUrlDefault = "/ext/password/reset"
	PlatformAccountCreateUrlDefault = "/ext/account/create"

	AppNameDefault = "service-authentication"

	ServiceJsonKeysHostDefault = "localhost"
	ServiceJsonKeysPortDefault = 8080

	RestPortDefault              = 8080
	RestTimeoutReadDefault       = 15 * time.Second
	RestTimeoutReadHeaderDefault = 3 * time.Second
	RestTimeoutWriteDefault      = 30 * time.Second
	RestTimeoutIdleDefault       = 60 * time.Second
	RestTimeoutRequestDefault    = 60 * time.Second
	RestMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault  = false
	CorsMaxAgeDefault            = 3600
)

// Default values for environment variables, if applicable.
var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

// Raw values for environment variables.
var (
	postgresDsn = getEnv("POSTGRES_DSN")

	platformAuthUrl               = getEnv("PLATFORM_AUTH_URL")
	platformAuthUpdateEmailUrl    = getEnv("PLATFORM_AUTH_URL_UPDATE_EMAIL")
	platformAuthUpdatePasswordUrl = getEnv("PLATFORM_AUTH_URL_UPDATE_PASSWORD")
	platformAuthRegisterUrl       = getEnv("PLATFORM_AUTH_URL_REGISTER")

	serviceJsonKeysHost = getEnv("SERVICE_JSON_KEYS_HOST")
	serviceJsonKeysPort = getEnv("SERVICE_JSON_KEYS_PORT")

	smtpAddr             = getEnv("SMTP_ADDR")
	smtpSenderName       = getEnv("SMTP_SENDER_NAME")
	smtpSenderEmail      = getEnv("SMTP_SENDER_EMAIL")
	smtpSenderPassword   = getEnv("SMTP_SENDER_PASSWORD")
	smtpSenderDomain     = getEnv("SMTP_SENDER_DOMAIN")
	smtpTimeout          = getEnv("SMTP_TIMEOUT")
	smtpForceUnencrypted = getEnv("SMTP_FORCE_UNENCRYPTED")

	appName = getEnv("APP_NAME")
	otel    = getEnv("OTEL")

	restPort              = getEnv("REST_PORT")
	restMaxRequestSize    = getEnv("REST_MAX_REQUEST_SIZE")
	restTimeoutRead       = getEnv("REST_TIMEOUT_READ")
	restTimeoutReadHeader = getEnv("REST_TIMEOUT_READ_HEADER")
	restTimeoutWrite      = getEnv("REST_TIMEOUT_WRITE")
	restTimeoutIdle       = getEnv("REST_TIMEOUT_IDLE")
	restTimeoutRequest    = getEnv("REST_TIMEOUT_REQUEST")
	corsAllowedOrigins    = getEnv("REST_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders    = getEnv("REST_CORS_ALLOWED_HEADERS")
	corsAllowCredentials  = getEnv("REST_CORS_ALLOW_CREDENTIALS")
	corsMaxAge            = getEnv("REST_CORS_MAX_AGE")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

	superAdminEmail    = getEnv("SUPER_ADMIN_EMAIL")
	superAdminPassword = getEnv("SUPER_ADMIN_PASSWORD")
)

var (
	// PostgresDsn is the URL used to connect to the Postgres database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn

	// PlatformAuthUrl points to the authentication web client. It is used to insert URLs in emails.
	PlatformAuthUrl = platformAuthUrl
	// PlatformAuthUpdateEmailUrl points to a web client page used to complete the email update process.
	// It is used to insert URLs in emails.
	PlatformAuthUpdateEmailUrl = config.LoadEnv(
		platformAuthUpdateEmailUrl,
		PlatformAuthUrl+PlatformEmailUpdateUrlDefault,
		config.StringParser,
	)
	// PlatformAuthUpdatePasswordUrl points to a web client page used to complete the password reset process.
	// It is used to insert URLs in emails.
	PlatformAuthUpdatePasswordUrl = config.LoadEnv(
		platformAuthUpdatePasswordUrl,
		PlatformAuthUrl+PlatformPasswordResetUrlDefault,
		config.StringParser,
	)
	// PlatformAuthRegisterUrl points to a web client page used to complete the registration process.
	// It is used to insert URLs in emails.
	PlatformAuthRegisterUrl = config.LoadEnv(
		platformAuthRegisterUrl,
		PlatformAuthUrl+PlatformAccountCreateUrlDefault,
		config.StringParser,
	)

	// ServiceJsonKeysHost points to the host name (without protocol / port) on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysHost = config.LoadEnv(serviceJsonKeysHost, ServiceJsonKeysHostDefault, config.StringParser)
	// ServiceJsonKeysPort points to the port on which the JSON Keys Service is hosted.
	//
	// See https://github.com/a-novel/service-json-keys
	ServiceJsonKeysPort = config.LoadEnv(serviceJsonKeysPort, ServiceJsonKeysPortDefault, config.IntParser)

	// SmtpAddr is the address of the SMTP server used to send emails.
	//
	// It should be in the form domain:port.
	SmtpAddr = smtpAddr
	// SmtpSenderName is the display name that appears as the sender on outgoing emails.
	SmtpSenderName = smtpSenderName
	// SmtpSenderEmail is the address outgoing emails are sent from.
	SmtpSenderEmail = smtpSenderEmail
	// SmtpSenderPassword is the plain password used to authenticate the SmtpSenderEmail account. It is a sensitive
	// value and should be handled accordingly.
	SmtpSenderPassword = smtpSenderPassword
	// SmtpSenderDomain is the domain used when sending emails. It should match the host of SmtpAddr.
	SmtpSenderDomain = smtpSenderDomain
	// SmtpTimeout bounds how long a single email send may take.
	SmtpTimeout = config.LoadEnv(smtpTimeout, SmtpTimeoutDefault, config.DurationParser)
	// SmtpForceUnencrypted forces the SMTP client to send plain credentials over any connection. Go's SMTP client
	// refuses to send plain credentials over a non-TLS connection except to localhost, and under Docker the mail host
	// is rarely reachable as localhost, so this override is needed for local runs.
	//
	// It must never be set in production.
	SmtpForceUnencrypted = config.LoadEnv(smtpForceUnencrypted, false, config.BoolParser)

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// Otel enables OpenTelemetry instrumentation.
	//
	// See https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// RestPort is the port on which the rest server will listen for incoming requests.
	RestPort              = config.LoadEnv(restPort, RestPortDefault, config.IntParser)
	RestMaxRequestSize    = config.LoadEnv(restMaxRequestSize, RestMaxRequestSizeDefault, config.Int64Parser)
	RestTimeoutRead       = config.LoadEnv(restTimeoutRead, RestTimeoutReadDefault, config.DurationParser)
	RestTimeoutReadHeader = config.LoadEnv(restTimeoutReadHeader, RestTimeoutReadHeaderDefault, config.DurationParser)
	RestTimeoutWrite      = config.LoadEnv(restTimeoutWrite, RestTimeoutWriteDefault, config.DurationParser)
	RestTimeoutIdle       = config.LoadEnv(restTimeoutIdle, RestTimeoutIdleDefault, config.DurationParser)
	RestTimeoutRequest    = config.LoadEnv(restTimeoutRequest, RestTimeoutRequestDefault, config.DurationParser)
	CorsAllowedOrigins    = config.LoadEnv(
		corsAllowedOrigins, CorsAllowedOriginsDefault, config.SliceParser(config.StringParser),
	)
	CorsAllowedHeaders = config.LoadEnv(
		corsAllowedHeaders, CorsAllowedHeadersDefault, config.SliceParser(config.StringParser),
	)
	CorsAllowCredentials = config.LoadEnv(corsAllowCredentials, CorsAllowCredentialsDefault, config.BoolParser)
	CorsMaxAge           = config.LoadEnv(corsMaxAge, CorsMaxAgeDefault, config.IntParser)

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = gcloudProjectId
	// SuperAdminEmail sets the email address for the default super-admin on the platform. Unlike other accounts,
	// its email does not need to be validated.
	SuperAdminEmail = superAdminEmail
	// SuperAdminPassword sets the password for the default super-admin on the platform.
	SuperAdminPassword = superAdminPassword
)
