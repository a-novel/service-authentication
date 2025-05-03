package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/configurator"

	"github.com/a-novel/service-authentication/models"
)

//go:embed short_codes.yaml
var shortCodeFile []byte

type ShortCodeUsageConfig struct {
	TTL time.Duration `yaml:"ttl"`
}

type ShortCodesType struct {
	Size   int                                            `yaml:"size"`
	Usages map[models.ShortCodeUsage]ShortCodeUsageConfig `yaml:"usages"`
}

var ShortCodes = configurator.NewLoader[ShortCodesType](Loader).MustLoad(
	configurator.NewConfig("", shortCodeFile),
)
