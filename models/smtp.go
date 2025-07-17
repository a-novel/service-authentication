package models

type SMTPURLsConfig struct {
	UpdateEmail    string `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string `json:"updatePassword" yaml:"updatePassword"`
	Register       string `json:"register"       yaml:"register"`
}
