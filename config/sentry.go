package config

import (
	_ "embed"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"

	"github.com/a-novel-kit/configurator"
)

//go:embed sentry.yaml
var sentryFile []byte

type SentryType struct {
	DSN         string `yaml:"dsn"`
	ServerName  string `yaml:"serverName"`
	Release     string `yaml:"release"`
	Environment string `yaml:"environment"`
	Debug       bool   `yaml:"debug"`
}

var Sentry = configurator.NewLoader[SentryType](Loader).MustLoad(
	configurator.NewConfig("", sentryFile),
)

var SentryClient = sentry.ClientOptions{
	Dsn:              Sentry.DSN,
	EnableTracing:    true,
	EnableLogs:       true,
	TracesSampleRate: 1.0,
	Debug:            Sentry.Debug,
	DebugWriter:      os.Stderr,
	ServerName:       lo.CoalesceOrEmpty(Sentry.ServerName, "localhost"),
	Release:          lo.CoalesceOrEmpty(Sentry.Release, "local"),
	Environment:      lo.CoalesceOrEmpty(Sentry.Environment, "development"),
	BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		if hint == nil || hint.Context == nil {
			return event
		}

		if req, ok := hint.Context.Value(sentry.RequestContextKey).(*http.Request); ok {
			// Add IP Address to user information.
			event.User.IPAddress = req.RemoteAddr
		}

		return event
	},
}
