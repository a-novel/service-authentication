package models

import (
	_ "embed"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"
)

type ShortCodeUsageConfig struct {
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

type ShortCodesConfig struct {
	Size   int                                     `json:"size"   yaml:"size"`
	Usages map[ShortCodeUsage]ShortCodeUsageConfig `json:"usages" yaml:"usages"`
}

//go:embed short_codes.yaml
var defaultShortCodesFile []byte

var DefaultShortCodesConfig = config.MustUnmarshal[ShortCodesConfig](yaml.Unmarshal, defaultShortCodesFile)
