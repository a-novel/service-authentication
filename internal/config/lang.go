package config

import "github.com/a-novel/service-authentication/v2/internal/config/auth"

const (
	// LangFR is the French language code.
	LangFR = auth.LangFR
	// LangEN is the English language code.
	LangEN = auth.LangEN
)

// KnownLangs lists the language codes supported by the service.
var KnownLangs = auth.KnownLangs
