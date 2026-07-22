package config

// SmtpUrls configures the web-client links embedded in outgoing emails.
//
// The send timeout lives on the sender, which is the only thing that reaches the SMTP connection.
// This struct is read by the short-code services for their link text and goes no further.
type SmtpUrls struct {
	UpdateEmail    string `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string `json:"updatePassword" yaml:"updatePassword"`
	Register       string `json:"register"       yaml:"register"`
}
