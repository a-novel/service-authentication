package models

// ShortCodeUsage gives information about the intended usage of a short code.
type ShortCodeUsage string

func (usage ShortCodeUsage) String() string {
	return string(usage)
}

const (
	ShortCodeUsageValidateMail    ShortCodeUsage = "validateMail"
	ShortCodeUsageResetPassword   ShortCodeUsage = "resetPassword"
	ShortCodeUsageRequestRegister ShortCodeUsage = "requestRegister"
)
