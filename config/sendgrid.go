package config

import (
	_ "embed"

	"github.com/a-novel-kit/configurator"
)

//go:embed sendgrid.yaml
var sendgridFile []byte

//go:embed sendgrid.dev.yaml
var sendgridDevFile []byte

type SendgridType struct {
	APIKey  string `yaml:"apiKey"`
	Sandbox bool   `yaml:"sandbox"`
	Sender  struct {
		Mail string `yaml:"mail"`
		Name string `yaml:"name"`
	} `yaml:"sender"`
}

var Sendgrid = configurator.NewLoader[SendgridType](Loader).MustLoad(
	configurator.NewConfig("", sendgridFile),
	configurator.NewConfig("local", sendgridDevFile),
)
