package models

// KeyUsage gives information about the intended usage of a key. Multiple keys with the same usage are grouped together
// when retrieved.
type KeyUsage string

const (
	// KeyUsageAuth is used to issue signed authentication tokens.
	KeyUsageAuth KeyUsage = "auth"
	// KeyUsageRefresh is used to issue signed refresh tokens.
	KeyUsageRefresh KeyUsage = "refresh"
)

var KnownKeyUsages = []KeyUsage{
	KeyUsageAuth,
	KeyUsageRefresh,
}

// ShortCodeUsage gives information about the intended usage of a short code.
type ShortCodeUsage string

const (
	ShortCodeUsageValidateMail    ShortCodeUsage = "validateMail"
	ShortCodeUsageResetPassword   ShortCodeUsage = "resetPassword"
	ShortCodeUsageRequestRegister ShortCodeUsage = "requestRegister"
)
