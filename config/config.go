package config

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/samber/lo"

	"github.com/a-novel-kit/configurator"
)

var Loader = configurator.LoaderConfig{
	Deserializer: yaml.Unmarshal,
	Env:          lo.CoalesceOrEmpty(os.Getenv("ENV"), "local"),
	ExpandEnv:    true,
}
