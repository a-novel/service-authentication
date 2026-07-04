package config

import (
	_ "embed"
	"time"
)

// ShortCodeUsage holds the settings for a single short-code usage.
type ShortCodeUsage struct {
	// TTL is how long a short code issued for this usage stays valid.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

// ShortCodes configures the one-time codes emailed to users to authorize sensitive
// actions such as registration, email changes, and password resets.
type ShortCodes struct {
	// Size is the character length of a generated code.
	Size int `json:"size" yaml:"size"`
	// Usages holds the per-usage settings, keyed by usage name.
	Usages map[string]ShortCodeUsage `json:"usages" yaml:"usages"`
}
