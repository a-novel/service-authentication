package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/configurator"
)

//go:embed api.yaml
var apiFile []byte

//go:embed api.dev.yaml
var apiDevFile []byte

type APIType struct {
	Port     int `yaml:"port"`
	Timeouts struct {
		Read       time.Duration `yaml:"read"`
		ReadHeader time.Duration `yaml:"readHeader"`
		Write      time.Duration `yaml:"write"`
		Idle       time.Duration `yaml:"idle"`
		Request    time.Duration `yaml:"request"`
	} `yaml:"timeouts"`
	Cors struct {
		AllowedOrigins   []string `yaml:"allowedOrigins"`
		AllowedHeaders   []string `yaml:"allowedHeaders"`
		AllowCredentials bool     `yaml:"allowCredentials"`
		MaxAge           int      `yaml:"maxAge"`
	} `yaml:"cors"`
}

var API = configurator.NewLoader[APIType](Loader).MustLoad(
	configurator.NewConfig("", apiFile),
	configurator.NewConfig("local", apiDevFile),
)
