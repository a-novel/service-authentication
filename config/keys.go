package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/configurator"

	"github.com/a-novel/authentication/models"
)

//go:embed keys.yaml
var keysFile []byte

type Key struct {
	// How long a key for the target usage should be used, before being entirely discarded. To prevent rotation issues,
	// this value must be significantly larger than the Rotation value.
	TTL time.Duration `yaml:"ttl"`
	// Interval between key generations. To prevent key exhaustion, this value must be significantly smaller than the
	// TTL value.
	Rotation time.Duration `yaml:"rotation"`
}

type KeysType struct {
	Usages map[models.KeyUsage]Key `yaml:"usages"`
	Source struct {
		Cache struct {
			Duration time.Duration `yaml:"duration"`
		} `yaml:"cache"`
	} `yaml:"source"`
}

var Keys = configurator.NewLoader[KeysType](Loader).MustLoad(
	configurator.NewConfig("", keysFile),
)
