package config

import "time"

type SmtpUrls struct {
	UpdateEmail    string        `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string        `json:"updatePassword" yaml:"updatePassword"`
	Register       string        `json:"register"       yaml:"register"`
	Timeout        time.Duration `json:"timeout"        yaml:"timeout"`
}
