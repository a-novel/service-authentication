package config

import (
	"time"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"
)

type Main struct {
	Name string `json:"name" yaml:"name"`
}

type Dependencies struct {
	JsonKeysServiceUrl string `json:"jsonKeysServiceURL" yaml:"jsonKeysServiceURL"`
}

type APITimeouts struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
}

type Cors struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

type API struct {
	Port           int         `json:"port"           yaml:"port"`
	Timeouts       APITimeouts `json:"timeouts"       yaml:"timeouts"`
	MaxRequestSize int64       `json:"maxRequestSize" yaml:"maxRequestSize"`
	Cors           Cors        `json:"cors"           yaml:"cors"`
}

type App[Otel otel.Config, Pg postgres.Config, SMTP smtp.Sender] struct {
	App Main `json:"app" yaml:"app"`
	Api API  `json:"api" yaml:"api"`

	DependenciesConfig Dependencies `json:"dependencies" yaml:"dependencies"`
	Permissions        Permissions  `json:"permissions"  yaml:"permissions"`
	ShortCodesConfig   ShortCodes   `json:"shortCodes"   yaml:"shortCodes"`
	SmtpUrlsConfig     SmtpUrls     `json:"smtpUrls"     yaml:"smtpUrls"`

	Smtp     SMTP `json:"smtp"     yaml:"smtp"`
	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}
