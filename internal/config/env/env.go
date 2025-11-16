package env

import (
	"os"
	"time"

	"github.com/a-novel/golib/config"
)

// prefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	SmtpTimeoutDefault = 20 * time.Second

	platformEmailUpdateUrlDefault   = "/ext/email/validate"
	platformPasswordResetUrlDefault = "/ext/password/reset"
	platformAccountCreateUrlDefault = "/ext/account/create"

	appNameDefault = "service-authentication"

	ServiceJsonKeysHostDefault = "localhost"
	ServiceJsonKeysPortDefault = 8080

	ApiPortDefault              = 8080
	ApiTimeoutReadDefault       = 5 * time.Second
	ApiTimeoutReadHeaderDefault = 3 * time.Second
	ApiTimeoutWriteDefault      = 10 * time.Second
	ApiTimeoutIdleDefault       = 30 * time.Second
	ApiTimeoutRequestDefault    = 15 * time.Second
	ApiMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault = false
	CorsMaxAgeDefault           = 3600
)

// Default values for environment variables, if applicable.
var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

// Raw values for environment variables.
var (
	postgresDsn     = getEnv("POSTGRES_DSN")
	postgresDsnTest = getEnv("POSTGRES_DSN_TEST")

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

	apiPort              = getEnv("API_PORT")
	apiMaxRequestSize    = getEnv("API_MAX_REQUEST_SIZE")
	apiTimeoutRead       = getEnv("API_TIMEOUT_READ")
	apiTimeoutReadHeader = getEnv("API_TIMEOUT_READ_HEADER")
	apiTimeoutWrite      = getEnv("API_TIMEOUT_WRITE")
	apiTimeoutIdle       = getEnv("API_TIMEOUT_IDLE")
	apiTimeoutRequest    = getEnv("API_TIMEOUT_REQUEST")
	corsAllowedOrigins   = getEnv("API_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders   = getEnv("API_CORS_ALLOWED_HEADERS")
	corsAllowCredentials = getEnv("API_CORS_ALLOW_CREDENTIALS")
	corsMaxAge           = getEnv("API_CORS_MAX_AGE")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

	superAdminEmail    = getEnv("SUPER_ADMIN_EMAIL")
	superAdminPassword = getEnv("SUPER_ADMIN_PASSWORD")
)

var (
	// PostgresDsn is the url used to connect to the postgres database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn
	// PostgresDsnTest is the url used to connect to the postgres database test instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsnTest = postgresDsnTest

	// PlatformAuthUrl points to the authentication web client. It is used to insert URLs in emails.
	PlatformAuthUrl = platformAuthUrl
	// PlatformAuthUpdateEmailUrl points to a web client page used to complete the email update process.
	// It is used to insert URLs in emails.
	PlatformAuthUpdateEmailUrl = config.LoadEnv(
		platformAuthUpdateEmailUrl,
		PlatformAuthUrl+platformEmailUpdateUrlDefault,
		config.StringParser,
	)
	// PlatformAuthUpdatePasswordUrl points to a web client page used to complete the password reset process.
	// It is used to insert URLs in emails.
	PlatformAuthUpdatePasswordUrl = config.LoadEnv(
		platformAuthUpdatePasswordUrl,
		PlatformAuthUrl+platformPasswordResetUrlDefault,
		config.StringParser,
	)
	// PlatformAuthRegisterUrl points to a web client page used to complete the registration process.
	// It is used to insert URLs in emails.
	PlatformAuthRegisterUrl = config.LoadEnv(
		platformAuthRegisterUrl,
		PlatformAuthUrl+platformAccountCreateUrlDefault,
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

	// SmtpAddr is the address of the SMTP service used to send mails.
	//
	// It should be in the form <domain>:<host>.
	SmtpAddr = smtpAddr
	// SmtpSenderName defines the name that will appear as the sender in outgoing emails.
	SmtpSenderName = smtpSenderName
	// SmtpSenderEmail is the email used to send outgoing smtp emails.
	SmtpSenderEmail = smtpSenderEmail
	// SmtpSenderPassword is the plain password used to connect to the SmtpSenderEmail account. This is a sensitive
	// value that should be handled appropriately.
	SmtpSenderPassword = smtpSenderPassword
	// SmtpSenderDomain is the domain used for sending Smtp emails. It should match the hostname of the SmtpAddr.
	SmtpSenderDomain = smtpSenderDomain
	// SmtpTimeout is the timeout value when attempting to send an email.
	SmtpTimeout = config.LoadEnv(smtpTimeout, SmtpTimeoutDefault, config.DurationParser)
	// SmtpForceUnencrypted hacks the smtp client into sending plain credentials over any connection. This setting is
	// used because Go Smtp implementation refuses to send plain credentials over non-TLS connections, except for
	// localhost. But because we use Docker, localhost is not always the name of our actual local host.
	//
	// This setting MUST NEVER be set in production.
	SmtpForceUnencrypted = config.LoadEnv(smtpForceUnencrypted, false, config.BoolParser)

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, appNameDefault, config.StringParser)
	// Otel flag configures whether to use Open Telemetry or not.
	//
	// See: https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// ApiPort is the port on which the rest api server will listen for incoming requests.
	ApiPort              = config.LoadEnv(apiPort, ApiPortDefault, config.IntParser)
	ApiMaxRequestSize    = config.LoadEnv(apiMaxRequestSize, ApiMaxRequestSizeDefault, config.Int64Parser)
	ApiTimeoutRead       = config.LoadEnv(apiTimeoutRead, ApiTimeoutReadDefault, config.DurationParser)
	ApiTimeoutReadHeader = config.LoadEnv(apiTimeoutReadHeader, ApiTimeoutReadHeaderDefault, config.DurationParser)
	ApiTimeoutWrite      = config.LoadEnv(apiTimeoutWrite, ApiTimeoutWriteDefault, config.DurationParser)
	ApiTimeoutIdle       = config.LoadEnv(apiTimeoutIdle, ApiTimeoutIdleDefault, config.DurationParser)
	ApiTimeoutRequest    = config.LoadEnv(apiTimeoutRequest, ApiTimeoutRequestDefault, config.DurationParser)
	CorsAllowedOrigins   = config.LoadEnv(
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
	// its email does not require to be validated.
	SuperAdminEmail = superAdminEmail
	// SuperAdminPassword sets the password for the default super-admin on the platform.
	SuperAdminPassword = superAdminPassword
)
