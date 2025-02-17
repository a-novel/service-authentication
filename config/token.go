package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/configurator"

	"github.com/a-novel/authentication/models"
)

//go:embed token.yaml
var tokenFile []byte

type Token struct {
	TTL      time.Duration `yaml:"ttl"`
	Issuer   string        `yaml:"issuer"`
	Audience string        `yaml:"audience"`
	Subject  string        `yaml:"subject"`
	Leeway   time.Duration `yaml:"leeway"`
}

type TokenTypes struct {
	Usages map[models.KeyUsage]Token `yaml:"usages"`
}

var Tokens = configurator.NewLoader[TokenTypes](Loader).MustLoad(
	configurator.NewConfig("", tokenFile),
)
