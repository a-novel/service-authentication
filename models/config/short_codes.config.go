package config

import (
	_ "embed"
	"time"

	"github.com/a-novel/service-authentication/models"
)

type ShortCodeUsage struct {
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

type ShortCodes struct {
	Size   int                                      `json:"size"   yaml:"size"`
	Usages map[models.ShortCodeUsage]ShortCodeUsage `json:"usages" yaml:"usages"`
}
