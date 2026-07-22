package config

// SmtpUrls configures the web-client links embedded in outgoing emails.
//
// The send timeout does not live here. It sat on this struct while nothing read it — the sender is
// what has to honour a timeout, and this struct never reaches the sender. A value on the wrong type
// looks configured, which is why the setting appeared plumbed for as long as it did.
type SmtpUrls struct {
	UpdateEmail    string `json:"updateEmail"    yaml:"updateEmail"`
	UpdatePassword string `json:"updatePassword" yaml:"updatePassword"`
	Register       string `json:"register"       yaml:"register"`
}
