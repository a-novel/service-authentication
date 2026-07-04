package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed short_codes.config.yaml
var defaultShortCodesFile []byte

// ShortCodesPresetDefault is the default short-code configuration, loaded from the
// embedded short_codes.config.yaml.
var ShortCodesPresetDefault = config.MustUnmarshal[ShortCodes](yaml.Unmarshal, defaultShortCodesFile)
