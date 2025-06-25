package config

import (
	_ "embed"
	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"
	"net/http"
	"os"

	"github.com/a-novel-kit/configurator"
)

//go:embed sentry.yaml
var sentryFile []byte

type SentryType struct {
	DSN string `yaml:"dsn"`
}

var Sentry = configurator.NewLoader[SentryType](Loader).MustLoad(
	configurator.NewConfig("", sentryFile),
)

var SentryClient = sentry.ClientOptions{
	Dsn:              Sentry.DSN,
	EnableTracing:    true,
	EnableLogs:       true,
	TracesSampleRate: 1.0,
	Debug:            os.Getenv("DEBUG") == "true",
	ServerName:       lo.CoalesceOrEmpty(os.Getenv("SERVER_NAME"), "localhost"),
	Release:          lo.CoalesceOrEmpty(os.Getenv("RELEASE"), "local"),
	Environment:      lo.CoalesceOrEmpty(os.Getenv("ENV"), "development"),
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
