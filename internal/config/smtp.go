package config

import "time"

// SmtpUrls configures the web-client links embedded in outgoing emails and the timeout for sending them.
type SmtpUrls struct {
	UpdateEmail    string        `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string        `json:"updatePassword" yaml:"updatePassword"`
	Register       string        `json:"register"       yaml:"register"`
	Timeout        time.Duration `json:"timeout"        yaml:"timeout"`
}
