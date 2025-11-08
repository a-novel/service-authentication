package config

import (
	_ "embed"
	"time"
)

type ShortCodeUsage struct {
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

type ShortCodes struct {
	Size   int                       `json:"size"   yaml:"size"`
	Usages map[string]ShortCodeUsage `json:"usages" yaml:"usages"`
}
