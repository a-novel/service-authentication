package auth

import "github.com/a-novel-kit/golib/httpf"

// Limits caps what one caller, one mailbox or a burst can make the REST API spend. Rates are
// written "<requests>/<window>" in their variables.
type Limits struct {
	// Strict caps each caller of a route issuing anonymous tokens or invitation requests.
	Strict httpf.Rate `json:"strict" yaml:"strict"`
	// LoginBurst caps each caller's requests checking a password or short code, logins first, over a
	// short window.
	LoginBurst httpf.Rate `json:"loginBurst" yaml:"loginBurst"`
	// Login caps the same requests over a longer window, per caller and per email or account named.
	Login httpf.Rate `json:"login" yaml:"login"`
	// Mailbox caps the emails each address receives from one route.
	Mailbox httpf.Rate `json:"mailbox" yaml:"mailbox"`
	// CodeExchange caps each client's authorization code exchanges.
	CodeExchange httpf.Rate `json:"codeExchange" yaml:"codeExchange"`
	// WaitlistMaxConcurrent caps the waitlist requests waiting on Google at once, per instance.
	WaitlistMaxConcurrent int `json:"waitlistMaxConcurrent" yaml:"waitlistMaxConcurrent"`
	// WaitlistMaxRequestSize caps a waitlist request body, in bytes.
	WaitlistMaxRequestSize int64 `json:"waitlistMaxRequestSize" yaml:"waitlistMaxRequestSize"`
	// Argon2MaxConcurrent caps the password hashes running at once, per instance. It defaults to the
	// CPUs the container may use, since hashing is CPU-bound.
	Argon2MaxConcurrent int `json:"argon2MaxConcurrent" yaml:"argon2MaxConcurrent"`
}
