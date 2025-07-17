package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"
)

//go:embed short_codes.config.yaml
var defaultShortCodesFile []byte

var ShortCodesPresetDefault = config.MustUnmarshal[ShortCodes](yaml.Unmarshal, defaultShortCodesFile)
