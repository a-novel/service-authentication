package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"
)

// Main holds the core identity of the service, used to tag its logs and traces.
type Main struct {
	Name string `json:"name" yaml:"name"`
}

// Dependencies configures how the service reaches the backing services it calls.
type Dependencies struct {
	ServiceJsonKeysHost        string                    `json:"jsonKeysServiceHost" yaml:"jsonKeysServiceHost"`
	ServiceJsonKeysPort        int                       `json:"jsonKeysServicePort" yaml:"jsonKeysServicePort"`
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// RestTimeouts bounds the phases of the REST server's request lifecycle.
type RestTimeouts struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
	// Shutdown bounds the whole stop: the HTTP server's drain and the wait on detached work share it.
	Shutdown time.Duration `json:"shutdown" yaml:"shutdown"`
}

// Cors configures the cross-origin resource sharing policy of the REST server.
type Cors struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

// Rest configures the public REST HTTP server.
type Rest struct {
	Port           int          `json:"port"           yaml:"port"`
	Timeouts       RestTimeouts `json:"timeouts"       yaml:"timeouts"`
	MaxRequestSize int64        `json:"maxRequestSize" yaml:"maxRequestSize"`
	Cors           Cors         `json:"cors"           yaml:"cors"`
}

// App is the fully resolved configuration for the service, assembled at startup from the environment.
type App struct {
	App  Main `json:"app"  yaml:"app"`
	Rest Rest `json:"rest" yaml:"rest"`

	DependenciesConfig Dependencies `json:"dependencies" yaml:"dependencies"`
	Permissions        Permissions  `json:"permissions"  yaml:"permissions"`
	ShortCodesConfig   ShortCodes   `json:"shortCodes"   yaml:"shortCodes"`
	SmtpUrlsConfig     SmtpUrls     `json:"smtpUrls"     yaml:"smtpUrls"`

	Smtp       smtp.Sender        `json:"smtp"       yaml:"smtp"`
	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	HttpLogger logging.HTTPConfig `json:"httplogger" yaml:"httplogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
