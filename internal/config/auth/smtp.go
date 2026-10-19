package auth

// SmtpUrls configures the web-client links embedded in outgoing emails.
type SmtpUrls struct {
	UpdateEmail    string `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string `json:"updatePassword" yaml:"updatePassword"`
	Register       string `json:"register"       yaml:"register"`
}
