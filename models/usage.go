package models

// ShortCodeUsage gives information about the intended usage of a short code.
type ShortCodeUsage string

const (
	ShortCodeUsageValidateMail    ShortCodeUsage = "validateMail"
	ShortCodeUsageResetPassword   ShortCodeUsage = "resetPassword"
	ShortCodeUsageRequestRegister ShortCodeUsage = "requestRegister"
)
