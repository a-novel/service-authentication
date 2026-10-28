package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/sentryf"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// Main holds the core identity of the service, used to tag its logs and traces.
type Main struct {
	Name string `json:"name" yaml:"name"`
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time `json:"downtimeStart" yaml:"downtimeStart"`
}

// Dependencies configures how the service reaches the backing services it calls.
type Dependencies struct {
	ServiceJsonKeysHost        string                    `json:"jsonKeysServiceHost" yaml:"jsonKeysServiceHost"`
	ServiceJsonKeysPort        int                       `json:"jsonKeysServicePort" yaml:"jsonKeysServicePort"`
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// App is the fully resolved configuration for the service, assembled at startup from the environment.
type App struct {
	App    Main               `json:"app"    yaml:"app"`
	Rest   httpf.ServerConfig `json:"rest"   yaml:"rest"`
	Limits authconfig.Limits  `json:"limits" yaml:"limits"`

	DependenciesConfig Dependencies           `json:"dependencies" yaml:"dependencies"`
	Permissions        authconfig.Permissions `json:"permissions"  yaml:"permissions"`
	ShortCodesConfig   authconfig.ShortCodes  `json:"shortCodes"   yaml:"shortCodes"`
	SmtpUrlsConfig     authconfig.SmtpUrls    `json:"smtpUrls"     yaml:"smtpUrls"`
	Waitlist           authconfig.Waitlist    `json:"waitlist"     yaml:"waitlist"`

	Smtp       smtp.Sender        `json:"smtp"       yaml:"smtp"`
	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Sentry     sentryf.Config     `json:"sentry"     yaml:"sentry"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	HttpLogger logging.HTTPConfig `json:"httplogger" yaml:"httplogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
