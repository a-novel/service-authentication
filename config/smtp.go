package config

import (
	_ "embed"

	"github.com/a-novel-kit/configurator"
)

//go:embed smtp.yaml
var smtpFile []byte

//go:embed smtp.dev.yaml
var smtpDevFile []byte

type SMTPConfig struct {
	Sender struct {
		Name     string `yaml:"name"`
		Email    string `yaml:"email"`
		Password string `yaml:"password"`
		Domain   string `yaml:"domain"`
	} `yaml:"sender"`
	Addr    string `yaml:"addr"`
	Sandbox bool   `yaml:"sandbox"`
	URLs    struct {
		UpdateEmail    string `yaml:"updateEmail"`
		UpdatePassword string `yaml:"updatePassword"`
		Register       string `yaml:"register"`
	} `yaml:"urls"`
}

var SMTP = configurator.NewLoader[SMTPConfig](Loader).MustLoad(
	configurator.NewConfig("", smtpFile),
	configurator.NewConfig("local", smtpDevFile),
)
