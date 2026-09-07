package config

import "github.com/a-novel/service-authentication/v2/internal/config/auth"

// ShortCodeUsage holds the settings for a single short-code usage.
type ShortCodeUsage = auth.ShortCodeUsage

// ShortCodes configures the one-time codes that authorize account changes.
type ShortCodes = auth.ShortCodes
