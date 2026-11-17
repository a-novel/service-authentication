package auth

// SMTPURLs configures the web-client links embedded in outgoing emails.
type SMTPURLs struct {
	UpdateEmail    string
	UpdatePassword string
	Register       string
}
