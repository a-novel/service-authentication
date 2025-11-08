package env

import (
	"os"
	"time"
)

// EnvPrefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var EnvPrefix = os.Getenv("SERVICE_AUTHENTICATION_ENV")

func getEnv(name string) string {
	if EnvPrefix != "" {
		return os.Getenv(EnvPrefix + "_" + name)
	}

	return os.Getenv(name)
}

const (
	SmtpTimeoutDefault = 20 * time.Second

	AppNameDefault = "service-authentication"

	ApiPortDefault                 = 8080
	ApiTimeoutReadDefault          = 5 * time.Second
	ApiTimeoutReadHeaderDefault    = 3 * time.Second
	ApiTimeoutWriteDefault         = 10 * time.Second
	ApiTimeoutIdleDefault          = 30 * time.Second
	ApiTimeoutRequestDefault       = 15 * time.Second
	ApiMaxRequestSizeDefault       = 2 << 20 // 2 MiB
	ApiCorsAllowCredentialsDefault = false
	ApiCorsMaxAgeDefault           = 3600
)

var (
	ApiCorsAllowedOriginsDefault = []string{"*"}
	ApiCorsAllowedHeadersDefault = []string{"*"}
)

var (
	PostgresDsn     = getEnv("POSTGRES_DSN")
	PostgresDsnTest = getEnv("POSTGRES_DSN_TEST")

	AuthPlatformUrl               = getEnv("AUTH_PLATFORM_URL")
	AuthPlatformUpdateEmailUrl    = getEnv("AUTH_PLATFORM_URL_UPDATE_EMAIL")
	AuthPlatformUpdatePasswordUrl = getEnv("AUTH_PLATFORM_URL_UPDATE_PASSWORD")
	AuthPlatformRegisterUrl       = getEnv("AUTH_PLATFORM_URL_REGISTER")

	JsonKeysServiceUrl = getEnv("JSON_KEYS_SERVICE_URL")

	SmtpAddr           = getEnv("SMTP_ADDR")
	SmtpSenderName     = getEnv("SMTP_SENDER_NAME")
	SmtpSenderEmail    = getEnv("SMTP_SENDER_EMAIL")
	SmtpSenderPassword = getEnv("SMTP_SENDER_PASSWORD")
	SmtpSenderDomain   = getEnv("SMTP_SENDER_DOMAIN")
	SmtpTimeout        = getEnv("SMTP_TIMEOUT")

	AppName                 = getEnv("APP_NAME")
	ApiPort                 = getEnv("API_PORT")
	ApiMaxRequestSize       = getEnv("API_MAX_REQUEST_SIZE")
	ApiTimeoutRead          = getEnv("API_TIMEOUT_READ")
	ApiTimeoutReadHeader    = getEnv("API_TIMEOUT_READ_HEADER")
	ApiTimeoutWrite         = getEnv("API_TIMEOUT_WRITE")
	ApiTimeoutIdle          = getEnv("API_TIMEOUT_IDLE")
	ApiTimeoutRequest       = getEnv("API_TIMEOUT_REQUEST")
	ApiCorsAllowedOrigins   = getEnv("API_CORS_ALLOWED_ORIGINS")
	ApiCorsAllowedHeaders   = getEnv("API_CORS_ALLOWED_HEADERS")
	ApiCorsAllowCredentials = getEnv("API_CORS_ALLOW_CREDENTIALS")
	ApiCorsMaxAge           = getEnv("API_CORS_MAX_AGE")
	GcloudProjectId         = getEnv("GCLOUD_PROJECT_ID")

	PrettyConsole = getEnv("PRETTY_CONSOLE")
)
