package config

import (
	_ "embed"

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
