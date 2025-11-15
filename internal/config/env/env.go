package env

import (
	"os"
	"time"

	"github.com/a-novel/golib/config"
)

// Prefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var Prefix = os.Getenv("SERVICE_AUTHENTICATION_ENV_PREFIX")

func getEnv(name string) string {
	if Prefix != "" {
		return os.Getenv(Prefix + "_" + name)
	}

	return os.Getenv(name)
}

const (
	SmtpTimeoutDefault = 20 * time.Second

	AppNameDefault = "service-authentication"

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

var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

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
	PostgresDsn     = postgresDsn
	PostgresDsnTest = postgresDsnTest

	PlatformAuthUrl            = platformAuthUrl
	PlatformAuthUpdateEmailUrl = config.LoadEnv(
		platformAuthUpdateEmailUrl,
		PlatformAuthUrl+"/ext/email/validate",
		config.StringParser,
	)
	PlatformAuthUpdatePasswordUrl = config.LoadEnv(
		platformAuthUpdatePasswordUrl,
		PlatformAuthUrl+"/ext/password/reset",
		config.StringParser,
	)
	PlatformAuthRegisterUrl = config.LoadEnv(
		platformAuthRegisterUrl,
		PlatformAuthUrl+"/ext/account/create",
		config.StringParser,
	)

	ServiceJsonKeysHost = config.LoadEnv(serviceJsonKeysHost, ServiceJsonKeysHostDefault, config.StringParser)
	ServiceJsonKeysPort = config.LoadEnv(serviceJsonKeysPort, ServiceJsonKeysPortDefault, config.IntParser)

	SmtpAddr             = smtpAddr
	SmtpSenderName       = smtpSenderName
	SmtpSenderEmail      = smtpSenderEmail
	SmtpSenderPassword   = smtpSenderPassword
	SmtpSenderDomain     = smtpSenderDomain
	SmtpTimeout          = config.LoadEnv(smtpTimeout, SmtpTimeoutDefault, config.DurationParser)
	SmtpForceUnencrypted = config.LoadEnv(smtpForceUnencrypted, false, config.BoolParser)

	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	Otel    = config.LoadEnv(otel, false, config.BoolParser)

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

	GcloudProjectId    = gcloudProjectId
	SuperAdminEmail    = superAdminEmail
	SuperAdminPassword = superAdminPassword
)
